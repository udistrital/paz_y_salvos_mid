package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/udistrital/paz_y_salvos_mid/models"
	"github.com/udistrital/utils_oas/v2/request"
)

var cedulaGrado = regexp.MustCompile(`^[1-9][0-9]{0,37}$`)

type parametroModalidadGrado struct {
	models.ModalidadGrado
	TipoParametroId struct {
		CodigoAbreviacion string `json:"CodigoAbreviacion"`
		Activo            bool   `json:"Activo"`
	} `json:"TipoParametroId"`
}

func ListarDirectoresGrado(ctx context.Context, auth string, terceroID int) ([]models.DirectorGrado, error) {
	user, err := resolverEstudiante(ctx, auth, terceroID)
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

func ValidarDirectorGrado(ctx context.Context, auth string, terceroID int, identificacion string) error {
	if !cedulaGrado.MatchString(identificacion) {
		return falloGrado(http.StatusBadRequest, "Director inválido")
	}
	user, err := resolverEstudiante(ctx, auth, terceroID)
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

func ListarModalidadesGrado(ctx context.Context, auth string, terceroID int) ([]models.ModalidadGrado, error) {
	user, err := resolverEstudiante(ctx, auth, terceroID)
	if err != nil {
		return nil, err
	}
	base, err := baseGrado("UrlcrudParametros")
	if err != nil {
		return nil, err
	}
	q := url.Values{"query": {"TipoParametroId.CodigoAbreviacion:MOD_TRG,TipoParametroId.Activo:true,Activo:true"}, "limit": {"0"}}
	var raw json.RawMessage
	if _, err := request.GetWithContext(user.Ctx, base+"parametro?"+q.Encode(), &raw); err != nil {
		return nil, falloGrado(503, "No se pudo consultar la lista de modalidades")
	}
	parametros, err := listaGrado[parametroModalidadGrado](raw)
	if err != nil || len(parametros) == 0 {
		return nil, falloGrado(503, "Respuesta de modalidades no verificable")
	}
	modalidades := make([]models.ModalidadGrado, 0, len(parametros))
	vistos := make(map[int]bool)
	codigosVistos := make(map[string]bool)
	for i := range parametros {
		parametro := &parametros[i]
		modalidad := &parametro.ModalidadGrado
		modalidad.Nombre = strings.TrimSpace(modalidad.Nombre)
		modalidad.CodigoAbreviacion = strings.TrimSpace(modalidad.CodigoAbreviacion)
		if modalidad.Id <= 0 || modalidad.Nombre == "" || modalidad.CodigoAbreviacion == "" || !modalidad.Activo ||
			parametro.TipoParametroId.CodigoAbreviacion != "MOD_TRG" || !parametro.TipoParametroId.Activo || vistos[modalidad.Id] || codigosVistos[modalidad.CodigoAbreviacion] {
			return nil, falloGrado(503, "Catálogo de modalidades inconsistente")
		}
		vistos[modalidad.Id] = true
		codigosVistos[modalidad.CodigoAbreviacion] = true
		modalidades = append(modalidades, *modalidad)
	}
	sort.Slice(modalidades, func(i, j int) bool {
		if modalidades[i].NumeroOrden != modalidades[j].NumeroOrden {
			return modalidades[i].NumeroOrden < modalidades[j].NumeroOrden
		}
		return modalidades[i].Nombre < modalidades[j].Nombre
	})
	return modalidades, nil
}

func ValidarModalidadGrado(ctx context.Context, auth string, terceroID int, codigo string) error {
	codigo = strings.TrimSpace(codigo)
	if codigo == "" {
		return falloGrado(http.StatusBadRequest, "Modalidad inválida")
	}
	modalidades, err := ListarModalidadesGrado(ctx, auth, terceroID)
	if err != nil {
		return err
	}
	for _, modalidad := range modalidades {
		if modalidad.CodigoAbreviacion == codigo {
			return nil
		}
	}
	return falloGrado(http.StatusConflict, "La modalidad seleccionada ya no está activa")
}

type tipoLugarGrado struct {
	Id     int    `json:"Id"`
	Nombre string `json:"Nombre"`
	Activo bool   `json:"Activo"`
}

type lugarGrado struct {
	Id          int            `json:"Id"`
	Nombre      string         `json:"Nombre"`
	TipoLugarId tipoLugarGrado `json:"TipoLugarId"`
	Activo      bool           `json:"Activo"`
}

type relacionLugarGrado struct {
	Id           int        `json:"Id"`
	LugarPadreId lugarGrado `json:"LugarPadreId"`
	LugarHijoId  lugarGrado `json:"LugarHijoId"`
	Activo       bool       `json:"Activo"`
}

func nombreCatalogoGrado(valor string) string {
	valor = strings.ToLower(strings.TrimSpace(valor))
	var limpio strings.Builder
	for _, r := range valor {
		switch r {
		case 'á', 'à', 'ä', 'â':
			r = 'a'
		case 'é', 'è', 'ë', 'ê':
			r = 'e'
		case 'í', 'ì', 'ï', 'î':
			r = 'i'
		case 'ó', 'ò', 'ö', 'ô':
			r = 'o'
		case 'ú', 'ù', 'ü', 'û':
			r = 'u'
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			limpio.WriteRune(r)
		}
	}
	return limpio.String()
}

type catalogoUbicacionGrado struct {
	base   string
	tipoID map[string]int
}

func nuevoCatalogoUbicacionGrado(ctx context.Context) (*catalogoUbicacionGrado, error) {
	base, err := baseGrado("UrlUbicacionesCrud")
	if err != nil {
		return nil, err
	}
	var tipos []tipoLugarGrado
	catalogo := &catalogoUbicacionGrado{base: base, tipoID: make(map[string]int)}
	if err := catalogo.consultar(ctx, "tipo_lugar?"+url.Values{"query": {"Activo:true"}, "limit": {"-1"}}.Encode(), &tipos); err != nil {
		return nil, falloGrado(http.StatusServiceUnavailable, "No se pudieron resolver los tipos de ubicación")
	}
	for _, tipo := range tipos {
		nombre := nombreCatalogoGrado(tipo.Nombre)
		if tipo.Id > 0 && tipo.Activo && (nombre == "pais" || nombre == "departamento" || nombre == "ciudad") {
			if catalogo.tipoID[nombre] != 0 {
				return nil, falloGrado(http.StatusServiceUnavailable, "Tipos de ubicación ambiguos")
			}
			catalogo.tipoID[nombre] = tipo.Id
		}
	}
	if catalogo.tipoID["pais"] == 0 || catalogo.tipoID["departamento"] == 0 || catalogo.tipoID["ciudad"] == 0 {
		return nil, falloGrado(http.StatusServiceUnavailable, "Jerarquía de ubicaciones incompleta")
	}
	return catalogo, nil
}

func (c *catalogoUbicacionGrado) consultar(ctx context.Context, ruta string, destino interface{}) error {
	var raw json.RawMessage
	if _, err := request.GetWithContext(ctx, c.base+ruta, &raw); err != nil {
		return err
	}
	return json.Unmarshal(raw, destino)
}

func (c *catalogoUbicacionGrado) relaciones(ctx context.Context, padreID, hijoTipoID int) ([]relacionLugarGrado, error) {
	q := url.Values{"query": {"LugarPadreId.Id:" + strconv.Itoa(padreID) + ",LugarHijoId.TipoLugarId.Id:" + strconv.Itoa(hijoTipoID) + ",Activo:true,LugarPadreId.Activo:true,LugarHijoId.Activo:true"}, "limit": {"-1"}}
	var relaciones []relacionLugarGrado
	if err := c.consultar(ctx, "relacion_lugares?"+q.Encode(), &relaciones); err != nil {
		return nil, err
	}
	return relaciones, nil
}

func ListarPaisesExpedicionGrado(ctx context.Context, auth string, terceroID int) ([]models.PaisExpedicionGrado, error) {
	user, err := resolverEstudiante(ctx, auth, terceroID)
	if err != nil {
		return nil, err
	}
	catalogo, err := nuevoCatalogoUbicacionGrado(user.Ctx)
	if err != nil {
		return nil, err
	}
	var paises []lugarGrado
	qPais := url.Values{"query": {"TipoLugarId.Id:" + strconv.Itoa(catalogo.tipoID["pais"]) + ",Activo:true"}, "limit": {"-1"}}
	if err := catalogo.consultar(user.Ctx, "lugar?"+qPais.Encode(), &paises); err != nil {
		return nil, falloGrado(http.StatusServiceUnavailable, "No se pudieron consultar los países")
	}
	resultado := make([]models.PaisExpedicionGrado, 0, len(paises))
	vistos := make(map[int]bool, len(paises))
	for _, pais := range paises {
		pais.Nombre = strings.TrimSpace(pais.Nombre)
		if pais.Id <= 0 || pais.Nombre == "" || !pais.Activo || pais.TipoLugarId.Id != catalogo.tipoID["pais"] || vistos[pais.Id] {
			return nil, falloGrado(http.StatusServiceUnavailable, "Catálogo de países inconsistente")
		}
		vistos[pais.Id] = true
		resultado = append(resultado, models.PaisExpedicionGrado{Id: pais.Id, Nombre: pais.Nombre})
	}
	if len(resultado) == 0 {
		return nil, falloGrado(http.StatusServiceUnavailable, "No hay países configurados")
	}
	sort.Slice(resultado, func(i, j int) bool { return resultado[i].Nombre < resultado[j].Nombre })
	return resultado, nil
}

func ListarDepartamentosExpedicionGrado(ctx context.Context, auth string, terceroID, paisID int) ([]models.DepartamentoExpedicionGrado, error) {
	if paisID <= 0 {
		return nil, falloGrado(http.StatusBadRequest, "País requerido")
	}
	user, err := resolverEstudiante(ctx, auth, terceroID)
	if err != nil {
		return nil, err
	}
	catalogo, err := nuevoCatalogoUbicacionGrado(user.Ctx)
	if err != nil {
		return nil, err
	}
	departamentos, err := catalogo.relaciones(user.Ctx, paisID, catalogo.tipoID["departamento"])
	if err != nil {
		return nil, falloGrado(http.StatusServiceUnavailable, "No se pudieron consultar los departamentos del país")
	}
	resultado := make([]models.DepartamentoExpedicionGrado, 0, len(departamentos))
	vistos := make(map[int]bool, len(departamentos))
	for _, relacion := range departamentos {
		pais := relacion.LugarPadreId
		departamento := relacion.LugarHijoId
		pais.Nombre = strings.TrimSpace(pais.Nombre)
		departamento.Nombre = strings.TrimSpace(departamento.Nombre)
		if !relacion.Activo || pais.Id != paisID || !pais.Activo || pais.TipoLugarId.Id != catalogo.tipoID["pais"] || pais.Nombre == "" ||
			departamento.Id <= 0 || !departamento.Activo || departamento.TipoLugarId.Id != catalogo.tipoID["departamento"] || departamento.Nombre == "" || vistos[departamento.Id] {
			return nil, falloGrado(http.StatusServiceUnavailable, "Jerarquía de departamentos inconsistente")
		}
		vistos[departamento.Id] = true
		resultado = append(resultado, models.DepartamentoExpedicionGrado{Id: departamento.Id, Nombre: departamento.Nombre, PaisId: pais.Id, PaisNombre: pais.Nombre})
	}
	sort.Slice(resultado, func(i, j int) bool { return resultado[i].Nombre < resultado[j].Nombre })
	return resultado, nil
}

func ListarLugaresExpedicionGrado(ctx context.Context, auth string, terceroID, paisID, departamentoID int) ([]models.LugarExpedicionGrado, error) {
	if paisID <= 0 || departamentoID <= 0 {
		return nil, falloGrado(http.StatusBadRequest, "País y departamento requeridos")
	}
	user, err := resolverEstudiante(ctx, auth, terceroID)
	if err != nil {
		return nil, err
	}
	catalogo, err := nuevoCatalogoUbicacionGrado(user.Ctx)
	if err != nil {
		return nil, err
	}
	departamentos, err := catalogo.relaciones(user.Ctx, paisID, catalogo.tipoID["departamento"])
	if err != nil {
		return nil, falloGrado(http.StatusServiceUnavailable, "No se pudo verificar el departamento")
	}
	var pais lugarGrado
	var departamento lugarGrado
	for _, relacion := range departamentos {
		if relacion.LugarHijoId.Id == departamentoID {
			if departamento.Id != 0 {
				return nil, falloGrado(http.StatusServiceUnavailable, "Departamento ambiguo para el país")
			}
			pais, departamento = relacion.LugarPadreId, relacion.LugarHijoId
		}
	}
	if pais.Id != paisID || departamento.Id != departamentoID {
		return nil, falloGrado(http.StatusConflict, "El departamento no pertenece al país seleccionado")
	}
	ciudades, err := catalogo.relaciones(user.Ctx, departamentoID, catalogo.tipoID["ciudad"])
	if err != nil {
		return nil, falloGrado(http.StatusServiceUnavailable, "No se pudieron consultar las ciudades del departamento")
	}
	resultado := make([]models.LugarExpedicionGrado, 0, len(ciudades))
	vistos := make(map[int]models.LugarExpedicionGrado, len(ciudades))
	for _, relacion := range ciudades {
		ciudad := relacion.LugarHijoId
		ciudad.Nombre = strings.TrimSpace(ciudad.Nombre)
		if !relacion.Activo || relacion.LugarPadreId.Id != departamentoID || ciudad.Id <= 0 || !ciudad.Activo || ciudad.TipoLugarId.Id != catalogo.tipoID["ciudad"] || ciudad.Nombre == "" {
			return nil, falloGrado(http.StatusServiceUnavailable, "Jerarquía de ciudades inconsistente")
		}
		lugar := models.LugarExpedicionGrado{Id: ciudad.Id, Nombre: ciudad.Nombre, DepartamentoId: departamento.Id, DepartamentoNombre: strings.TrimSpace(departamento.Nombre), PaisId: pais.Id, PaisNombre: strings.TrimSpace(pais.Nombre)}
		if anterior, existe := vistos[ciudad.Id]; existe {
			if anterior != lugar {
				return nil, falloGrado(http.StatusServiceUnavailable, "Ciudad asociada a jerarquías diferentes")
			}
			continue
		}
		vistos[ciudad.Id] = lugar
		resultado = append(resultado, lugar)
	}
	sort.Slice(resultado, func(i, j int) bool { return resultado[i].Nombre < resultado[j].Nombre })
	return resultado, nil
}

func ValidarLugarExpedicionGrado(ctx context.Context, auth string, terceroID, lugarID int) (*models.LugarExpedicionGrado, error) {
	if lugarID <= 0 {
		return nil, falloGrado(http.StatusBadRequest, "Lugar de expedición inválido")
	}
	user, err := resolverEstudiante(ctx, auth, terceroID)
	if err != nil {
		return nil, err
	}
	return resolverLugarExpedicionGrado(user.Ctx, lugarID)
}

func resolverLugarExpedicionGrado(ctx context.Context, lugarID int) (*models.LugarExpedicionGrado, error) {
	if lugarID <= 0 {
		return nil, falloGrado(http.StatusBadRequest, "Lugar de expedición inválido")
	}
	catalogo, err := nuevoCatalogoUbicacionGrado(ctx)
	if err != nil {
		return nil, err
	}
	qCiudad := url.Values{"query": {"LugarHijoId.Id:" + strconv.Itoa(lugarID) + ",LugarPadreId.TipoLugarId.Id:" + strconv.Itoa(catalogo.tipoID["departamento"]) + ",LugarHijoId.TipoLugarId.Id:" + strconv.Itoa(catalogo.tipoID["ciudad"]) + ",Activo:true,LugarPadreId.Activo:true,LugarHijoId.Activo:true"}, "limit": {"-1"}}
	var ciudades []relacionLugarGrado
	if err := catalogo.consultar(ctx, "relacion_lugares?"+qCiudad.Encode(), &ciudades); err != nil {
		return nil, falloGrado(http.StatusServiceUnavailable, "No se pudo revalidar la ciudad de expedición")
	}
	if len(ciudades) == 0 {
		return nil, falloGrado(http.StatusConflict, "La ciudad de expedición ya no tiene una jerarquía única")
	}
	ciudadRelacion := ciudades[0]
	departamento := ciudadRelacion.LugarPadreId
	ciudad := ciudadRelacion.LugarHijoId
	for _, relacion := range ciudades[1:] {
		if relacion.LugarPadreId.Id != departamento.Id || relacion.LugarHijoId.Id != ciudad.Id || strings.TrimSpace(relacion.LugarHijoId.Nombre) != strings.TrimSpace(ciudad.Nombre) {
			return nil, falloGrado(http.StatusConflict, "La ciudad de expedición ya no tiene una jerarquía única")
		}
	}
	qDepartamento := url.Values{"query": {"LugarHijoId.Id:" + strconv.Itoa(departamento.Id) + ",LugarPadreId.TipoLugarId.Id:" + strconv.Itoa(catalogo.tipoID["pais"]) + ",LugarHijoId.TipoLugarId.Id:" + strconv.Itoa(catalogo.tipoID["departamento"]) + ",Activo:true,LugarPadreId.Activo:true,LugarHijoId.Activo:true"}, "limit": {"-1"}}
	var departamentos []relacionLugarGrado
	if err := catalogo.consultar(ctx, "relacion_lugares?"+qDepartamento.Encode(), &departamentos); err != nil {
		return nil, falloGrado(http.StatusServiceUnavailable, "No se pudo revalidar el departamento de expedición")
	}
	if len(departamentos) == 0 || departamentos[0].LugarHijoId.Id != departamento.Id {
		return nil, falloGrado(http.StatusConflict, "El departamento de expedición ya no tiene una jerarquía única")
	}
	for _, relacion := range departamentos[1:] {
		if relacion.LugarPadreId.Id != departamentos[0].LugarPadreId.Id || relacion.LugarHijoId.Id != departamento.Id {
			return nil, falloGrado(http.StatusConflict, "El departamento de expedición ya no tiene una jerarquía única")
		}
	}
	pais := departamentos[0].LugarPadreId
	if !ciudadRelacion.Activo || !ciudad.Activo || ciudad.TipoLugarId.Id != catalogo.tipoID["ciudad"] ||
		!departamentos[0].Activo || !departamento.Activo || departamento.TipoLugarId.Id != catalogo.tipoID["departamento"] ||
		!pais.Activo || pais.TipoLugarId.Id != catalogo.tipoID["pais"] || strings.TrimSpace(ciudad.Nombre) == "" || strings.TrimSpace(departamento.Nombre) == "" || strings.TrimSpace(pais.Nombre) == "" {
		return nil, falloGrado(http.StatusServiceUnavailable, "Jerarquía del lugar de expedición inconsistente")
	}
	return &models.LugarExpedicionGrado{Id: ciudad.Id, Nombre: strings.TrimSpace(ciudad.Nombre), DepartamentoId: departamento.Id, DepartamentoNombre: strings.TrimSpace(departamento.Nombre), PaisId: pais.Id, PaisNombre: strings.TrimSpace(pais.Nombre)}, nil
}
