package controllers

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	beego "github.com/beego/beego/v2/server/web"
	"github.com/udistrital/paz_y_salvos_mid/models"
	"github.com/udistrital/paz_y_salvos_mid/services"
)

type InscripcionGradoController struct{ beego.Controller }

func (c *InscripcionGradoController) respuesta(status int, data interface{}, mensaje string) {
	c.Ctx.Output.SetStatus(status)
	c.Data["json"] = models.APIResponse{Success: status < 400, Status: status, Message: mensaje, Data: data}
	_ = c.ServeJSON()
}

func (c *InscripcionGradoController) errorGrado(err error) {
	var e *services.ErrorInscripcionGrado
	if errors.As(err, &e) {
		c.respuesta(e.Status, nil, e.Mensaje)
		return
	}
	c.respuesta(http.StatusInternalServerError, nil, "Error procesando inscripción a grado")
}

func (c *InscripcionGradoController) jsonEntrada(destino interface{}) bool {
	d := json.NewDecoder(bytes.NewReader(c.Ctx.Input.RequestBody))
	d.DisallowUnknownFields()
	if err := d.Decode(destino); err != nil {
		c.respuesta(400, nil, "JSON inválido o campos desconocidos")
		return false
	}
	if err := d.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		c.respuesta(400, nil, "Se requiere un único objeto JSON")
		return false
	}
	return true
}

func (c *InscripcionGradoController) ConsultarPazSalvos() {
	id, err := strconv.Atoi(c.Ctx.Input.Param(":id"))
	if err != nil || id <= 0 {
		c.respuesta(http.StatusBadRequest, nil, "Solicitud inválida")
		return
	}
	resultado, err := services.ConsultarPazSalvosGrado(c.Ctx.Request.Context(), c.Ctx.Input.Header("Authorization"), id, c.GetString("perfil"))
	if err != nil {
		c.errorGrado(err)
		return
	}
	c.respuesta(http.StatusOK, resultado, "Paz y Salvos consultados")
}

func (c *InscripcionGradoController) ListarPazSalvos() {
	limit, errLimit := c.GetInt("limit", 20)
	offset, errOffset := c.GetInt("offset", 0)
	periodo, errPeriodo := c.GetInt("periodo_id", 0)
	programa, errPrograma := c.GetInt("programa_id", 0)
	facultad, errFacultad := c.GetInt("facultad_id", 0)
	if errLimit != nil || errOffset != nil || errPeriodo != nil || errPrograma != nil || errFacultad != nil {
		c.respuesta(http.StatusBadRequest, nil, "Filtros de bandeja inválidos")
		return
	}
	resultado, err := services.ListarPazSalvosGrado(c.Ctx.Request.Context(), c.Ctx.Input.Header("Authorization"), c.GetString("tipo"), c.GetString("perfil"), limit, offset, periodo, programa, facultad, c.GetString("codigo"))
	if err != nil {
		c.errorGrado(err)
		return
	}
	c.respuesta(http.StatusOK, resultado, "Bandeja de Paz y Salvos consultada")
}

func (c *InscripcionGradoController) FiltrosPazSalvos() {
	resultado, err := services.FiltrosPazSalvosGrado(c.Ctx.Request.Context(), c.Ctx.Input.Header("Authorization"), c.GetString("tipo"), c.GetString("perfil"))
	if err != nil {
		c.errorGrado(err)
		return
	}
	c.respuesta(http.StatusOK, resultado, "Filtros de Paz y Salvos consultados")
}

func (c *InscripcionGradoController) UsuarioPazSalvos() {
	resultado, err := services.UsuarioPazSalvosGrado(c.Ctx.Request.Context(), c.Ctx.Input.Header("Authorization"), c.GetString("perfil"))
	if err != nil {
		c.errorGrado(err)
		return
	}
	c.respuesta(http.StatusOK, resultado, "Usuario de Paz y Salvos consultado")
}

