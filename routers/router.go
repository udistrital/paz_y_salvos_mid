// @APIVersion 1.0.0
// @Title Paz y Salvos MID
// @Description Orquesta consultas y decisiones del semáforo y de solicitudes de grado
// @Contact astaxie@gmail.com
// @TermsOfServiceUrl http://beego.me/
// @License Apache 2.0
// @LicenseUrl http://www.apache.org/licenses/LICENSE-2.0.html
package routers

import (
	beego "github.com/beego/beego/v2/server/web"
	"github.com/udistrital/paz_y_salvos_mid/controllers"
)

func init() {
	ns := beego.NewNamespace("/v1",

		beego.NSNamespace("/semaforo",
			beego.NSInclude(
				&controllers.SemaforoController{},
			),
		),
		beego.NSNamespace("/solicitud-grado",
			beego.NSRouter("/paz-salvos/aprobados", &controllers.InscripcionGradoController{}, "get:ListarAprobadosSecretaria"),
			beego.NSRouter("/paz-salvos", &controllers.InscripcionGradoController{}, "get:ListarPazSalvos"),
			beego.NSRouter("/paz-salvos/usuario", &controllers.InscripcionGradoController{}, "get:UsuarioPazSalvos"),
			beego.NSRouter("/paz-salvos/filtros", &controllers.InscripcionGradoController{}, "get:FiltrosPazSalvos"),
			beego.NSRouter("/:id/paz-salvos", &controllers.InscripcionGradoController{}, "get:ConsultarPazSalvos"),
			beego.NSRouter("/:id/paz-salvos/:tipo", &controllers.InscripcionGradoController{}, "post:DecidirPazSalvo"),
			beego.NSRouter("/:id/paz-salvos/soportes/:tipo", &controllers.InscripcionGradoController{}, "get:DescargarSoportePazSalvo"),
			beego.NSRouter("/revision/filtros", &controllers.InscripcionGradoController{}, "get:FiltrosRevisionDocumental"),
			beego.NSRouter("/revision", &controllers.InscripcionGradoController{}, "get:ListarRevisionDocumental"),
			beego.NSRouter("/revision/:id", &controllers.InscripcionGradoController{}, "get:ConsultarRevisionDocumental;post:RevisarDocumentacion"),
			beego.NSRouter("/revision/:id/versiones", &controllers.InscripcionGradoController{}, "get:ListarVersionesSecretaria"),
			beego.NSRouter("/revision/:id/versiones/:numero", &controllers.InscripcionGradoController{}, "get:ConsultarVersionSecretaria"),
			beego.NSRouter("/revision/:id/versiones/:numero/soportes/:tipo", &controllers.InscripcionGradoController{}, "get:DescargarVersionSecretaria"),
			beego.NSRouter("/revision/:id/soportes/:tipo", &controllers.InscripcionGradoController{}, "get:DescargarSoporteRevision"),
			beego.NSRouter("/borrador/:id/radicar", &controllers.InscripcionGradoController{}, "post:Radicar"),
			beego.NSRouter("/borrador/:id/subsanar", &controllers.InscripcionGradoController{}, "post:Subsanar"),
			beego.NSRouter("/borrador/:id/soportes", &controllers.InscripcionGradoController{}, "get:ListarSoportes"),
			beego.NSRouter("/borrador/:id/versiones", &controllers.InscripcionGradoController{}, "get:ListarVersionesEstudiante"),
			beego.NSRouter("/borrador/:id/versiones/:numero", &controllers.InscripcionGradoController{}, "get:ConsultarVersionEstudiante"),
			beego.NSRouter("/borrador/:id/versiones/:numero/soportes/:tipo", &controllers.InscripcionGradoController{}, "get:DescargarVersionEstudiante"),
			beego.NSRouter("/borrador/:id/soportes/:tipo", &controllers.InscripcionGradoController{}, "put:CargarSoporte;get:DescargarSoporte;delete:EliminarSoporte"),
			beego.NSRouter("/directores", &controllers.InscripcionGradoController{}, "get:ListarDirectores"),
			beego.NSRouter("/modalidades", &controllers.InscripcionGradoController{}, "get:ListarModalidades"),
			beego.NSRouter("/paises-expedicion", &controllers.InscripcionGradoController{}, "get:ListarPaisesExpedicion"),
			beego.NSRouter("/departamentos-expedicion", &controllers.InscripcionGradoController{}, "get:ListarDepartamentosExpedicion"),
			beego.NSRouter("/lugares-expedicion", &controllers.InscripcionGradoController{}, "get:ListarLugaresExpedicion"),
			beego.NSRouter("/lugares-expedicion/:id", &controllers.InscripcionGradoController{}, "get:ObtenerLugarExpedicion"),
			beego.NSRouter("/lugar-expedicion-identificacion", &controllers.InscripcionGradoController{}, "get:ObtenerLugarExpedicionIdentificacion"),
			beego.NSRouter("/borrador", &controllers.InscripcionGradoController{}, "post:CrearBorrador;get:ObtenerBorrador"),
			beego.NSRouter("/borrador/:id", &controllers.InscripcionGradoController{}, "put:GuardarBorrador"),
		),
	)
	beego.AddNamespace(ns)
}
