package services

import (
	"context"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/udistrital/paz_y_salvos_mid/models"
	"github.com/udistrital/utils_oas/v2/request"
)

var cedulaGrado = regexp.MustCompile(`^[1-9][0-9]{0,37}$`)

func ListarDirectoresGrado(ctx context.Context, auth string) ([]models.DirectorGrado, error) {
	user, err := resolverEstudiante(ctx, auth)
	if err != nil {
		return nil, err
	}
	base, err := baseGrado("UrlAcademicaCore")
	if err != nil {
		return nil, err
	}
	var resp models.APIResponseData[[]models.DirectorGrado]
	if _, err := request.GetWithContext(user.Ctx, base+"grados/directores", &resp); err != nil {
		return nil, falloGrado(503, "No se pudo consultar la lista de directores")
	}
	if !resp.Success || resp.Status != 200 || resp.Data == nil {
		return nil, falloGrado(503, "Respuesta de directores no verificable")
	}
	vistos := make(map[string]bool)
	for _, director := range resp.Data {
		if !cedulaGrado.MatchString(director.Identificacion) || strings.TrimSpace(director.Nombre) == "" || strings.TrimSpace(director.Apellido) == "" || director.Estado != "A" || vistos[director.Identificacion] {
			return nil, falloGrado(503, "Catálogo de directores inconsistente")
		}
		vistos[director.Identificacion] = true
	}
	return resp.Data, nil
}

func ValidarDirectorGrado(ctx context.Context, auth, identificacion string) error {
	if !cedulaGrado.MatchString(identificacion) {
		return falloGrado(http.StatusBadRequest, "Director inválido")
	}
	user, err := resolverEstudiante(ctx, auth)
	if err != nil {
		return err
	}
	base, err := baseGrado("UrlAcademicaCore")
	if err != nil {
		return err
	}
	var resp models.APIResponseData[models.DirectorGrado]
	status, err := request.GetWithContext(user.Ctx, base+"grados/directores/"+url.PathEscape(identificacion), &resp)
	if err != nil {
		if status == http.StatusNotFound {
			return falloGrado(http.StatusConflict, "El director seleccionado ya no es docente activo")
		}
		return falloGrado(http.StatusServiceUnavailable, "No se pudo revalidar el director")
	}
	d := resp.Data
	if !resp.Success || resp.Status != http.StatusOK || d.Identificacion != identificacion ||
		strings.TrimSpace(d.Nombre) == "" || strings.TrimSpace(d.Apellido) == "" || d.Estado != "A" {
		return falloGrado(http.StatusServiceUnavailable, "Respuesta de director no verificable")
	}
	return nil
}

func ListarModalidadesGrado(ctx context.Context, auth string) ([]models.ModalidadGrado, error) {
	user, err := resolverEstudiante(ctx, auth)
	if err != nil {
		return nil, err
	}
	base, err := baseGrado("UrlAcademicaCore")
	if err != nil {
		return nil, err
	}
	var resp models.APIResponseData[[]models.ModalidadGrado]
	if _, err := request.GetWithContext(user.Ctx, base+"grados/modalidades", &resp); err != nil {
		return nil, falloGrado(503, "No se pudo consultar la lista de modalidades")
	}
	if !resp.Success || resp.Status != 200 || resp.Data == nil {
		return nil, falloGrado(503, "Respuesta de modalidades no verificable")
	}
	vistos := make(map[int64]bool)
	for _, modalidad := range resp.Data {
		if modalidad.Codigo <= 0 || strings.TrimSpace(modalidad.Nombre) == "" || strings.TrimSpace(modalidad.Abreviatura) == "" || modalidad.Estado != "A" || vistos[modalidad.Codigo] {
			return nil, falloGrado(503, "Catálogo de modalidades inconsistente")
		}
		vistos[modalidad.Codigo] = true
	}
	return resp.Data, nil
}

func ValidarModalidadGrado(ctx context.Context, auth string, codigo int64) error {
	if codigo <= 0 {
		return falloGrado(http.StatusBadRequest, "Modalidad inválida")
	}
	user, err := resolverEstudiante(ctx, auth)
	if err != nil {
		return err
	}
	base, err := baseGrado("UrlAcademicaCore")
	if err != nil {
		return err
	}
	var resp models.APIResponseData[models.ModalidadGrado]
	status, err := request.GetWithContext(user.Ctx, base+"grados/modalidades/"+strconv.FormatInt(codigo, 10), &resp)
	if err != nil {
		if status == http.StatusNotFound {
			return falloGrado(http.StatusConflict, "La modalidad seleccionada ya no está activa")
		}
		return falloGrado(http.StatusServiceUnavailable, "No se pudo revalidar la modalidad")
	}
	m := resp.Data
	if !resp.Success || resp.Status != http.StatusOK || m.Codigo != codigo || strings.TrimSpace(m.Nombre) == "" ||
		strings.TrimSpace(m.Abreviatura) == "" || m.Estado != "A" {
		return falloGrado(http.StatusServiceUnavailable, "Respuesta de modalidad no verificable")
	}
	return nil
}
