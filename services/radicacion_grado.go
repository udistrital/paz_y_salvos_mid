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

type camposRadicacionGrado struct {
	Director1         string
	Director2         string
	Modalidad         string
	LugarExpedicionID int
}

func validarContenidoRadicacionGrado(raw json.RawMessage) (json.RawMessage, camposRadicacionGrado, error) {
	if !contenidoGradoValido(raw) {
		return nil, camposRadicacionGrado{}, falloGrado(400, "Contenido del formulario inválido")
	}
	var formulario struct {
		TrabajoGrado               string `json:"trabajoGrado"`
		Director1                  string `json:"director1"`
		Director2                  string `json:"director2"`
		Modalidad                  string `json:"modalidad"`
		LugarExpedicionDocumentoId int    `json:"lugarExpedicionDocumentoId"`
		NumeroActaSustentacion     string `json:"numeroActaSustentacion"`
		NumeroRegistroSNP          string `json:"numeroRegistroSnp"`
		TrabajaActualmente         *bool  `json:"trabajaActualmente"`
		Empresa                    string `json:"empresa"`
		DireccionEmpresa           string `json:"direccionEmpresa"`
		TelefonoEmpresa            string `json:"telefonoEmpresa"`
	}
	var contenido map[string]interface{}
	if json.Unmarshal(raw, &formulario) != nil || json.Unmarshal(raw, &contenido) != nil {
		return nil, camposRadicacionGrado{}, falloGrado(400, "Contenido del formulario inválido")
	}
	formulario.TrabajoGrado = strings.TrimSpace(formulario.TrabajoGrado)
	formulario.Director1 = strings.TrimSpace(formulario.Director1)
	formulario.Director2 = strings.TrimSpace(formulario.Director2)
	formulario.Modalidad = strings.TrimSpace(formulario.Modalidad)
	formulario.NumeroActaSustentacion = strings.TrimSpace(formulario.NumeroActaSustentacion)
	formulario.NumeroRegistroSNP = strings.ToUpper(strings.TrimSpace(formulario.NumeroRegistroSNP))
	if formulario.TrabajoGrado == "" || !cedulaGrado.MatchString(formulario.Director1) || formulario.Modalidad == "" ||
		formulario.LugarExpedicionDocumentoId <= 0 || formulario.NumeroActaSustentacion == "" ||
		!registroSNPGrado.MatchString(formulario.NumeroRegistroSNP) || formulario.TrabajaActualmente == nil {
		return nil, camposRadicacionGrado{}, falloGrado(400, "Completa todos los campos obligatorios antes de radicar")
	}
	if formulario.Director2 != "" && (!cedulaGrado.MatchString(formulario.Director2) || formulario.Director2 == formulario.Director1) {
		return nil, camposRadicacionGrado{}, falloGrado(400, "Los directores seleccionados no son válidos o están repetidos")
	}
	formulario.Empresa = strings.TrimSpace(formulario.Empresa)
	formulario.DireccionEmpresa = strings.TrimSpace(formulario.DireccionEmpresa)
	formulario.TelefonoEmpresa = strings.TrimSpace(formulario.TelefonoEmpresa)
	if *formulario.TrabajaActualmente && (formulario.Empresa == "" || formulario.DireccionEmpresa == "" || formulario.TelefonoEmpresa == "") {
		return nil, camposRadicacionGrado{}, falloGrado(400, "Completa los datos laborales antes de radicar")
	}
	for clave, valor := range map[string]interface{}{
		"trabajoGrado": formulario.TrabajoGrado, "director1": formulario.Director1, "modalidad": formulario.Modalidad,
		"lugarExpedicionDocumentoId": formulario.LugarExpedicionDocumentoId, "numeroActaSustentacion": formulario.NumeroActaSustentacion,
		"numeroRegistroSnp": formulario.NumeroRegistroSNP, "trabajaActualmente": *formulario.TrabajaActualmente,
	} {
		contenido[clave] = valor
	}
	delete(contenido, "lugarExpedicionDocumento")
	delete(contenido, "departamentoExpedicionDocumentoId")
	delete(contenido, "departamentoExpedicionDocumento")
	delete(contenido, "paisExpedicionDocumentoId")
	delete(contenido, "paisExpedicionDocumento")
	if formulario.Director2 == "" {
		delete(contenido, "director2")
	} else {
		contenido["director2"] = formulario.Director2
	}
	if *formulario.TrabajaActualmente {
		contenido["empresa"], contenido["direccionEmpresa"], contenido["telefonoEmpresa"] = formulario.Empresa, formulario.DireccionEmpresa, formulario.TelefonoEmpresa
	} else {
		delete(contenido, "empresa")
		delete(contenido, "direccionEmpresa")
		delete(contenido, "telefonoEmpresa")
	}
	normalizado, err := json.Marshal(contenido)
	if err != nil {
		return nil, camposRadicacionGrado{}, falloGrado(400, "Contenido del formulario inválido")
	}
	return normalizado, camposRadicacionGrado{Director1: formulario.Director1, Director2: formulario.Director2, Modalidad: formulario.Modalidad, LugarExpedicionID: formulario.LugarExpedicionDocumentoId}, nil
}

