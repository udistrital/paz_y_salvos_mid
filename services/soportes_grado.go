package services

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/udistrital/paz_y_salvos_mid/models"
	"github.com/udistrital/utils_oas/v2/request"
)

const MaxPDFGrado = 5 * 1024 * 1024

var tiposSoporteGrado = []struct {
	Codigo        string
	TipoDocumento string
}{
	{Codigo: "TSG_ACTA_SUST", TipoDocumento: "ACT"},
	{Codigo: "TSG_RESULTADO_SABER", TipoDocumento: "SEE"},
	{Codigo: "TSG_PAGO_DERECHOS", TipoDocumento: "CPDP"},
	{Codigo: "TSG_TITULO_PREVIO", TipoDocumento: "TAP"},
}

// El tipo de soporte PSGA no es el tipo_documento de Documento CRUD/Nuxeo.
func codigoDocumentoGrado(tipo string) (string, error) {
	for _, soporte := range tiposSoporteGrado {
		if soporte.Codigo == tipo {
			return soporte.TipoDocumento, nil
		}
	}
	return "", falloGrado(400, "Tipo de soporte de grado no admitido")
}

func validarPDFGrado(encoded, mime string) ([]byte, error) {
	if len(encoded) > base64.StdEncoding.EncodedLen(MaxPDFGrado) {
		return nil, falloGrado(413, "El PDF no debe superar 5 MiB")
	}
	contenido, err := base64.StdEncoding.Strict().DecodeString(encoded)
	if err != nil || len(contenido) == 0 {
		return nil, falloGrado(400, "Archivo base64 inválido o vacío")
	}
	if len(contenido) > MaxPDFGrado {
		return nil, falloGrado(413, "El PDF no debe superar 5 MiB")
	}
	if mime != "application/pdf" || http.DetectContentType(contenido) != "application/pdf" ||
		!bytes.HasPrefix(contenido, []byte("%PDF-")) || !bytes.HasSuffix(bytes.TrimSpace(contenido), []byte("%%EOF")) {
		return nil, falloGrado(400, "El soporte debe ser un archivo PDF válido")
	}
	return contenido, nil
}

func borradorParaSoportes(ctx context.Context, auth string, id, terceroID int, permitirRadicada bool) (*estudianteGrado, *models.BorradorGrado, int, error) {
	if id <= 0 || terceroID <= 0 {
		return nil, nil, 0, falloGrado(400, "Solicitud inválida")
	}
	user, err := resolverEstudiante(ctx, auth, terceroID)
	if err != nil {
		return nil, nil, 0, err
	}
	estado, err := resolverEstadoBorrador(user.Ctx)
	if err != nil {
		return nil, nil, 0, err
	}
	base, err := baseGrado("UrlCrudPazySalvos")
	if err != nil {
		return nil, nil, 0, err
	}
	q := url.Values{"tercero_id": {strconv.Itoa(user.TerceroID)}, "estado_borrador_id": {strconv.Itoa(estado)}}
	if permitirRadicada {
		estadoRadicada, err := resolverEstadoRadicada(user.Ctx)
		if err != nil {
			return nil, nil, 0, err
		}
		q.Set("estado_radicada_id", strconv.Itoa(estadoRadicada))
	}
	var resp models.APIResponseData[models.BorradorGrado]
	status, err := request.GetWithContext(user.Ctx, base+"solicitud-grado/borrador/"+strconv.Itoa(id)+"?"+q.Encode(), &resp)
	if err != nil {
		if status == 404 {
			return nil, nil, 0, falloGrado(404, "Borrador no encontrado")
		}
		if status == 409 {
			return nil, nil, 0, falloGrado(409, "La entrega ya no es un borrador editable")
		}
		return nil, nil, 0, falloGrado(503, "No se pudo consultar el borrador")
	}
	if !resp.Success || resp.Status != 200 || resp.Data.Solicitud.Id != id || resp.Data.Solicitud.TerceroId != user.TerceroID ||
		resp.Data.Formulario.Id <= 0 || (!permitirRadicada && resp.Data.Formulario.FechaRadicacion != nil) {
		return nil, nil, 0, falloGrado(503, "Respuesta de solicitud inválida")
	}
	codigo, err := resolverCodigoPrograma(user.Ctx, user.TerceroID, resp.Data.Solicitud.DependenciaOikosId)
	if err != nil {
		return nil, nil, 0, err
	}
	if resp.Data.Solicitud.CodigoEstudiante != codigo {
		return nil, nil, 0, falloGrado(409, "El código del borrador no corresponde al programa")
	}
	for _, s := range resp.Data.Soportes {
		if !s.Activo || s.Id <= 0 || s.DocumentoId <= 0 || s.SolicitudGradoId != id || s.FormularioSolicitudGradoId != resp.Data.Formulario.Id {
			return nil, nil, 0, falloGrado(503, "Soportes inconsistentes en el borrador")
		}
	}
	return user, &resp.Data, estado, nil
}

