package helpers

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/beego/beego/v2/core/logs"
	beego "github.com/beego/beego/v2/server/web"
	"github.com/udistrital/paz_y_salvos_mid/models"
)

func RenderResponse(c *beego.Controller, response models.APIResponse) {
	c.Ctx.Output.SetStatus(response.Status)
	c.Data["json"] = response
	if err := c.ServeJSON(); err != nil {
		logs.Error("error al serializar la respuesta: %v", err)
	}
}

func RenderError(c *beego.Controller, status int, message string) {
	RenderResponse(c, models.APIResponse{Success: false, Status: status, Message: message, Data: nil})
}

func GetPathParamOrError(c *beego.Controller, param string) (string, bool) {
	value := c.Ctx.Input.Param(":" + param)
	if value == "" {
		RenderError(c, http.StatusBadRequest, fmt.Sprintf("El parámetro '%s' es obligatorio", param))
		return "", false
	}
	return value, true
}

func HandlePanic(c *beego.Controller) {
	if recovered := recover(); recovered != nil {
		logs.Error("Panic: %v", recovered)
		debug.PrintStack()
		message := fmt.Sprintf("Error service %s: An internal server error occurred.", ConfigString("appname"))
		message += fmt.Sprintf(" Request Info: URL: %s, Method: %s", c.Ctx.Request.URL, c.Ctx.Request.Method)
		message += " Time: " + time.Now().Format(time.RFC3339)
		RenderResponse(c, models.APIResponse{Success: false, Status: http.StatusInternalServerError, Message: message, Data: nil})
	}
}
