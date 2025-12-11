package controllers

import (
	"github.com/astaxie/beego"
	"github.com/udistrital/paz_y_salvos_mid/helpers"
	"github.com/udistrital/paz_y_salvos_mid/services"
	"github.com/udistrital/utils_oas/errorhandler"
)

// SemaforoController operations for Semaforo
type SemaforoController struct {
	beego.Controller
}

// URLMapping ...
func (c *SemaforoController) URLMapping() {
	c.Mapping("ObtenerEstudiante", c.ObtenerEstudiante)
	c.Mapping("ObtenerEstudiantes", c.ObtenerEstudiantes)
	c.Mapping("ObtenerEstudiantesProyecto", c.ObtenerEstudiantesProyecto)
	c.Mapping("ObtenerEstudiantesFacultad", c.ObtenerEstudiantesFacultad)
	c.Mapping("ObtenerEstudiantesFacultadLaboratorios", c.ObtenerEstudiantesFacultadLaboratorios)
}

// ObtenerEstudiante ...
// @Title ObtenerEstudiante
// @Description Obtiene información del semáforo de un estudiante por código
// @Param	codigo	path	int	true	"Código del estudiante"
// @Param	limit	query	int	false	"Número de registros por página"
// @Param	offset	query	int	false	"Número de registros a saltar"
// @Success 200 {array} models.SemaforoTable
// @Failure 400 {object} requestresponse.APIResponse "Parámetro 'codigo' es obligatorio"
// @Failure 404 {object} requestresponse.APIResponse "Estudiante no encontrado"
// @Failure 503 {object} requestresponse.APIResponse "Error al consultar el servicio externo"
// @router /estudiante/:codigo [get]
func (c *SemaforoController) ObtenerEstudiante() {
	defer errorhandler.HandlePanic(&c.Controller)

	if codigo, ok := helpers.GetPathParamOrError(&c.Controller, "codigo"); ok {
		limit, _ := c.GetInt("limit", 10)
		offset, _ := c.GetInt("offset", 0)
		resp := services.ConsultarEstudiante(codigo, limit, offset)
		helpers.RenderResponse(&c.Controller, resp)
	}
}

// ObtenerEstudiantes ...
// @Title ObtenerEstudiantes
// @Description obtener todos los estudiantes
// @Param	limit	query	int	false	"Número de registros por página"
// @Param	offset	query	int	false	"Número de registros a saltar"
// @Success 200 {object} []models.SemaforoTable
// @Failure 503
// @router / [get]
func (c *SemaforoController) ObtenerEstudiantes() {
	defer errorhandler.HandlePanic(&c.Controller)

	limit, _ := c.GetInt("limit", 10)
	offset, _ := c.GetInt("offset", 0)
	resp := services.ConsultarEstudiantes(limit, offset)
	helpers.RenderResponse(&c.Controller, resp)
}

// ObtenerEstudiantesProyecto ...
// @Title ObtenerEstudiantesProyecto
// @Description obtener estudiantes por proyectos del coordinador
// @Param	id_coordinador	path 	int	true	"ID del coordinador"
// @Param	limit	query	int	false	"Número de registros por página"
// @Param	offset	query	int	false	"Número de registros a saltar"
// @Success 200 {object} []models.SemaforoTable
// @Failure 400 :id_coordinador is empty
// @router /proyecto/:id_coordinador [get]
func (c *SemaforoController) ObtenerEstudiantesProyecto() {
	defer errorhandler.HandlePanic(&c.Controller)
	if id_coordinador, ok := helpers.GetPathParamOrError(&c.Controller, "id_coordinador"); ok {
		limit, _ := c.GetInt("limit", 10)
		offset, _ := c.GetInt("offset", 0)
		resp := services.ConsultarEstudiantesProyecto(id_coordinador, limit, offset)
		helpers.RenderResponse(&c.Controller, resp)
	}
}

// ObtenerEstudiantesFacultad ...
// @Title ObtenerEstudiantesFacultad
// @Description obtener estudiantes por facultad del secretario academico
// @Param	id_secretario	path 	int	true	"ID del secretario académico"
// @Param	limit	query	int	false	"Número de registros por página"
// @Param	offset	query	int	false	"Número de registros a saltar"
// @Success 200 {object} []models.SemaforoTable
// @Failure 400 :id_secretario is empty
// @router /facultad/:id_secretario [get]
func (c *SemaforoController) ObtenerEstudiantesFacultad() {
	defer errorhandler.HandlePanic(&c.Controller)

	if id_secretario, ok := helpers.GetPathParamOrError(&c.Controller, "id_secretario"); ok {
		limit, _ := c.GetInt("limit", 10)
		offset, _ := c.GetInt("offset", 0)
		resp := services.ConsultarEstudiantesFacultad(id_secretario, limit, offset)
		helpers.RenderResponse(&c.Controller, resp)
	}
}

// ObtenerEstudiantesFacultadLaboratorios ...
// @Title ObtenerEstudiantesFacultadLaboratorios
// @Description obtener estudiantes por facultad del coordinador de laboratorios
// @Param	id_coordinador_lab	path 	int	true	"ID del coordinador de laboratorios"
// @Success 200 {object} []models.SemaforoTable
// @Failure 400 :id_coordinador_lab is empty
// @router /facultad/laboratorios/:id_coordinador_lab [get]
func (c *SemaforoController) ObtenerEstudiantesFacultadLaboratorios() {
	defer errorhandler.HandlePanic(&c.Controller)

	if id_coordinador_lab, ok := helpers.GetPathParamOrError(&c.Controller, "id_coordinador_lab"); ok {
		resp := services.ConsultarEstudiantesFacultadLaboratorios(id_coordinador_lab)
		helpers.RenderResponse(&c.Controller, resp)
	}
}
