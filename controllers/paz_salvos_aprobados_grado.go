package controllers

import (
	"net/http"

	"github.com/udistrital/paz_y_salvos_mid/services"
)

// ListarAprobadosSecretaria expone al consumidor de Diplomas exclusivamente
// el estado actual aprobado de TPS_SECRETARIA, sin aceptar un perfil del cliente.
func (c *InscripcionGradoController) ListarAprobadosSecretaria() {
	periodo, errPeriodo := c.GetInt("periodo_id")
	limit, errLimit := c.GetInt("limit", 20)
	offset, errOffset := c.GetInt("offset", 0)
	if errPeriodo != nil || errLimit != nil || errOffset != nil || periodo <= 0 || limit <= 0 || limit > 100 || offset < 0 {
		c.respuesta(http.StatusBadRequest, nil, "Periodo y paginación válidos requeridos")
		return
	}
	resultado, err := services.ListarAprobadosSecretariaGrado(c.Ctx.Request.Context(), periodo, limit, offset)
	if err != nil {
		c.errorGrado(err)
		return
	}
	c.respuesta(http.StatusOK, resultado, "Paz y Salvos aprobados por Secretaría")
}