func (c *InscripcionGradoController) DecidirPazSalvo() {
	id, err := strconv.Atoi(c.Ctx.Input.Param(":id"))
	if err != nil || id <= 0 {
		c.respuesta(http.StatusBadRequest, nil, "Solicitud inválida")
		return
	}
	var entrada models.DecidirPazSalvoGrado
	if !c.jsonEntrada(&entrada) {
		return
	}
	resultado, err := services.DecidirPazSalvoGrado(c.Ctx.Request.Context(), c.Ctx.Input.Header("Authorization"), id, c.Ctx.Input.Param(":tipo"), entrada)
	if err != nil {
		c.errorGrado(err)
		return
	}
	c.respuesta(http.StatusOK, resultado, "Decisión de Paz y Salvo registrada")
}

func (c *InscripcionGradoController) DescargarSoportePazSalvo() {
	id, err := strconv.Atoi(c.Ctx.Input.Param(":id"))
	if err != nil || id <= 0 {
		c.respuesta(http.StatusBadRequest, nil, "Solicitud inválida")
		return
	}
	resultado, err := services.DescargarSoportePazSalvoGrado(c.Ctx.Request.Context(), c.Ctx.Input.Header("Authorization"), id,
		c.GetString("tipo_check"), c.Ctx.Input.Param(":tipo"), c.GetString("perfil"))
	if err != nil {
		c.errorGrado(err)
		return
	}
	c.respuesta(http.StatusOK, resultado, "Consulta del PDF")
}

func (c *InscripcionGradoController) CrearBorrador() {
	var entrada models.CrearBorradorGrado
	if !c.jsonEntrada(&entrada) {
		return
	}
	res, err := services.CrearBorradorGrado(c.Ctx.Request.Context(), c.Ctx.Input.Header("Authorization"), entrada)
	if err != nil {
		c.errorGrado(err)
		return
	}
	c.respuesta(http.StatusCreated, res, "Borrador guardado")
}

func (c *InscripcionGradoController) ObtenerBorrador() {
	tercero, e0 := c.GetInt("tercero_id")
	periodo, e1 := c.GetInt("periodo_id")
	programa, e2 := c.GetInt("programa_id")
	if e0 != nil || e1 != nil || e2 != nil || tercero <= 0 || periodo <= 0 || programa <= 0 {
		c.respuesta(400, nil, "Tercero, periodo y programa requeridos")
		return
	}
	res, err := services.ObtenerBorradorGrado(c.Ctx.Request.Context(), c.Ctx.Input.Header("Authorization"), tercero, periodo, programa)
	if err != nil {
		c.errorGrado(err)
		return
	}
	c.respuesta(http.StatusOK, res, "Consulta exitosa")
}

func (c *InscripcionGradoController) GuardarBorrador() {
	id, err := strconv.Atoi(c.Ctx.Input.Param(":id"))
	if err != nil || id <= 0 {
		c.respuesta(400, nil, "ID inválido")
		return
	}
	var entrada models.GuardarBorradorGrado
	if !c.jsonEntrada(&entrada) {
		return
	}
	res, err := services.GuardarBorradorGrado(c.Ctx.Request.Context(), c.Ctx.Input.Header("Authorization"), id, entrada.TerceroId, entrada.Contenido)
	if err != nil {
		c.errorGrado(err)
		return
	}
	c.respuesta(http.StatusOK, res, "Borrador actualizado")
}

func (c *InscripcionGradoController) Radicar() {
	id, err := strconv.Atoi(c.Ctx.Input.Param(":id"))
	if err != nil || id <= 0 {
		c.respuesta(http.StatusBadRequest, nil, "ID inválido")
		return
	}
	var entrada models.RadicarGrado
	if !c.jsonEntrada(&entrada) {
		return
	}
	res, err := services.RadicarGrado(c.Ctx.Request.Context(), c.Ctx.Input.Header("Authorization"), id, entrada)
	if err != nil {
		c.errorGrado(err)
		return
	}
	c.respuesta(http.StatusOK, res, "Inscripción radicada")
}

