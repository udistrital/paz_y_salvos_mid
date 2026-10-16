package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/udistrital/paz_y_salvos_mid/models"
	"github.com/udistrital/utils_oas/v2/request"
)

type ambitoVersionGrado struct {
	ctx          context.Context
	tercero      int
	dependencias []int
}

func resolverAmbitoVersionGrado(ctx context.Context, auth string, secretaria bool) (*ambitoVersionGrado, error) {
	if secretaria {
		actor, err := resolverSecretariaGrado(ctx, auth)
		if err != nil {
			return nil, err
		}
		return &ambitoVersionGrado{ctx: actor.Ctx, dependencias: actor.Dependencias}, nil
	}
	actor, err := resolverIdentidadPazSalvoGrado(ctx, auth)
	if err != nil {
		return nil, err
	}
	return &ambitoVersionGrado{ctx: actor.Ctx, tercero: actor.TerceroID}, nil
}

// El CRUD limita cada lectura a un único ámbito verificado por el MID.
func consultarVersionesCRUD[T any](ambito *ambitoVersionGrado, id int, sufijo string) (T, error) {
	var vacio T
	base, err := baseGrado("UrlCrudPazySalvos")
	if err != nil {
		return vacio, err
	}
	dependencias := ambito.dependencias
	if ambito.tercero > 0 {
		dependencias = []int{0}
	}
	for _, dependencia := range dependencias {
		query := url.Values{}
		if ambito.tercero > 0 {
			query.Set("tercero_id", strconv.Itoa(ambito.tercero))
		} else {
			query.Set("dependencia_id", strconv.Itoa(dependencia))
		}
		var respuesta models.APIResponseData[T]
		status, err := request.GetWithContext(ambito.ctx, base+"solicitud-grado/"+strconv.Itoa(id)+"/versiones"+sufijo+"?"+query.Encode(), &respuesta)
		if status == http.StatusNotFound {
			continue
		}
		if err != nil || status != http.StatusOK || !respuesta.Success || respuesta.Status != http.StatusOK {
			return vacio, falloGrado(http.StatusServiceUnavailable, "No fue posible consultar las entregas de la solicitud")
		}
		return respuesta.Data, nil
	}
	return vacio, falloGrado(http.StatusNotFound, "Entrega no encontrada en el ámbito autorizado")
}

func ListarVersionesGrado(ctx context.Context, auth string, id int, secretaria bool) ([]models.VersionGradoResumen, error) {
	if id <= 0 {
		return nil, falloGrado(http.StatusBadRequest, "Solicitud inválida")
	}
	ambito, err := resolverAmbitoVersionGrado(ctx, auth, secretaria)
	if err != nil {
		return nil, err
	}
	versiones, err := consultarVersionesCRUD[[]models.VersionGradoResumen](ambito, id, "")
	if err != nil {
		return nil, err
	}
	estados, err := estadosSolicitudRevisionGrado(ambito.ctx)
	if err != nil {
		return nil, err
	}
	for i := range versiones {
		version := &versiones[i]
		if version.FormularioId <= 0 || version.Version <= 0 || version.FechaRadicacion == "" {
			return nil, falloGrado(http.StatusServiceUnavailable, "Lista de entregas inconsistente")
		}
		for codigo, id := range estados {
			if version.EstadoSolicitudId == id {
				version.EstadoCodigo = codigo
			}
		}
		if version.EstadoCodigo == "" || version.EstadoCodigo == "SG_BORRADOR" {
			return nil, falloGrado(http.StatusServiceUnavailable, "Estado histórico no verificable")
		}
	}
	return versiones, nil
}