func validarVentanaSoporte(user *estudianteGrado, b *models.BorradorGrado) error {
	programa, err := resolverProgramaGrado(user.Ctx, user.TerceroID, b.Solicitud.ProgramaAcademicoId)
	if err != nil {
		return err
	}
	if programa.DependenciaId != b.Solicitud.DependenciaOikosId {
		return falloGrado(409, "Programa del borrador inconsistente")
	}
	_, apr, err := eventosGrado(user.Ctx, programa, b.Solicitud.PeriodoId)
	if err != nil {
		return err
	}
	return validarVentana(eventoGrado{}, apr, false)
}

type documentoGrado struct {
	Id            int    `json:"Id"`
	Nombre        string `json:"Nombre"`
	Enlace        string `json:"Enlace"`
	Activo        bool   `json:"Activo"`
	TipoDocumento struct {
		Id int `json:"Id"`
	} `json:"TipoDocumento"`
}

func consultarDocumentoGrado(ctx context.Context, id int) (*documentoGrado, error) {
	base, err := baseGrado("UrlDocumentos")
	if err != nil {
		return nil, err
	}
	var doc documentoGrado
	if _, err := request.GetWithContext(ctx, base+"documento/"+strconv.Itoa(id), &doc); err != nil || doc.Id != id || !doc.Activo || strings.TrimSpace(doc.Nombre) == "" ||
		doc.Enlace == "" || strings.ContainsAny(doc.Enlace, "/?#\\") || doc.Enlace == "." || doc.Enlace == ".." {
		return nil, falloGrado(503, "Documento no verificable en el gestor documental")
	}
	return &doc, nil
}

func resolverTipoDocumentoGrado(ctx context.Context, codigo string) (int, error) {
	base, err := baseGrado("UrlDocumentos")
	if err != nil {
		return 0, err
	}
	q := url.Values{"query": {"CodigoAbreviacion:" + codigo + ",Activo:true"}, "limit": {"100"}}
	var raw json.RawMessage
	if _, err := request.GetWithContext(ctx, base+"tipo_documento?"+q.Encode(), &raw); err != nil {
		return 0, falloGrado(503, "No se pudo resolver el tipo documental")
	}
	tipos, err := listaGrado[struct {
		Id                int
		CodigoAbreviacion string
		Activo            bool
	}](raw)
	if err != nil || len(tipos) != 1 || tipos[0].Id <= 0 || !tipos[0].Activo || tipos[0].CodigoAbreviacion != codigo {
		return 0, falloGrado(503, "Tipo documental ausente o ambiguo")
	}
	return tipos[0].Id, nil
}

func archivoDocumentoGrado(ctx context.Context, doc *documentoGrado) (*models.ArchivoSoporteGrado, []byte, error) {
	base, err := baseGrado("UrlGestorDocumental")
	if err != nil {
		return nil, nil, err
	}
	var archivo struct {
		File    string `json:"file"`
		Content struct {
			Mime string `json:"mime-type"`
		} `json:"file:content"`
	}
	if _, err := request.GetWithContext(ctx, base+"document/"+url.PathEscape(doc.Enlace), &archivo); err != nil {
		return nil, nil, falloGrado(503, "No se pudo recuperar el PDF de Nuxeo")
	}
	contenido, err := validarPDFGrado(archivo.File, archivo.Content.Mime)
	if err != nil {
		return nil, nil, falloGrado(503, "El documento de Nuxeo no es un PDF verificable")
	}
	return &models.ArchivoSoporteGrado{Nombre: doc.Nombre, MimeType: "application/pdf", Archivo: archivo.File}, contenido, nil
}