func (c *InscripcionGradoController) Subsanar() {
	id, err := strconv.Atoi(c.Ctx.Input.Param(":id"))
	if err != nil || id <= 0 {
		c.respuesta(http.StatusBadRequest, nil, "Solicitud inválida")
		return
	}
	var entrada models.SubsanarGrado
	if !c.jsonEntrada(&entrada) {
		return
	}
	res, err := services.SubsanarGrado(c.Ctx.Request.Context(), c.Ctx.Input.Header("Authorization"), id, entrada)
	if err != nil {
		c.errorGrado(err)
		return
	}
	c.respuesta(http.StatusCreated, res, "Subsanación iniciada")
}

func (c *InscripcionGradoController) ListarDirectores() {
	tercero, err := c.GetInt("tercero_id")
	if err != nil || tercero <= 0 {
		c.respuesta(400, nil, "Tercero requerido")
		return
	}
	directores, err := services.ListarDirectoresGrado(c.Ctx.Request.Context(), c.Ctx.Input.Header("Authorization"), tercero)
	if err != nil {
		c.errorGrado(err)
		return
	}
	c.respuesta(http.StatusOK, directores, "Consulta exitosa")
}

func (c *InscripcionGradoController) ListarModalidades() {
	tercero, err := c.GetInt("tercero_id")
	if err != nil || tercero <= 0 {
		c.respuesta(400, nil, "Tercero requerido")
		return
	}
	modalidades, err := services.ListarModalidadesGrado(c.Ctx.Request.Context(), c.Ctx.Input.Header("Authorization"), tercero)
	if err != nil {
		c.errorGrado(err)
		return
	}
	c.respuesta(http.StatusOK, modalidades, "Consulta exitosa")
}

func (c *InscripcionGradoController) ListarPaisesExpedicion() {
	tercero, err := c.GetInt("tercero_id")
	if err != nil || tercero <= 0 {
		c.respuesta(http.StatusBadRequest, nil, "Tercero requerido")
		return
	}
	paises, err := services.ListarPaisesExpedicionGrado(c.Ctx.Request.Context(), c.Ctx.Input.Header("Authorization"), tercero)
	if err != nil {
		c.errorGrado(err)
		return
	}
	c.respuesta(http.StatusOK, paises, "Consulta exitosa")
}

func (c *InscripcionGradoController) ListarDepartamentosExpedicion() {
	tercero, errTercero := c.GetInt("tercero_id")
	pais, errPais := c.GetInt("pais_id")
	if errTercero != nil || errPais != nil || tercero <= 0 || pais <= 0 {
		c.respuesta(http.StatusBadRequest, nil, "Tercero y país requeridos")
		return
	}
	departamentos, err := services.ListarDepartamentosExpedicionGrado(c.Ctx.Request.Context(), c.Ctx.Input.Header("Authorization"), tercero, pais)
	if err != nil {
		c.errorGrado(err)
		return
	}
	c.respuesta(http.StatusOK, departamentos, "Consulta exitosa")
}

func (c *InscripcionGradoController) ListarLugaresExpedicion() {
	tercero, errTercero := c.GetInt("tercero_id")
	pais, errPais := c.GetInt("pais_id")
	departamento, errDepartamento := c.GetInt("departamento_id")
	if errTercero != nil || errPais != nil || errDepartamento != nil || tercero <= 0 || pais <= 0 || departamento <= 0 {
		c.respuesta(http.StatusBadRequest, nil, "Tercero, país y departamento requeridos")
		return
	}
	lugares, err := services.ListarLugaresExpedicionGrado(c.Ctx.Request.Context(), c.Ctx.Input.Header("Authorization"), tercero, pais, departamento)
	if err != nil {
		c.errorGrado(err)
		return
	}
	c.respuesta(http.StatusOK, lugares, "Consulta exitosa")
}

