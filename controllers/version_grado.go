package controllers

import (
	"net/http"
	"strconv"

	"github.com/udistrital/paz_y_salvos_mid/services"
)

func (c *InscripcionGradoController) parametrosVersion() (int, int, bool) {
	id, err := strconv.Atoi(c.Ctx.Input.Param(":id"))
	if err != nil || id <= 0 {
		c.respuesta(http.StatusBadRequest, nil, "Solicitud inválida")
		return 0, 0, false
	}
	version, err := strconv.Atoi(c.Ctx.Input.Param(":numero"))
	if err != nil || version <= 0 {
		c.respuesta(http.StatusBadRequest, nil, "Versión inválida")
		return 0, 0, false
	}
	return id, version, true
}

func (c *InscripcionGradoController) listarVersiones(secretaria bool) {
	id, err := strconv.Atoi(c.Ctx.Input.Param(":id"))
	if err != nil || id <= 0 {
		c.respuesta(http.StatusBadRequest, nil, "Solicitud inválida")
		return
	}
	resultado, err := services.ListarVersionesGrado(c.Ctx.Request.Context(), c.Ctx.Input.Header("Authorization"), id, secretaria)
	if err != nil {
		c.errorGrado(err)
		return
	}
	c.respuesta(http.StatusOK, resultado, "Entregas radicadas consultadas")
}

func (c *InscripcionGradoController) consultarVersion(secretaria bool) {
	id, version, ok := c.parametrosVersion()
	if !ok {
		return
	}
	resultado, err := services.ConsultarVersionGrado(c.Ctx.Request.Context(), c.Ctx.Input.Header("Authorization"), id, version, secretaria)
	if err != nil {
		c.errorGrado(err)
		return
	}
	c.respuesta(http.StatusOK, resultado, "Entrega histórica consultada")
}

func (c *InscripcionGradoController) descargarSoporteVersion(secretaria bool) {
	id, version, ok := c.parametrosVersion()
	if !ok {
		return
	}
	resultado, err := services.DescargarSoporteVersionGrado(c.Ctx.Request.Context(), c.Ctx.Input.Header("Authorization"), id, version, c.Ctx.Input.Param(":tipo"), secretaria)
	if err != nil {
		c.errorGrado(err)
		return
	}
	c.respuesta(http.StatusOK, resultado, "PDF de entrega histórica consultado")
}

func (c *InscripcionGradoController) ListarVersionesEstudiante()  { c.listarVersiones(false) }
func (c *InscripcionGradoController) ConsultarVersionEstudiante() { c.consultarVersion(false) }
func (c *InscripcionGradoController) DescargarVersionEstudiante() { c.descargarSoporteVersion(false) }
func (c *InscripcionGradoController) ListarVersionesSecretaria()  { c.listarVersiones(true) }
func (c *InscripcionGradoController) ConsultarVersionSecretaria() { c.consultarVersion(true) }
func (c *InscripcionGradoController) DescargarVersionSecretaria() { c.descargarSoporteVersion(true) }
