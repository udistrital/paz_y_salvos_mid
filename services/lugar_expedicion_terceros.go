package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/udistrital/paz_y_salvos_mid/models"
	"github.com/udistrital/utils_oas/v2/request"
)

var validarLugarExpedicionTerceros = validator.New(validator.WithRequiredStructEnabled())

type identificacionExpedicionGrado struct {
	Id               int    `json:"Id" validate:"gt=0"`
	Numero           string `json:"Numero" validate:"required"`
	CiudadExpedicion int    `json:"CiudadExpedicion" validate:"gte=0"`
	Activo           bool   `json:"Activo" validate:"eq=true"`
	TerceroId        struct {
		Id int `json:"Id" validate:"gt=0"`
	} `json:"TerceroId" validate:"required"`
	TipoDocumentoId struct {
		Id                int    `json:"Id" validate:"gt=0"`
		CodigoAbreviacion string `json:"CodigoAbreviacion" validate:"required"`
	} `json:"TipoDocumentoId" validate:"required"`
}

func identificacionAutenticadaGrado(user *estudianteGrado) (*identificacionExpedicionGrado, map[string]interface{}, error) {
	autorizacion, _ := user.Ctx.Value("Authorization").(string)
	sesion, err := sesionAutenticadaGrado(user.Ctx, autorizacion)
	if err != nil {
		return nil, nil, err
	}
	if err := validarLugarExpedicionTerceros.Struct(struct {
		Sub       string `validate:"required"`
		Documento string `validate:"required"`
	}{Sub: sesion.Sub, Documento: sesion.Documento}); err != nil {
		return nil, nil, falloGrado(http.StatusUnauthorized, "La identidad de la sesión no coincide")
	}

	base, err := baseGrado("UrlTercerosCrud")
	if err != nil {
		return nil, nil, err
	}
	q := url.Values{"query": {fmt.Sprintf("Activo:true,TerceroId.Id:%d,Numero:%s", user.TerceroID, sesion.Documento)}, "limit": {"0"}}
	var raw json.RawMessage
	if _, err := request.GetWithContext(user.Ctx, base+"datos_identificacion?"+q.Encode(), &raw); err != nil {
		return nil, nil, falloGrado(http.StatusServiceUnavailable, "No se pudo consultar el lugar de expedición en Terceros")
	}
	registros, err := listaGrado[map[string]interface{}](raw)
	if err != nil || len(registros) == 0 {
		return nil, nil, falloGrado(http.StatusServiceUnavailable, "Datos de identificación ausentes o incompletos")
	}

	var encontrada *identificacionExpedicionGrado
	var original map[string]interface{}
	for _, registro := range registros {
		contenido, err := json.Marshal(registro)
		if err != nil {
			return nil, nil, falloGrado(http.StatusServiceUnavailable, "Datos de identificación inválidos")
		}
		var identificacion identificacionExpedicionGrado
		if json.Unmarshal(contenido, &identificacion) != nil || validarLugarExpedicionTerceros.Struct(identificacion) != nil {
			return nil, nil, falloGrado(http.StatusServiceUnavailable, "Datos de identificación inválidos")
		}
		if identificacion.TerceroId.Id != user.TerceroID || strings.TrimSpace(identificacion.Numero) != sesion.Documento ||
			strings.EqualFold(strings.TrimSpace(identificacion.TipoDocumentoId.CodigoAbreviacion), "CODE") {
			continue
		}
		if encontrada != nil {
			return nil, nil, falloGrado(http.StatusConflict, "La identificación autenticada no es única")
		}
		copia := identificacion
		encontrada = &copia
		original = registro
	}
	if encontrada == nil {
		return nil, nil, falloGrado(http.StatusForbidden, "La identificación autenticada no pertenece al estudiante")
	}
	return encontrada, original, nil
}

func ObtenerLugarExpedicionIdentificacionGrado(ctx context.Context, auth string, terceroID int) (*models.LugarExpedicionIdentificacionGrado, error) {
	user, err := resolverEstudiante(ctx, auth, terceroID)
	if err != nil {
		return nil, err
	}
	identificacion, _, err := identificacionAutenticadaGrado(user)
	if err != nil {
		return nil, err
	}
	if identificacion.CiudadExpedicion == 0 {
		return &models.LugarExpedicionIdentificacionGrado{Registrado: false}, nil
	}
	lugar, err := resolverLugarExpedicionGrado(user.Ctx, identificacion.CiudadExpedicion)
	if err != nil {
		return nil, err
	}
	return &models.LugarExpedicionIdentificacionGrado{Registrado: true, Lugar: lugar}, nil
}

func RegistrarLugarExpedicionIdentificacionGrado(ctx context.Context, auth string, entrada models.RegistrarLugarExpedicionGrado) (*models.LugarExpedicionIdentificacionGrado, error) {
	if err := validarLugarExpedicionTerceros.Struct(entrada); err != nil {
		return nil, falloGrado(http.StatusBadRequest, "Tercero y ciudad requeridos")
	}
	user, err := resolverEstudiante(ctx, auth, entrada.TerceroId)
	if err != nil {
		return nil, err
	}
	identificacion, original, err := identificacionAutenticadaGrado(user)
	if err != nil {
		return nil, err
	}
	if identificacion.CiudadExpedicion > 0 {
		lugar, err := resolverLugarExpedicionGrado(user.Ctx, identificacion.CiudadExpedicion)
		if err != nil {
			return nil, err
		}
		if lugar.Id != entrada.LugarId {
			return nil, falloGrado(http.StatusConflict, "El lugar de expedición ya está registrado en Terceros")
		}
		return &models.LugarExpedicionIdentificacionGrado{Registrado: true, Lugar: lugar}, nil
	}
	lugar, err := resolverLugarExpedicionGrado(user.Ctx, entrada.LugarId)
	if err != nil {
		return nil, err
	}
	original["CiudadExpedicion"] = lugar.Id
	base, err := baseGrado("UrlTercerosCrud")
	if err != nil {
		return nil, err
	}
	var respuesta json.RawMessage
	status, err := request.PutWithContext(user.Ctx, base+"datos_identificacion/"+strconv.Itoa(identificacion.Id), original, &respuesta)
	if err != nil || status != http.StatusOK {
		return nil, falloGrado(http.StatusServiceUnavailable, "No se pudo actualizar el lugar de expedición en Terceros")
	}
	actualizada, _, err := identificacionAutenticadaGrado(user)
	if err != nil || actualizada.Id != identificacion.Id || actualizada.CiudadExpedicion != lugar.Id {
		return nil, falloGrado(http.StatusServiceUnavailable, "No se pudo confirmar el lugar de expedición en Terceros")
	}
	return &models.LugarExpedicionIdentificacionGrado{Registrado: true, Lugar: lugar}, nil
}