func (c *InscripcionGradoController) ObtenerLugarExpedicion() {
	id, errID := strconv.Atoi(c.Ctx.Input.Param(":id"))
	tercero, errTercero := c.GetInt("tercero_id")
	if errID != nil || errTercero != nil || id <= 0 || tercero <= 0 {
		c.respuesta(http.StatusBadRequest, nil, "Tercero y ciudad requeridos")
		return
	}
	lugar, err := services.ValidarLugarExpedicionGrado(c.Ctx.Request.Context(), c.Ctx.Input.Header("Authorization"), tercero, id)
	if err != nil {
		c.errorGrado(err)
		return
	}
	c.respuesta(http.StatusOK, lugar, "Consulta exitosa")
}

func (c *InscripcionGradoController) ObtenerLugarExpedicionIdentificacion() {
	tercero, err := c.GetInt("tercero_id")
	if err != nil || tercero <= 0 {
		c.respuesta(http.StatusBadRequest, nil, "Tercero requerido")
		return
	}
	resultado, err := services.ObtenerLugarExpedicionIdentificacionGrado(c.Ctx.Request.Context(), c.Ctx.Input.Header("Authorization"), tercero)
	if err != nil {
		c.errorGrado(err)
		return
	}
	c.respuesta(http.StatusOK, resultado, "Consulta exitosa")
}

func (c *InscripcionGradoController) ListarSoportes() {
	id, err := strconv.Atoi(c.Ctx.Input.Param(":id"))
	tercero, errTercero := c.GetInt("tercero_id")
	if err != nil || errTercero != nil || id <= 0 || tercero <= 0 {
		c.respuesta(400, nil, "Solicitud inválida")
		return
	}
	res, err := services.ListarSoportesGrado(c.Ctx.Request.Context(), c.Ctx.Input.Header("Authorization"), id, tercero)
	if err != nil {
		c.errorGrado(err)
		return
	}
	c.respuesta(200, res, "Soportes provisionales del borrador")
}

func (c *InscripcionGradoController) CargarSoporte() {
	id, err := strconv.Atoi(c.Ctx.Input.Param(":id"))
	if err != nil || id <= 0 {
		c.respuesta(400, nil, "Solicitud inválida")
		return
	}
	if len(c.Ctx.Input.RequestBody) > (services.MaxPDFGrado+2)/3*4+4096 {
		c.respuesta(413, nil, "El PDF no debe superar 5 MiB")
		return
	}
	var entrada models.CargarSoporteGrado
	if !c.jsonEntrada(&entrada) {
		return
	}
	res, err := services.CargarSoporteGrado(c.Ctx.Request.Context(), c.Ctx.Input.Header("Authorization"), id, c.Ctx.Input.Param(":tipo"), entrada)
	if err != nil {
		c.errorGrado(err)
		return
	}
	c.respuesta(200, res, "PDF asociado al borrador; documento provisional")
}

func (c *InscripcionGradoController) EliminarSoporte() {
	id, errID := strconv.Atoi(c.Ctx.Input.Param(":id"))
	tercero, errTercero := c.GetInt("tercero_id")
	formulario, errFormulario := c.GetInt("formulario_id")
	soporte, errSoporte := c.GetInt("soporte_actual_id")
	if errID != nil || errTercero != nil || errFormulario != nil || errSoporte != nil ||
		id <= 0 || tercero <= 0 || formulario <= 0 || soporte <= 0 {
		c.respuesta(http.StatusBadRequest, nil, "Solicitud, versión y soporte requeridos")
		return
	}
	res, err := services.EliminarSoporteGrado(c.Ctx.Request.Context(), c.Ctx.Input.Header("Authorization"), id,
		tercero, formulario, soporte, c.Ctx.Input.Param(":tipo"))
	if err != nil {
		c.errorGrado(err)
		return
	}
	c.respuesta(http.StatusOK, res, "PDF retirado del borrador; historial conservado")
}

