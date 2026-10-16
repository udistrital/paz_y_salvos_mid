package services

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/udistrital/paz_y_salvos_mid/models"
	"github.com/udistrital/utils_oas/v2/request"
)

func baseOikosGradoV2() (string, error) {
	base, err := baseGrado("UrlcrudOikos")
	if err != nil {
		return "", err
	}
	base = strings.TrimSuffix(base, "/")
	if strings.HasSuffix(base, "/v1") {
		base = strings.TrimSuffix(base, "/v1")
	} else if strings.HasSuffix(base, "/v2") {
		base = strings.TrimSuffix(base, "/v2")
	}
	return base + "/v2/", nil
}

// facultadLaboratoriosGrado obtiene la facultad desde Oikos. Una vinculación
// puede apuntar directamente a una facultad (272) aun si esta tiene un padre
// administrativo; en los demás casos se exige una única relación padre-hija.
func facultadLaboratoriosGrado(ctx context.Context, dependenciaID int) (int, error) {
	base, err := baseOikosGradoV2()
	if err != nil {
		return 0, err
	}
	var dependencia models.DependenciaOikosV2
	if _, err := request.GetWithContext(ctx, base+"dependencia/"+strconv.Itoa(dependenciaID), &dependencia); err != nil ||
		dependencia.Id != dependenciaID || !dependencia.Activo || strings.TrimSpace(dependencia.Nombre) == "" {
		return 0, falloGrado(http.StatusServiceUnavailable, "Dependencia de Laboratorios no verificable en Oikos")
	}
	if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(dependencia.Nombre)), "FACULTAD ") {
		return dependenciaID, nil
	}
	return facultadDependenciaGrado(ctx, dependenciaID)
}