func ListarSoportesGrado(ctx context.Context, auth string, id, terceroID int) ([]models.SoporteBorradorGrado, error) {
	user, b, _, err := borradorParaSoportes(ctx, auth, id, terceroID, true)
	if err != nil {
		return nil, err
	}
	resultado := []models.SoporteBorradorGrado{}
	for _, definicion := range tiposSoporteGrado {
		tipo, err := resolverParametroGrado(user.Ctx, "TIP_SOP_GRADO", definicion.Codigo)
		if err != nil {
			return nil, err
		}
		for _, soporte := range b.Soportes {
			if soporte.TipoDocumentoId != tipo {
				continue
			}
			doc, err := consultarDocumentoGrado(user.Ctx, soporte.DocumentoId)
			if err != nil {
				return nil, err
			}
			resultado = append(resultado, models.SoporteBorradorGrado{Id: soporte.Id, FormularioId: b.Formulario.Id, TipoSoporte: definicion.Codigo, DocumentoId: doc.Id, Nombre: doc.Nombre})
		}
	}
	if len(resultado) != len(b.Soportes) || len(resultado) > len(tiposSoporteGrado) {
		return nil, falloGrado(503, "Tipos de soporte inconsistentes")
	}
	return resultado, nil
}

func DescargarSoporteGrado(ctx context.Context, auth string, id, terceroID int, codigo string) (*models.ArchivoSoporteGrado, error) {
	if _, err := codigoDocumentoGrado(codigo); err != nil {
		return nil, err
	}
	user, b, _, err := borradorParaSoportes(ctx, auth, id, terceroID, true)
	if err != nil {
		return nil, err
	}
	tipo, err := resolverParametroGrado(user.Ctx, "TIP_SOP_GRADO", codigo)
	if err != nil {
		return nil, err
	}
	for _, s := range b.Soportes {
		if s.TipoDocumentoId != tipo {
			continue
		}
		doc, err := consultarDocumentoGrado(user.Ctx, s.DocumentoId)
		if err != nil {
			return nil, err
		}
		archivo, _, err := archivoDocumentoGrado(user.Ctx, doc)
		return archivo, err
	}
	return nil, falloGrado(404, "No hay un soporte cargado de este tipo")
}

