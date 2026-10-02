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
