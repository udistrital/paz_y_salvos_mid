package services

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/udistrital/paz_y_salvos_mid/models"
	"github.com/udistrital/utils_oas/v2/request"
)

// ListarAprobadosSecretariaGrado consulta únicamente el último estado del check
// TPS_SECRETARIA. Es una consulta pública global por periodo; el cliente no
// puede indicar perfil, tipo ni estado del check.
func ListarAprobadosSecretariaGrado(ctx context.Context, periodoID, limit, offset int) (*models.PaginaAprobadosSecretariaGrado, error) {
	if periodoID <= 0 || limit <= 0 || limit > 100 || offset < 0 {
		return nil, falloGrado(http.StatusBadRequest, "Periodo y paginación válidos requeridos")
	}
	tipoSecretaria, err := resolverParametroGrado(ctx, "TIP_PAZ_SALVO", "TPS_SECRETARIA")
	if err != nil {
		return nil, err
	}
	estadoAprobado, err := resolverParametroGrado(ctx, "EST_PAZ_SALVO", "PS_APROBADO")
	if err != nil {
		return nil, err
	}
	estadoDocumental, err := resolverParametroGrado(ctx, "EST_SOL_GRADO", "SG_DOC_APROBADA")
	if err != nil {
		return nil, err
	}
	base, err := baseGrado("UrlCrudPazySalvos")
	if err != nil {
		return nil, err
	}
	q := url.Values{
		"periodo_id":                       {strconv.Itoa(periodoID)},
		"limit":                            {strconv.Itoa(limit)},
		"offset":                           {strconv.Itoa(offset)},
		"estado_documentacion_aprobada_id": {strconv.Itoa(estadoDocumental)},
		"tipo_secretaria_id":               {strconv.Itoa(tipoSecretaria)},
		"estado_aprobado_id":               {strconv.Itoa(estadoAprobado)},
	}
	var respuesta models.APIResponseData[models.PaginaAprobadosSecretariaGrado]
	status, err := request.GetWithContext(ctx, base+"solicitud-grado/paz-salvos/aprobados?"+q.Encode(), &respuesta)
	if err != nil || status != http.StatusOK || !respuesta.Success || respuesta.Status != http.StatusOK ||
		respuesta.Data.Total < len(respuesta.Data.Solicitudes) || respuesta.Data.Solicitudes == nil {
		return nil, falloGrado(http.StatusServiceUnavailable, "Respuesta de aprobados de Secretaría inválida")
	}
	for _, solicitud := range respuesta.Data.Solicitudes {
		if solicitud.SolicitudId <= 0 || solicitud.TerceroId <= 0 || solicitud.PeriodoId != periodoID ||
			solicitud.ProgramaAcademicoId <= 0 || solicitud.DependenciaOikosId <= 0 || solicitud.FechaAprobacion.IsZero() {
			return nil, falloGrado(http.StatusServiceUnavailable, "El CRUD devolvió una aprobación inconsistente")
		}
	}
	return &respuesta.Data, nil
}
