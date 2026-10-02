package routers

import (
	beego "github.com/beego/beego/v2/server/web"
	"github.com/beego/beego/v2/server/web/context/param"
)

func init() {

	beego.GlobalControllerRouter["github.com/udistrital/paz_y_salvos_mid/controllers:SemaforoController"] = append(beego.GlobalControllerRouter["github.com/udistrital/paz_y_salvos_mid/controllers:SemaforoController"],
		beego.ControllerComments{
			Method:           "ObtenerEstudiantes",
			Router:           "/",
			AllowHTTPMethods: []string{"get"},
			MethodParams:     param.Make(),
			Filters:          nil,
			Params:           nil})

	beego.GlobalControllerRouter["github.com/udistrital/paz_y_salvos_mid/controllers:SemaforoController"] = append(beego.GlobalControllerRouter["github.com/udistrital/paz_y_salvos_mid/controllers:SemaforoController"],
		beego.ControllerComments{
			Method:           "ObtenerSemaforosAsistente",
			Router:           "/asistente_proyecto/:cedula",
			AllowHTTPMethods: []string{"get"},
			MethodParams:     param.Make(),
			Filters:          nil,
			Params:           nil})

	beego.GlobalControllerRouter["github.com/udistrital/paz_y_salvos_mid/controllers:SemaforoController"] = append(beego.GlobalControllerRouter["github.com/udistrital/paz_y_salvos_mid/controllers:SemaforoController"],
		beego.ControllerComments{
			Method:           "ObtenerEstudiante",
			Router:           "/estudiante/:codigo",
			AllowHTTPMethods: []string{"get"},
			MethodParams:     param.Make(),
			Filters:          nil,
			Params:           nil})

	beego.GlobalControllerRouter["github.com/udistrital/paz_y_salvos_mid/controllers:SemaforoController"] = append(beego.GlobalControllerRouter["github.com/udistrital/paz_y_salvos_mid/controllers:SemaforoController"],
		beego.ControllerComments{
			Method:           "ObtenerEstudiantesFacultad",
			Router:           "/facultad/:id_secretario",
			AllowHTTPMethods: []string{"get"},
			MethodParams:     param.Make(),
			Filters:          nil,
			Params:           nil})

	beego.GlobalControllerRouter["github.com/udistrital/paz_y_salvos_mid/controllers:SemaforoController"] = append(beego.GlobalControllerRouter["github.com/udistrital/paz_y_salvos_mid/controllers:SemaforoController"],
		beego.ControllerComments{
			Method:           "ObtenerEstudiantesFacultadLaboratorios",
			Router:           "/laboratorios/:id_coordinador_lab",
			AllowHTTPMethods: []string{"get"},
			MethodParams:     param.Make(),
			Filters:          nil,
			Params:           nil})

	beego.GlobalControllerRouter["github.com/udistrital/paz_y_salvos_mid/controllers:SemaforoController"] = append(beego.GlobalControllerRouter["github.com/udistrital/paz_y_salvos_mid/controllers:SemaforoController"],
		beego.ControllerComments{
			Method:           "ObtenerEstudiantesProyecto",
			Router:           "/proyecto/:id_coordinador",
			AllowHTTPMethods: []string{"get"},
			MethodParams:     param.Make(),
			Filters:          nil,
			Params:           nil})

}
