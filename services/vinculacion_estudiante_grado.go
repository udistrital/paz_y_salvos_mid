package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/udistrital/utils_oas/v2/request"
)

type vinculacionEstudianteGrado struct {
	DependenciaId          int    `json:"DependenciaId"`
	Activo                 bool   `json:"Activo"`
	FechaInicioVinculacion string `json:"FechaInicioVinculacion"`
	FechaFinVinculacion    string `json:"FechaFinVinculacion"`
}

// vinculacionesEstudianteGrado devuelve las vinculaciones TV/EST activas y
// vigentes del tercero identificado por el MID.
func vinculacionesEstudianteGrado(ctx context.Context, terceroID int) ([]vinculacionEstudianteGrado, error) {
	if terceroID <= 0 {
		return nil, falloGrado(http.StatusBadRequest, "Tercero inválido")
	}
	paramURL, err := baseGrado("UrlcrudParametros")
	if err != nil {
		return nil, err
	}
	query := url.Values{"query": {"CodigoAbreviacion:EST,TipoParametroId.CodigoAbreviacion:TV,Activo:true"}, "limit": {"0"}}
	var paramRaw json.RawMessage
	if _, err := request.GetWithContext(ctx, paramURL+"parametro?"+query.Encode(), &paramRaw); err != nil {
		return nil, falloGrado(http.StatusServiceUnavailable, "Vinculación estudiantil no verificable")
	}
	tipos, err := listaGrado[struct {
		Id                int    `json:"Id"`
		CodigoAbreviacion string `json:"CodigoAbreviacion"`
		Activo            bool   `json:"Activo"`
		TipoParametroId   struct {
			CodigoAbreviacion string `json:"CodigoAbreviacion"`
			Activo            bool   `json:"Activo"`
		} `json:"TipoParametroId"`
	}](paramRaw)
	if err != nil || len(tipos) != 1 || tipos[0].Id <= 0 || !tipos[0].Activo || tipos[0].CodigoAbreviacion != "EST" ||
		!tipos[0].TipoParametroId.Activo || tipos[0].TipoParametroId.CodigoAbreviacion != "TV" {
		return nil, falloGrado(http.StatusServiceUnavailable, "Tipo de vinculación estudiantil ambiguo")
	}
	tercerosURL, err := baseGrado("UrlTercerosCrud")
	if err != nil {
		return nil, err
	}
	q := url.Values{"query": {fmt.Sprintf("TerceroPrincipalId.Id:%d,TipoVinculacionId:%d,Activo:true", terceroID, tipos[0].Id)}, "limit": {"0"}}
	var raw json.RawMessage
	if _, err := request.GetWithContext(ctx, tercerosURL+"vinculacion?"+q.Encode(), &raw); err != nil {
		return nil, falloGrado(http.StatusServiceUnavailable, "No se pudo verificar la vinculación")
	}
	vinculos, err := listaGrado[vinculacionEstudianteGrado](raw)
	if err != nil {
		return nil, falloGrado(http.StatusServiceUnavailable, "Vinculación inválida")
	}
	resultado := make([]vinculacionEstudianteGrado, 0, len(vinculos))
	ahora := time.Now()
	for _, vinculo := range vinculos {
		if !vinculo.Activo || vinculo.DependenciaId <= 0 || !fechaVinculacionGradoVigente(vinculo.FechaInicioVinculacion, ahora, false) ||
			!fechaVinculacionGradoVigente(vinculo.FechaFinVinculacion, ahora, true) {
			continue
		}
		resultado = append(resultado, vinculo)
	}
	return resultado, nil
}

func fechaVinculacionGradoVigente(raw string, ahora time.Time, fin bool) bool {
	if raw == "" || strings.HasPrefix(raw, "0001-") {
		return true
	}
	fecha, err := fechaGradoServidor(raw)
	if err != nil {
		return false
	}
	if fin {
		return !ahora.After(fecha)
	}
	return !ahora.Before(fecha)
}

func verificarVinculacionEstudianteGrado(ctx context.Context, terceroID int) error {
	vinculos, err := vinculacionesEstudianteGrado(ctx, terceroID)
	if err != nil {
		return err
	}
	if len(vinculos) == 0 {
		return falloGrado(http.StatusForbidden, "El tercero no tiene vinculación estudiantil vigente")
	}
	return nil
}
