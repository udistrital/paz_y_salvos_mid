// @APIVersion 1.0.0
// @Title beego Test API
// @Description beego has a very cool tools to autogenerate documents for your API
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
			beego.NSRouter("/revision/filtros", &controllers.InscripcionGradoController{}, "get:FiltrosRevisionDocumental"),
			beego.NSRouter("/revision", &controllers.InscripcionGradoController{}, "get:ListarRevisionDocumental"),
			beego.NSRouter("/revision/:id", &controllers.InscripcionGradoController{}, "get:ConsultarRevisionDocumental;post:RevisarDocumentacion"),
			beego.NSRouter("/revision/:id/soportes/:tipo", &controllers.InscripcionGradoController{}, "get:DescargarSoporteRevision"),
			beego.NSRouter("/borrador/:id/radicar", &controllers.InscripcionGradoController{}, "post:Radicar"),
			beego.NSRouter("/borrador/:id/subsanar", &controllers.InscripcionGradoController{}, "post:Subsanar"),
			beego.NSRouter("/borrador/:id/soportes", &controllers.InscripcionGradoController{}, "get:ListarSoportes"),
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