func consultarVersionGrado(ctx context.Context, auth string, id, version int, secretaria bool) (*models.VersionGradoDetalle, context.Context, error) {
	if id <= 0 || version <= 0 {
		return nil, nil, falloGrado(http.StatusBadRequest, "Solicitud o versión inválida")
	}
	ambito, err := resolverAmbitoVersionGrado(ctx, auth, secretaria)
	if err != nil {
		return nil, nil, err
	}
	detalle, err := consultarVersionesCRUD[models.VersionGradoDetalle](ambito, id, "/"+strconv.Itoa(version))
	if err != nil {
		return nil, nil, err
	}
	if detalle.Solicitud.Id != id || detalle.Formulario.Version != version || detalle.Formulario.Id <= 0 ||
		detalle.Formulario.FechaRadicacion == nil || len(detalle.Historial) == 0 ||
		(ambito.tercero > 0 && detalle.Solicitud.TerceroId != ambito.tercero) {
		return nil, nil, falloGrado(http.StatusServiceUnavailable, "Entrega histórica inconsistente")
	}
	estados, err := estadosSolicitudRevisionGrado(ambito.ctx)
	if err != nil {
		return nil, nil, err
	}
	for i := range detalle.Historial {
		item := &detalle.Historial[i]
		if item.FormularioId == nil || *item.FormularioId != detalle.Formulario.Id {
			return nil, nil, falloGrado(http.StatusServiceUnavailable, "Historial de entrega inconsistente")
		}
		for codigo, id := range estados {
			if item.EstadoSolicitudId == id {
				item.EstadoCodigo = codigo
			}
		}
		if item.EstadoCodigo == "" {
			return nil, nil, falloGrado(http.StatusServiceUnavailable, "Estado de entrega no verificable")
		}
	}
	detalle.Estado = detalle.Historial[len(detalle.Historial)-1].EstadoCodigo
	if detalle.Estado == "SG_BORRADOR" {
		return nil, nil, falloGrado(http.StatusServiceUnavailable, "Entrega histórica no radicada")
	}
	tipos := make(map[int]string)
	for _, definicion := range tiposSoporteGrado {
		tipoID, err := resolverParametroGrado(ambito.ctx, "TIP_SOP_GRADO", definicion.Codigo)
		if err != nil {
			return nil, nil, err
		}
		tipos[tipoID] = definicion.Codigo
	}
	estadosSoporte := make(map[int]string)
	for _, codigo := range []string{"SD_PEND_REV", "SD_OBSERVADO", "SD_APROBADO"} {
		estadoID, err := resolverParametroGrado(ambito.ctx, "EST_SOP_GRADO", codigo)
		if err != nil {
			return nil, nil, err
		}
		estadosSoporte[estadoID] = codigo
	}
	vistos := make(map[int]bool)
	for i := range detalle.Soportes {
		item := &detalle.Soportes[i]
		if item.Soporte.Id <= 0 || item.Soporte.SolicitudGradoId != id || item.Soporte.FormularioSolicitudGradoId != detalle.Formulario.Id ||
			item.Soporte.DocumentoId <= 0 || tipos[item.Soporte.TipoDocumentoId] == "" || vistos[item.Soporte.TipoDocumentoId] || len(item.Historial) == 0 {
			return nil, nil, falloGrado(http.StatusServiceUnavailable, "Soporte de entrega inconsistente")
		}
		vistos[item.Soporte.TipoDocumentoId] = true
		item.TipoSoporte = tipos[item.Soporte.TipoDocumentoId]
		for j := range item.Historial {
			actuacion := &item.Historial[j]
			if actuacion.SoporteGradoId != item.Soporte.Id || estadosSoporte[actuacion.EstadoSoporteId] == "" {
				return nil, nil, falloGrado(http.StatusServiceUnavailable, "Historial de soporte inconsistente")
			}
			actuacion.EstadoCodigo = estadosSoporte[actuacion.EstadoSoporteId]
		}
		item.EstadoActual = item.Historial[len(item.Historial)-1]
	}
	// Las referencias de modalidad y directores complementan la entrega cuando
	// están disponibles; su ausencia no impide consultar el contenido guardado.
	var formulario contenidoRevisionGrado
	if json.Unmarshal(detalle.Formulario.Contenido, &formulario) == nil {
		if nombre, err := resolverModalidadRevisionGrado(ambito.ctx, strings.TrimSpace(formulario.Modalidad)); err == nil {
			detalle.Referencias.Modalidad = nombre
		}
		if nombre, err := resolverDirectorRevisionGrado(ambito.ctx, strings.TrimSpace(formulario.Director1)); err == nil {
			detalle.Referencias.Director1 = nombre
		}
		if nombre, err := resolverDirectorRevisionGrado(ambito.ctx, strings.TrimSpace(formulario.Director2)); err == nil {
			detalle.Referencias.Director2 = nombre
		}
	}
	return &detalle, ambito.ctx, nil
}

func ConsultarVersionGrado(ctx context.Context, auth string, id, version int, secretaria bool) (*models.VersionGradoDetalle, error) {
	detalle, _, err := consultarVersionGrado(ctx, auth, id, version, secretaria)
	return detalle, err
}

func DescargarSoporteVersionGrado(ctx context.Context, auth string, id, version int, codigo string, secretaria bool) (*models.ArchivoSoporteGrado, error) {
	codigoDoc, err := codigoDocumentoGrado(codigo)
	if err != nil {
		return nil, err
	}
	detalle, contexto, err := consultarVersionGrado(ctx, auth, id, version, secretaria)
	if err != nil {
		return nil, err
	}
	for _, soporte := range detalle.Soportes {
		if soporte.TipoSoporte != codigo {
			continue
		}
		documento, err := consultarDocumentoGrado(contexto, soporte.Soporte.DocumentoId)
		if err != nil {
			return nil, err
		}
		tipoDocumento, err := resolverTipoDocumentoGrado(contexto, codigoDoc)
		if err != nil || documento.TipoDocumento.Id != tipoDocumento {
			return nil, falloGrado(http.StatusServiceUnavailable, "Tipo documental histórico no verificable")
		}
		archivo, _, err := archivoDocumentoGrado(contexto, documento)
		return archivo, err
	}
	return nil, falloGrado(http.StatusNotFound, "El soporte no pertenece a esta entrega")
}