func CargarSoporteGrado(ctx context.Context, auth string, id int, codigo string, entrada models.CargarSoporteGrado) (*models.SoporteBorradorGrado, error) {
	codigoDoc, err := codigoDocumentoGrado(codigo)
	if err != nil {
		return nil, err
	}
	if entrada.FormularioId <= 0 || entrada.SoporteActualId < 0 || strings.TrimSpace(entrada.Nombre) == "" || len(entrada.Nombre) > 180 ||
		strings.ContainsAny(entrada.Nombre, "/\\\r\n\x00") || !strings.HasSuffix(strings.ToLower(entrada.Nombre), ".pdf") {
		return nil, falloGrado(400, "Versión o nombre de archivo inválido")
	}
	contenido, err := validarPDFGrado(entrada.Archivo, entrada.MimeType)
	if err != nil {
		return nil, err
	}
	user, b, estado, err := borradorParaSoportes(ctx, auth, id, entrada.TerceroId, false)
	if err != nil {
		return nil, err
	}
	if b.Formulario.Id != entrada.FormularioId {
		return nil, falloGrado(409, "La versión del borrador cambió; recarga sus soportes")
	}
	if err := validarVentanaSoporte(user, b); err != nil {
		return nil, err
	}
	tipo, err := resolverParametroGrado(user.Ctx, "TIP_SOP_GRADO", codigo)
	if err != nil {
		return nil, err
	}
	actual := 0
	for _, s := range b.Soportes {
		if s.TipoDocumentoId == tipo {
			actual = s.Id
		}
	}
	if actual != entrada.SoporteActualId {
		return nil, falloGrado(409, "El soporte cambió; recarga antes de reemplazarlo")
	}
	estadoSoporte, err := resolverParametroGrado(user.Ctx, "EST_SOP_GRADO", "SD_PEND_REV")
	if err != nil {
		return nil, err
	}
	tipoDocumento, err := resolverTipoDocumentoGrado(user.Ctx, codigoDoc)
	if err != nil {
		return nil, err
	}
	gestor, err := baseGrado("UrlGestorDocumental")
	if err != nil {
		return nil, err
	}
	// Contrato uploadAnyFormat usado por Inscripción: una carga produce {Status,res}.
	var subida struct {
		Status  json.RawMessage
		Success *bool
		Res     documentoGrado `json:"res"`
	}
	body := []map[string]interface{}{{"IdTipoDocumento": tipoDocumento, "nombre": entrada.Nombre, "file": entrada.Archivo,
		"descripcion": "Soporte provisional de inscripción a grado", "metadatos": map[string]interface{}{
			"solicitud_grado_id": id, "formulario_grado_id": entrada.FormularioId, "tipo_soporte": codigo, "tercero_id": user.TerceroID}}}
	if _, err := request.PostWithContext(user.Ctx, gestor+"document/uploadAnyFormat", body, &subida); err != nil ||
		len(subida.Status) == 0 || !statusExternoValido(subida.Status) || (subida.Success != nil && !*subida.Success) || subida.Res.Id <= 0 {
		return nil, falloGrado(503, "No se pudo confirmar la carga en Nuxeo; recarga los soportes antes de reintentar")
	}
	doc, err := consultarDocumentoGrado(user.Ctx, subida.Res.Id)
	if err != nil {
		return nil, err
	}
	_, persistido, err := archivoDocumentoGrado(user.Ctx, doc)
	if err != nil {
		return nil, err
	}
	if doc.TipoDocumento.Id != tipoDocumento || sha256.Sum256(persistido) != sha256.Sum256(contenido) {
		return nil, falloGrado(503, "El documento almacenado no corresponde al PDF enviado")
	}
	// La carga externa puede tardar: se revalida la ventana antes de asociar.
	if err := validarVentanaSoporte(user, b); err != nil {
		return nil, err
	}
	base, err := baseGrado("UrlCrudPazySalvos")
	if err != nil {
		return nil, err
	}
	q := url.Values{"tercero_id": {strconv.Itoa(user.TerceroID)}, "estado_borrador_id": {strconv.Itoa(estado)}}
	var resp models.APIResponseData[models.BorradorGrado]
	status, err := request.PutWithContext(user.Ctx, fmt.Sprintf("%ssolicitud-grado/borrador/%d/soportes/%d?%s", base, id, tipo, q.Encode()),
		map[string]int{"FormularioId": entrada.FormularioId, "SoporteActualId": entrada.SoporteActualId, "DocumentoId": doc.Id, "EstadoSoporteId": estadoSoporte}, &resp)
	if err != nil {
		if status == 409 {
			return nil, falloGrado(409, "El borrador o soporte cambió durante la carga. Recarga antes de reintentar")
		}
		return nil, falloGrado(503, "El PDF fue cargado pero no se confirmó su asociación. Recarga los soportes")
	}
	if !resp.Success || resp.Status != 200 || resp.Data.Solicitud.Id != id || resp.Data.Solicitud.TerceroId != user.TerceroID || resp.Data.Formulario.Id != entrada.FormularioId {
		return nil, falloGrado(503, "Asociación de soporte no verificable")
	}
	for _, s := range resp.Data.Soportes {
		if s.Id > 0 && s.Activo && s.DocumentoId == doc.Id && s.TipoDocumentoId == tipo && s.SolicitudGradoId == id && s.FormularioSolicitudGradoId == entrada.FormularioId {
			return &models.SoporteBorradorGrado{Id: s.Id, FormularioId: entrada.FormularioId, TipoSoporte: codigo, DocumentoId: doc.Id, Nombre: doc.Nombre}, nil
		}
	}
	return nil, falloGrado(503, "No se confirmó el soporte en el borrador")
}
