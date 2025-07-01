package helpers

import (
	"fmt"

	"github.com/astaxie/beego"
	"github.com/udistrital/utils_oas/requestresponse"
)

// Renderiza una respuesta estructurada
func RenderResponse(c *beego.Controller, resp requestresponse.APIResponse) {
	c.Ctx.Output.SetStatus(resp.Status)
	c.Data["json"] = resp
	c.ServeJSON()
}

// Renderiza un error directamente
func RenderError(c *beego.Controller, status int, message string) {
	resp := requestresponse.APIResponseDTO(false, status, nil, message)
	RenderResponse(c, resp)
}

// Obtiene un parámetro y valida que exista, o retorna error
func GetPathParamOrError(c *beego.Controller, param string) (string, bool) {
	val := c.Ctx.Input.Param(":" + param)
	if val == "" {
		RenderError(c, 400, fmt.Sprintf("El parámetro '%s' es obligatorio", param))
		return "", false
	}
	return val, true
}