func (c *InscripcionGradoController) DescargarSoporte() {
	id, err := strconv.Atoi(c.Ctx.Input.Param(":id"))
	tercero, errTercero := c.GetInt("tercero_id")
	if err != nil || errTercero != nil || id <= 0 || tercero <= 0 {
		c.respuesta(400, nil, "Solicitud inválida")
		return
	}
	res, err := services.DescargarSoporteGrado(c.Ctx.Request.Context(), c.Ctx.Input.Header("Authorization"), id, tercero, c.Ctx.Input.Param(":tipo"))
	if err != nil {
		c.errorGrado(err)
		return
	}
	c.respuesta(200, res, "Consulta del PDF")
}

func (c *InscripcionGradoController) ListarRevisionDocumental() {
	limit, errLimit := c.GetInt("limit", 20)
	offset, errOffset := c.GetInt("offset", 0)
	periodo, errPeriodo := c.GetInt("periodo_id", 0)
	programa, errPrograma := c.GetInt("programa_id", 0)
	if errLimit != nil || errOffset != nil || errPeriodo != nil || errPrograma != nil {
		c.respuesta(http.StatusBadRequest, nil, "Filtros de consulta inválidos")
		return
	}
	res, err := services.ListarRevisionDocumentalGrado(c.Ctx.Request.Context(), c.Ctx.Input.Header("Authorization"), limit, offset,
		periodo, programa, c.GetString("estado"), c.GetString("texto"))
	if err != nil {
		c.errorGrado(err)
		return
	}
	c.respuesta(http.StatusOK, res, "Solicitudes para revisión documental")
}

func (c *InscripcionGradoController) FiltrosRevisionDocumental() {
	res, err := services.FiltrosRevisionDocumentalGrado(c.Ctx.Request.Context(), c.Ctx.Input.Header("Authorization"))
	if err != nil {
		c.errorGrado(err)
		return
	}
	c.respuesta(http.StatusOK, res, "Filtros de revisión documental")
}

func (c *InscripcionGradoController) ConsultarRevisionDocumental() {
	id, err := strconv.Atoi(c.Ctx.Input.Param(":id"))
	if err != nil || id <= 0 {
		c.respuesta(http.StatusBadRequest, nil, "Solicitud inválida")
		return
	}
	res, err := services.ConsultarRevisionDocumentalGrado(c.Ctx.Request.Context(), c.Ctx.Input.Header("Authorization"), id)
	if err != nil {
		c.errorGrado(err)
		return
	}
	c.respuesta(http.StatusOK, res, "Expediente para revisión documental")
}

func (c *InscripcionGradoController) RevisarDocumentacion() {
	id, err := strconv.Atoi(c.Ctx.Input.Param(":id"))
	if err != nil || id <= 0 {
		c.respuesta(http.StatusBadRequest, nil, "Solicitud inválida")
		return
	}
	var entrada models.RevisarDocumentacionGrado
	if !c.jsonEntrada(&entrada) {
		return
	}
	res, err := services.RevisarDocumentacionSecretariaGrado(c.Ctx.Request.Context(), c.Ctx.Input.Header("Authorization"), id, entrada)
	if err != nil {
		c.errorGrado(err)
		return
	}
	c.respuesta(http.StatusOK, res, "Revisión documental registrada")
}

func (c *InscripcionGradoController) DescargarSoporteRevision() {
	id, err := strconv.Atoi(c.Ctx.Input.Param(":id"))
	if err != nil || id <= 0 {
		c.respuesta(http.StatusBadRequest, nil, "Solicitud inválida")
		return
	}
	res, err := services.DescargarSoporteRevisionGrado(c.Ctx.Request.Context(), c.Ctx.Input.Header("Authorization"), id, c.Ctx.Input.Param(":tipo"))
	if err != nil {
		c.errorGrado(err)
		return
	}
	c.respuesta(http.StatusOK, res, "Consulta del PDF para revisión")
}