func agregarLugarExpedicionGrado(raw json.RawMessage, lugar models.LugarExpedicionGrado) (json.RawMessage, error) {
	var contenido map[string]interface{}
	if json.Unmarshal(raw, &contenido) != nil {
		return nil, falloGrado(http.StatusBadRequest, "Contenido del formulario inválido")
	}
	contenido["lugarExpedicionDocumentoId"] = lugar.Id
	contenido["lugarExpedicionDocumento"] = lugar.Nombre
	normalizado, err := json.Marshal(contenido)
	if err != nil {
		return nil, falloGrado(http.StatusBadRequest, "Contenido del formulario inválido")
	}
	return normalizado, nil
}

func RadicarGrado(ctx context.Context, auth string, id int, entrada models.RadicarGrado) (*models.BorradorGrado, error) {
	if id <= 0 || entrada.TerceroId <= 0 || entrada.FormularioId <= 0 {
		return nil, falloGrado(400, "Solicitud o versión inválida")
	}
	contenido, campos, err := validarContenidoRadicacionGrado(entrada.Contenido)
	if err != nil {
		return nil, err
	}
	user, borrador, estadoBorrador, err := borradorParaSoportes(ctx, auth, id, entrada.TerceroId, false)
	if err != nil {
		return nil, err
	}
	if borrador.Formulario.Id != entrada.FormularioId {
		return nil, falloGrado(409, "La versión cambió; recarga antes de radicar")
	}
	programa, err := resolverProgramaGrado(user.Ctx, user.TerceroID, borrador.Solicitud.ProgramaAcademicoId)
	if err != nil {
		return nil, err
	}
	inscripcion, aprobacion, err := eventosGrado(user.Ctx, programa, borrador.Solicitud.PeriodoId)
	if err != nil {
		return nil, err
	}
	if inscripcion.EventoId != borrador.Solicitud.CalendarioEventoInscripcionId || aprobacion.EventoId != borrador.Solicitud.CalendarioEventoAprobacionId {
		return nil, falloGrado(409, "La configuración de fechas cambió; la solicitud requiere revisión")
	}
	if err := validarVentana(inscripcion, aprobacion, true); err != nil {
		return nil, err
	}
	if err := ValidarDirectorGrado(ctx, auth, entrada.TerceroId, campos.Director1); err != nil {
		return nil, err
	}
	if campos.Director2 != "" {
		if err := ValidarDirectorGrado(ctx, auth, entrada.TerceroId, campos.Director2); err != nil {
			return nil, err
		}
	}
	if err := ValidarModalidadGrado(ctx, auth, entrada.TerceroId, campos.Modalidad); err != nil {
		return nil, err
	}
	lugar, err := ValidarLugarExpedicionGrado(ctx, auth, entrada.TerceroId, campos.LugarExpedicionID)
	if err != nil {
		return nil, err
	}
	contenido, err = agregarLugarExpedicionGrado(contenido, *lugar)
	if err != nil {
		return nil, err
	}
	tipos := make([]int, 0, len(tiposSoporteGrado))
	documentos := make(map[int]bool, len(tiposSoporteGrado))
	for _, definicion := range tiposSoporteGrado {
		tipo, err := resolverParametroGrado(user.Ctx, "TIP_SOP_GRADO", definicion.Codigo)
		if err != nil {
			return nil, err
		}
		tipos = append(tipos, tipo)
		var soporte *models.SoporteGrado
		for i := range borrador.Soportes {
			if borrador.Soportes[i].TipoDocumentoId == tipo {
				if soporte != nil {
					return nil, falloGrado(409, "Hay soportes duplicados en la versión")
				}
				soporte = &borrador.Soportes[i]
			}
		}
		if soporte == nil || documentos[soporte.DocumentoId] {
			return nil, falloGrado(400, "Carga los cuatro soportes obligatorios antes de radicar")
		}
		documentos[soporte.DocumentoId] = true
		tipoDocumento, err := resolverTipoDocumentoGrado(user.Ctx, definicion.TipoDocumento)
		if err != nil {
			return nil, err
		}
		documento, err := consultarDocumentoGrado(user.Ctx, soporte.DocumentoId)
		if err != nil || documento.TipoDocumento.Id != tipoDocumento {
			return nil, falloGrado(503, "Uno de los soportes no corresponde al tipo documental esperado")
		}
		if _, _, err := archivoDocumentoGrado(user.Ctx, documento); err != nil {
			return nil, err
		}
	}
	if len(borrador.Soportes) != len(tiposSoporteGrado) {
		return nil, falloGrado(400, "Carga los cuatro soportes obligatorios antes de radicar")
	}
	estadoRadicada, err := resolverParametroGrado(user.Ctx, "EST_SOL_GRADO", "SG_RADICADA")
	if err != nil {
		return nil, err
	}
	estadoSoporte, err := resolverParametroGrado(user.Ctx, "EST_SOP_GRADO", "SD_PEND_REV")
	if err != nil {
		return nil, err
	}
	base, err := baseGrado("UrlCrudPazySalvos")
	if err != nil {
		return nil, err
	}
	q := url.Values{"tercero_id": {strconv.Itoa(user.TerceroID)}}
	body := map[string]interface{}{"FormularioId": entrada.FormularioId, "Contenido": json.RawMessage(contenido), "EstadoBorradorId": estadoBorrador,
		"EstadoRadicadaId": estadoRadicada, "EstadoSoportePendienteId": estadoSoporte, "TiposSoporteId": tipos}
	var resp models.APIResponseData[models.BorradorGrado]
	status, err := request.PostWithContext(user.Ctx, base+"solicitud-grado/borrador/"+strconv.Itoa(id)+"/radicar?"+q.Encode(), body, &resp)
	if err != nil {
		if status == http.StatusConflict {
			return nil, falloGrado(409, "La versión ya cambió o fue radicada")
		}
		if status == http.StatusBadRequest {
			return nil, falloGrado(400, "El formulario o sus soportes no cumplen los requisitos de radicación")
		}
		return nil, falloGrado(503, "No se pudo confirmar la radicación; consulta el estado antes de reintentar")
	}
	if !resp.Success || resp.Status != 200 || resp.Data.Solicitud.Id != id || resp.Data.Solicitud.TerceroId != user.TerceroID ||
		resp.Data.Formulario.Id != entrada.FormularioId || resp.Data.Formulario.FechaRadicacion == nil || len(resp.Data.Soportes) != 4 {
		return nil, falloGrado(503, "La respuesta de radicación no es verificable; consulta el estado antes de reintentar")
	}
	return &resp.Data, nil
}
