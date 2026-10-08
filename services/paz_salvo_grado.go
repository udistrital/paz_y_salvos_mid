package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/server/web"
	"github.com/udistrital/paz_y_salvos_mid/models"
	"github.com/udistrital/utils_oas/v2/request"
)

var rolesCheckGrado = map[string][]string{
	"TPS_COORDINACION": {"COORDINADOR", "ASIS_PROYECTO", "CONTRATISTA"},
	"TPS_FINANCIERO":   {"ASIS_FINANCIERA"},
	"TPS_BIBLIOTECA":   {"BIBLIOTECA"},
	"TPS_LABORATORIOS": {"LABORATORIOS"},
	"TPS_BIENESTAR":    {"ADMIN_BIENESTAR"},
	"TPS_URELINTER":    {"URELINTER"},
	"TPS_EXTENSION":    {"EXTENSION"},
	"TPS_SECRETARIA":   {rolSecretariaAcademica},
}

var tipoCheckPorPerfilGrado = map[string]string{
	"COORDINADOR": "TPS_COORDINACION", "ASIS_PROYECTO": "TPS_COORDINACION", "CONTRATISTA": "TPS_COORDINACION",
	"ASIS_FINANCIERA": "TPS_FINANCIERO", "BIBLIOTECA": "TPS_BIBLIOTECA",
	"LABORATORIOS": "TPS_LABORATORIOS", "ADMIN_BIENESTAR": "TPS_BIENESTAR",
	"URELINTER": "TPS_URELINTER", "EXTENSION": "TPS_EXTENSION", rolSecretariaAcademica: "TPS_SECRETARIA",
}

var contextoPorPerfilGrado = map[string]string{
	"ESTUDIANTE": "propio", "ASIS_PROYECTO": "programas", "CONTRATISTA": "programas",
	"COORDINADOR": "programas", rolSecretariaAcademica: "facultad", "LABORATORIOS": "laboratorios",
	"ADMIN_SGA": "global", "ASIS_FINANCIERA": "global", "BIBLIOTECA": "global",
	"ADMIN_BIENESTAR": "global", "URELINTER": "global", "ADMISIONES_REG": "global", "EXTENSION": "global",
}

type identidadPazSalvoGrado struct {
	TerceroID int
	Documento string
	Roles     []string
	Ctx       context.Context
}

func resolverIdentidadPazSalvoGrado(ctx context.Context, autorizacion string) (*identidadPazSalvoGrado, error) {
	ctxAutenticado := ctxAutorizado(ctx, autorizacion)
	usuario, err := sesionAutenticadaGrado(ctxAutenticado, autorizacion)
	if err != nil {
		return nil, err
	}
	roles, err := rolesUsuarioGrado(usuario.Role)
	if err != nil {
		return nil, err
	}
	tercero, err := terceroPorDocumentoGrado(ctxAutenticado, usuario.Documento)
	if err != nil {
		return nil, err
	}
	return &identidadPazSalvoGrado{TerceroID: tercero, Documento: usuario.Documento, Roles: roles, Ctx: ctxAutenticado}, nil
}

func checksAutorizadosGrado(roles []string) []string {
	resultado := make([]string, 0)
	for codigo, permitidos := range rolesCheckGrado {
		for _, rol := range permitidos {
			if contieneRolGrado(roles, rol) {
				resultado = append(resultado, codigo)
				break
			}
		}
	}
	sort.Strings(resultado)
	return resultado
}

func contieneTexto(valores []string, esperado string) bool {
	for _, valor := range valores {
		if valor == esperado {
			return true
		}
	}
	return false
}

func homologarProyectosGrado(ctx context.Context, codigos []string) (map[int]bool, error) {
	ws, err := baseGrado("UrlcrudWSO2")
	if err != nil {
		return nil, err
	}
	homologacion := strings.Trim(web.AppConfig.DefaultString("NscrudHomologacion", ""), "/")
	if homologacion == "" {
		return nil, falloGrado(http.StatusServiceUnavailable, "Homologación institucional no configurada")
	}
	resultado := make(map[int]bool)
	for _, codigo := range codigos {
		var respuesta models.HomologacionResponse
		if _, err := request.GetWithContext(externalRequestContext(ctx), ws+homologacion+"/proyecto_curricular_cod_proyecto/"+url.PathEscape(codigo), &respuesta); err != nil {
			return nil, falloGrado(http.StatusServiceUnavailable, "No se pudo homologar el alcance del funcionario")
		}
		if id := respuesta.OikosId(); id > 0 {
			resultado[id] = true
		}
	}
	return resultado, nil
}

func proyectosCoordinacionGrado(actor *identidadPazSalvoGrado, perfil string) (map[int]bool, error) {
	ws, err := baseGrado("UrlcrudWSO2")
	if err != nil {
		return nil, err
	}
	academica := strings.Trim(web.AppConfig.DefaultString("NscrudAcademica", ""), "/")
	if academica == "" {
		return nil, falloGrado(http.StatusServiceUnavailable, "Académica institucional no configurada")
	}
	codigos := make(map[string]bool)
	ctx := externalRequestContext(actor.Ctx)
	if perfil == "ASIS_PROYECTO" || perfil == "CONTRATISTA" {
		var respuesta models.AsistenteProyectoResponse
		if _, err := request.GetWithContext(ctx, ws+academica+"/asistente_proyecto/"+url.PathEscape(actor.Documento), &respuesta); err != nil {
			return nil, falloGrado(http.StatusServiceUnavailable, "No se pudo verificar el alcance de Asistente de Proyecto")
		}
		for _, proyecto := range respuesta.Asistente.Proyectos {
			if codigo := strings.TrimSpace(proyecto.Proyecto); codigo != "" {
				codigos[codigo] = true
			}
		}
	}
	if perfil == "COORDINADOR" {
		var respuesta models.CoordinadorCarreraResponse
		if _, err := request.GetWithContext(ctx, ws+academica+"/coordinador_carrera_snies/"+url.PathEscape(actor.Documento), &respuesta); err != nil {
			return nil, falloGrado(http.StatusServiceUnavailable, "No se pudo verificar el alcance de Coordinación")
		}
		for _, proyecto := range respuesta.CoordinadorCollection.Coordinadores {
			if codigo := strings.TrimSpace(proyecto.CodigoCondor); codigo != "" {
				codigos[codigo] = true
			}
		}
	}
	lista := make([]string, 0, len(codigos))
	for codigo := range codigos {
		lista = append(lista, codigo)
	}
	return homologarProyectosGrado(actor.Ctx, lista)
}

func relacionFacultadDependenciaGrado(ctx context.Context, dependencia int) (*models.DependenciaPadreOikos, error) {
	base, err := baseGrado("UrlcrudOikos")
	if err != nil {
		return nil, err
	}
	q := url.Values{"query": {"Hija:" + strconv.Itoa(dependencia)}, "limit": {"0"}}
	var raw json.RawMessage
	if _, err := request.GetWithContext(ctx, base+"dependencia_padre/?"+q.Encode(), &raw); err != nil {
		return nil, falloGrado(http.StatusServiceUnavailable, "No se pudo verificar la facultad de la solicitud")
	}
	relaciones, err := listaGrado[models.DependenciaPadreOikos](raw)
	if err != nil || len(relaciones) != 1 || relaciones[0].Hija.Id != dependencia || relaciones[0].Padre.Id <= 0 {
		return nil, falloGrado(http.StatusServiceUnavailable, "Facultad de la solicitud ausente o ambigua")
	}
	return &relaciones[0], nil
}

func facultadDependenciaGrado(ctx context.Context, dependencia int) (int, error) {
	relacion, err := relacionFacultadDependenciaGrado(ctx, dependencia)
	if err != nil {
		return 0, err
	}
	return relacion.Padre.Id, nil
}

func facultadesLaboratoriosGrado(actor *identidadPazSalvoGrado) (map[int]bool, error) {
	core, err := baseGrado("UrlcrudCore")
	if err != nil {
		return nil, err
	}
	fecha := time.Now().Format("2006-01-02")
	q := url.Values{"query": {"TerceroId:" + strconv.Itoa(actor.TerceroID) + ",FechaFin__gte:" + fecha + ",FechaInicio__lte:" + fecha}, "limit": {"0"}}
	var jefaturas []models.JefeDependencia
	if _, err := request.GetWithContext(actor.Ctx, core+"jefe_dependencia?"+q.Encode(), &jefaturas); err != nil {
		return nil, falloGrado(http.StatusServiceUnavailable, "No se pudo verificar el alcance de Laboratorios")
	}
	facultades := make(map[int]bool)
	for _, jefatura := range jefaturas {
		if jefatura.DependenciaId <= 0 {
			continue
		}
		facultad, err := facultadDependenciaGrado(actor.Ctx, jefatura.DependenciaId)
		if err == nil {
			facultades[facultad] = true
		}
	}
	return facultades, nil
}

func validarPerfilPazSalvoGrado(actor *identidadPazSalvoGrado, perfil string) (string, error) {
	perfil = strings.ToUpper(strings.TrimSpace(perfil))
	contexto := contextoPorPerfilGrado[perfil]
	if contexto == "" {
		return "", falloGrado(http.StatusForbidden, "El perfil seleccionado no está acreditado para la sesión")
	}
	// El contexto propio no concede privilegios: cada recurso se valida contra
	// el tercero autenticado. Los perfiles gestores sí deben venir en la sesión.
	if contexto != "propio" && !contieneRolGrado(actor.Roles, perfil) {
		return "", falloGrado(http.StatusForbidden, "El perfil seleccionado no está acreditado para la sesión")
	}
	return perfil, nil
}

func dependenciasPerfilPazSalvoGrado(actor *identidadPazSalvoGrado, perfil string) ([]int, error) {
	var dependencias map[int]bool
	switch contextoPorPerfilGrado[perfil] {
	case "programas":
		proyectos, err := proyectosCoordinacionGrado(actor, perfil)
		if err != nil {
			return nil, err
		}
		dependencias = proyectos
	case "laboratorios":
		facultades, err := facultadesLaboratoriosGrado(actor)
		if err != nil {
			return nil, err
		}
		ids := make([]int, 0, len(facultades))
		for id := range facultades {
			ids = append(ids, id)
		}
		return dependenciasFacultadesGrado(actor.Ctx, ids)
	case "facultad":
		facultades, err := facultadesSecretariaGrado(actor.Ctx, actor.Documento)
		if err != nil {
			return nil, err
		}
		return dependenciasFacultadesGrado(actor.Ctx, facultades)
	case "propio", "global":
		return nil, nil
	default:
		return nil, falloGrado(http.StatusForbidden, "El perfil no tiene un alcance de consulta válido")
	}
	resultado := make([]int, 0, len(dependencias))
	for id := range dependencias {
		resultado = append(resultado, id)
	}
	sort.Ints(resultado)
	if len(resultado) == 0 {
		return nil, falloGrado(http.StatusForbidden, "El funcionario no tiene dependencias asignadas")
	}
	return resultado, nil
}

func autorizarAlcancePerfilPazSalvoGrado(actor *identidadPazSalvoGrado, perfil string, solicitud models.SolicitudGrado) error {
	if contextoPorPerfilGrado[perfil] == "propio" {
		if solicitud.TerceroId != actor.TerceroID {
			return falloGrado(http.StatusForbidden, "La solicitud no pertenece al estudiante autenticado")
		}
		return nil
	}
	dependencias, err := dependenciasPerfilPazSalvoGrado(actor, perfil)
	if err != nil {
		return err
	}
	if dependencias == nil {
		return nil
	}
	for _, dependencia := range dependencias {
		if dependencia == solicitud.DependenciaOikosId {
			return nil
		}
	}
	return falloGrado(http.StatusForbidden, "La solicitud está fuera del alcance del perfil seleccionado")
}

func filtrarDependenciasPorFacultad(ctx context.Context, alcance []int, facultadID int) ([]int, error) {
	if facultadID == 0 {
		return alcance, nil
	}
	dependencias, err := dependenciasFacultadesGrado(ctx, []int{facultadID})
	if err != nil {
		return nil, err
	}
	if alcance == nil {
		return dependencias, nil
	}
	permitidas := make(map[int]bool, len(alcance))
	for _, id := range alcance {
		permitidas[id] = true
	}
	resultado := make([]int, 0, len(dependencias))
	for _, id := range dependencias {
		if permitidas[id] {
			resultado = append(resultado, id)
		}
	}
	return resultado, nil
}

func filtrosPazSalvosGrado(actor *identidadPazSalvoGrado, dependencias []int) (*models.FiltrosPazSalvoGrado, error) {
	parametros, err := baseGrado("UrlcrudParametros")
	if err != nil {
		return nil, err
	}
	var raw json.RawMessage
	qPeriodos := url.Values{"query": {"CodigoAbreviacion:PA,Activo:true"}, "limit": {"0"}, "sortby": {"Id"}, "order": {"desc"}}
	if _, err := request.GetWithContext(actor.Ctx, parametros+"periodo?"+qPeriodos.Encode(), &raw); err != nil {
		return nil, falloGrado(http.StatusServiceUnavailable, "No se pudieron consultar los periodos académicos")
	}
	periodos, err := listaGrado[struct {
		Id     int    `json:"Id"`
		Nombre string `json:"Nombre"`
		Activo bool   `json:"Activo"`
	}](raw)
	if err != nil {
		return nil, falloGrado(http.StatusServiceUnavailable, "Catálogo de periodos inválido")
	}
	resultado := &models.FiltrosPazSalvoGrado{
		Periodos: []models.OpcionFiltroPazSalvoGrado{}, Facultades: []models.OpcionFiltroPazSalvoGrado{},
		Programas: []models.ProgramaFiltroPazSalvoGrado{},
	}
	for _, periodo := range periodos {
		if periodo.Id <= 0 || !periodo.Activo || strings.TrimSpace(periodo.Nombre) == "" {
			return nil, falloGrado(http.StatusServiceUnavailable, "Catálogo de periodos inconsistente")
		}
		resultado.Periodos = append(resultado.Periodos, models.OpcionFiltroPazSalvoGrado{Id: periodo.Id, Nombre: strings.TrimSpace(periodo.Nombre)})
	}

	proyectosURL, err := baseGrado("UrlProyectoAcademico")
	if err != nil {
		return nil, err
	}
	type programaCatalogo struct {
		Id            int    `json:"Id"`
		Nombre        string `json:"Nombre"`
		DependenciaId int    `json:"DependenciaId"`
		Activo        bool   `json:"Activo"`
	}
	programas := make([]programaCatalogo, 0)
	consultar := func(dependencia int) error {
		query := "Activo:true"
		if dependencia > 0 {
			query = "DependenciaId:" + strconv.Itoa(dependencia) + ",Activo:true"
		}
		q := url.Values{"query": {query}, "limit": {"0"}}
		var respuesta json.RawMessage
		if _, err := request.GetWithContext(actor.Ctx, proyectosURL+"proyecto_academico_institucion?"+q.Encode(), &respuesta); err != nil {
			return falloGrado(http.StatusServiceUnavailable, "No se pudieron consultar los programas autorizados")
		}
		lista, err := listaGrado[programaCatalogo](respuesta)
		if err != nil {
			return falloGrado(http.StatusServiceUnavailable, "Catálogo de programas inválido")
		}
		for _, programa := range lista {
			if dependencia == 0 || programa.DependenciaId == dependencia {
				programas = append(programas, programa)
			}
		}
		return nil
	}
	if dependencias == nil {
		if err := consultar(0); err != nil {
			return nil, err
		}
	} else {
		for _, dependencia := range dependencias {
			if err := consultar(dependencia); err != nil {
				return nil, err
			}
		}
	}
	dependenciasPrograma := make(map[int]bool)
	for _, programa := range programas {
		if programa.Id > 0 && programa.DependenciaId > 0 && programa.Activo && strings.TrimSpace(programa.Nombre) != "" {
			dependenciasPrograma[programa.DependenciaId] = true
		}
	}
	oikos, err := baseGrado("UrlcrudOikos")
	if err != nil {
		return nil, err
	}
	facultadesValidas, err := facultadesOikosGrado(actor.Ctx, oikos)
	if err != nil {
		return nil, err
	}
	var relacionesRaw json.RawMessage
	if _, err := request.GetWithContext(actor.Ctx, oikos+"dependencia_padre/?limit=0", &relacionesRaw); err != nil {
		return nil, falloGrado(http.StatusServiceUnavailable, "No se pudieron consultar las facultades autorizadas")
	}
	relaciones, err := listaGrado[models.DependenciaPadreOikos](relacionesRaw)
	if err != nil {
		return nil, falloGrado(http.StatusServiceUnavailable, "Jerarquía de facultades inválida")
	}
	padresPorDependencia := make(map[int]models.DependenciaOikos, len(dependenciasPrograma))
	for _, relacion := range relaciones {
		if !dependenciasPrograma[relacion.Hija.Id] || !facultadesValidas[relacion.Padre.Id] {
			continue
		}
		if existente, ok := padresPorDependencia[relacion.Hija.Id]; ok && existente.Id != relacion.Padre.Id {
			return nil, falloGrado(http.StatusServiceUnavailable, "Facultad de programa ambigua")
		}
		padresPorDependencia[relacion.Hija.Id] = relacion.Padre
	}
	facultades := make(map[int]string)
	programasVistos := make(map[int]bool)
	for _, programa := range programas {
		if programa.Id <= 0 || programa.DependenciaId <= 0 || !programa.Activo || strings.TrimSpace(programa.Nombre) == "" || programasVistos[programa.Id] {
			continue
		}
		facultad, ok := padresPorDependencia[programa.DependenciaId]
		if !ok {
			continue
		}
		nombreFacultad := strings.TrimSpace(facultad.Nombre)
		if nombreFacultad == "" {
			continue
		}
		programasVistos[programa.Id] = true
		facultades[facultad.Id] = nombreFacultad
		resultado.Programas = append(resultado.Programas, models.ProgramaFiltroPazSalvoGrado{
			Id: programa.Id, Nombre: strings.TrimSpace(programa.Nombre), DependenciaId: programa.DependenciaId, FacultadId: facultad.Id,
		})
	}
	for id, nombre := range facultades {
		resultado.Facultades = append(resultado.Facultades, models.OpcionFiltroPazSalvoGrado{Id: id, Nombre: nombre})
	}
	sort.Slice(resultado.Facultades, func(i, j int) bool { return resultado.Facultades[i].Nombre < resultado.Facultades[j].Nombre })
	sort.Slice(resultado.Programas, func(i, j int) bool { return resultado.Programas[i].Nombre < resultado.Programas[j].Nombre })
	return resultado, nil
}

func facultadesOikosGrado(ctx context.Context, oikos string) (map[int]bool, error) {
	qTipo := url.Values{"query": {"Nombre:FACULTAD,Activo:true"}, "limit": {"0"}}
	var tipoRaw json.RawMessage
	if _, err := request.GetWithContext(ctx, oikos+"tipo_dependencia/?"+qTipo.Encode(), &tipoRaw); err != nil {
		return nil, falloGrado(http.StatusServiceUnavailable, "No se pudo verificar el tipo de dependencia Facultad")
	}
	tipos, err := listaGrado[struct {
		Id     int    `json:"Id"`
		Nombre string `json:"Nombre"`
	}](tipoRaw)
	if err != nil || len(tipos) != 1 || tipos[0].Id <= 0 || !strings.EqualFold(strings.TrimSpace(tipos[0].Nombre), "FACULTAD") {
		return nil, falloGrado(http.StatusServiceUnavailable, "Tipo de dependencia Facultad ausente o ambiguo")
	}

	qAsociaciones := url.Values{"query": {"TipoDependenciaId.Id:" + strconv.Itoa(tipos[0].Id) + ",Activo:true"}, "limit": {"0"}}
	var asociacionesRaw json.RawMessage
	if _, err := request.GetWithContext(ctx, oikos+"dependencia_tipo_dependencia/?"+qAsociaciones.Encode(), &asociacionesRaw); err != nil {
		return nil, falloGrado(http.StatusServiceUnavailable, "No se pudieron verificar las dependencias Facultad")
	}
	asociaciones, err := listaGrado[struct {
		DependenciaId struct {
			Id int `json:"Id"`
		} `json:"DependenciaId"`
		TipoDependenciaId struct {
			Id int `json:"Id"`
		} `json:"TipoDependenciaId"`
	}](asociacionesRaw)
	if err != nil {
		return nil, falloGrado(http.StatusServiceUnavailable, "Catálogo de dependencias Facultad inválido")
	}
	facultades := make(map[int]bool)
	for _, asociacion := range asociaciones {
		if asociacion.TipoDependenciaId.Id == tipos[0].Id && asociacion.DependenciaId.Id > 0 {
			facultades[asociacion.DependenciaId.Id] = true
		}
	}
	if len(facultades) == 0 {
		return nil, falloGrado(http.StatusServiceUnavailable, "No hay dependencias Facultad activas en Oikos")
	}
	return facultades, nil
}

func FiltrosPazSalvosGrado(ctx context.Context, autorizacion, codigo, perfil string) (*models.FiltrosPazSalvoGrado, error) {
	codigo = strings.ToUpper(strings.TrimSpace(codigo))
	actor, err := resolverIdentidadPazSalvoGrado(ctx, autorizacion)
	if err != nil {
		return nil, err
	}
	perfil, err = validarPerfilPazSalvoGrado(actor, perfil)
	if err != nil {
		return nil, err
	}
	if codigo != "" && tipoCheckPorPerfilGrado[perfil] != codigo {
		return nil, falloGrado(http.StatusForbidden, "El perfil no gestiona este Paz y Salvo")
	}
	if contextoPorPerfilGrado[perfil] == "propio" {
		return &models.FiltrosPazSalvoGrado{Periodos: []models.OpcionFiltroPazSalvoGrado{}, Facultades: []models.OpcionFiltroPazSalvoGrado{}, Programas: []models.ProgramaFiltroPazSalvoGrado{}}, nil
	}
	dependencias, err := dependenciasPerfilPazSalvoGrado(actor, perfil)
	if err != nil {
		return nil, err
	}
	return filtrosPazSalvosGrado(actor, dependencias)
}

func catalogoPazSalvoGrado(ctx context.Context) (map[string]int, map[string]int, error) {
	tipos := make(map[string]int, len(codigosCheckGrado))
	for _, codigo := range codigosCheckGrado {
		id, err := resolverParametroGrado(ctx, "TIP_PAZ_SALVO", codigo)
		if err != nil {
			return nil, nil, err
		}
		tipos[codigo] = id
	}
	estados := make(map[string]int, 4)
	for _, codigo := range []string{"SG_DOC_APROBADA", "PS_PENDIENTE", "PS_APROBADO", "PS_DESAPROBADO"} {
		tipo := "EST_PAZ_SALVO"
		if codigo == "SG_DOC_APROBADA" {
			tipo = "EST_SOL_GRADO"
		}
		id, err := resolverParametroGrado(ctx, tipo, codigo)
		if err != nil {
			return nil, nil, err
		}
		estados[codigo] = id
	}
	return tipos, estados, nil
}

func enriquecerPazSalvosGrado(resultado *models.PazSalvosSolicitudGrado, tipos, estados map[string]int) {
	tiposPorID := make(map[int]string, len(tipos))
	for codigo, id := range tipos {
		tiposPorID[id] = codigo
	}
	estadosPorID := make(map[int]string, len(estados))
	for codigo, id := range estados {
		estadosPorID[id] = codigo
	}
	for i := range resultado.Checks {
		check := &resultado.Checks[i]
		check.PazSalvo.TipoCodigo = tiposPorID[check.PazSalvo.TipoPazSalvoId]
		check.EstadoActual.EstadoCodigo = estadosPorID[check.EstadoActual.EstadoPazSalvoId]
		for j := range check.Historial {
			check.Historial[j].EstadoCodigo = estadosPorID[check.Historial[j].EstadoPazSalvoId]
		}
	}
}

func enriquecerReferenciasPazSalvoGrado(ctx context.Context, resultado *models.PazSalvosSolicitudGrado) error {
	programa, err := resolverProgramaRevisionGrado(ctx, resultado.Solicitud.ProgramaAcademicoId, resultado.Solicitud.DependenciaOikosId)
	if err != nil {
		return err
	}
	periodo, err := resolverPeriodoRevisionGrado(ctx, resultado.Solicitud.PeriodoId)
	if err != nil {
		return err
	}
	resultado.Programa = programa
	resultado.Periodo = periodo
	return nil
}

func enriquecerSoportesPazSalvoGrado(ctx context.Context, resultado *models.PazSalvosSolicitudGrado, estadoAprobada int) error {
	borrador, _, err := detalleRevisionCRUD(ctx, resultado.Solicitud.Id, resultado.Solicitud.DependenciaOikosId, []int{estadoAprobada})
	if err != nil {
		return err
	}
	expediente, err := expedienteRevisionGrado(ctx, *borrador, map[string]int{"SG_DOC_APROBADA": estadoAprobada}, true)
	if err != nil {
		return err
	}
	resultado.Soportes = expediente.Soportes
	return nil
}

func consultarPazSalvosCRUD(ctx context.Context, id int, tipos map[string]int, estadoAprobada int) (*models.PazSalvosSolicitudGrado, error) {
	base, err := baseGrado("UrlCrudPazySalvos")
	if err != nil {
		return nil, err
	}
	ids := make([]int, 0, len(tipos))
	for _, tipo := range tipos {
		ids = append(ids, tipo)
	}
	sort.Ints(ids)
	textos := make([]string, len(ids))
	for i, valor := range ids {
		textos[i] = strconv.Itoa(valor)
	}
	q := url.Values{"estado_documentacion_aprobada_id": {strconv.Itoa(estadoAprobada)}, "tipos": {strings.Join(textos, ",")}}
	var respuesta models.APIResponseData[models.PazSalvosSolicitudGrado]
	status, err := request.GetWithContext(ctx, base+"solicitud-grado/"+strconv.Itoa(id)+"/paz-salvos?"+q.Encode(), &respuesta)
	if err != nil {
		if status == http.StatusNotFound || status == http.StatusConflict {
			return nil, falloGrado(status, "Los Paz y Salvos de la solicitud no están disponibles")
		}
		return nil, falloGrado(http.StatusServiceUnavailable, "No se pudieron consultar los Paz y Salvos")
	}
	if status != http.StatusOK || !respuesta.Success || respuesta.Status != http.StatusOK || respuesta.Data.Solicitud.Id != id || len(respuesta.Data.Checks) != len(tipos) {
		return nil, falloGrado(http.StatusServiceUnavailable, "Respuesta de Paz y Salvos inválida")
	}
	return &respuesta.Data, nil
}

func listarPazSalvosCRUD(ctx context.Context, tipos map[string]int, estadoAprobada int, dependencias []int, limit, offset, periodoID, programaID, terceroID int, codigoEstudiante string) (*models.PaginaPazSalvosGrado, error) {
	base, err := baseGrado("UrlCrudPazySalvos")
	if err != nil {
		return nil, err
	}
	ids := make([]int, 0, len(tipos))
	for _, tipo := range tipos {
		ids = append(ids, tipo)
	}
	sort.Ints(ids)
	tiposTexto := make([]string, len(ids))
	for i, id := range ids {
		tiposTexto[i] = strconv.Itoa(id)
	}
	q := url.Values{
		"estado_documentacion_aprobada_id": {strconv.Itoa(estadoAprobada)}, "tipos": {strings.Join(tiposTexto, ",")},
		"limit": {strconv.Itoa(limit)}, "offset": {strconv.Itoa(offset)}, "periodo_id": {strconv.Itoa(periodoID)}, "programa_id": {strconv.Itoa(programaID)},
	}
	if len(dependencias) > 0 {
		valores := make([]string, len(dependencias))
		for i, id := range dependencias {
			valores[i] = strconv.Itoa(id)
		}
		q.Set("dependencias", strings.Join(valores, ","))
	}
	if codigoEstudiante != "" {
		q.Set("codigo", codigoEstudiante)
	}
	if terceroID > 0 {
		q.Set("tercero_id", strconv.Itoa(terceroID))
	}
	var respuesta models.APIResponseData[models.PaginaPazSalvosGrado]
	status, err := request.GetWithContext(ctx, base+"solicitud-grado/paz-salvos?"+q.Encode(), &respuesta)
	if err != nil {
		return nil, falloGrado(http.StatusServiceUnavailable, "No se pudo consultar la bandeja de Paz y Salvos")
	}
	if status != http.StatusOK || !respuesta.Success || respuesta.Status != http.StatusOK || respuesta.Data.Total < len(respuesta.Data.Solicitudes) {
		return nil, falloGrado(http.StatusServiceUnavailable, "Respuesta de bandeja de Paz y Salvos inválida")
	}
	for _, solicitud := range respuesta.Data.Solicitudes {
		if solicitud.Solicitud.Id <= 0 || len(solicitud.Checks) != len(tipos) {
			return nil, falloGrado(http.StatusServiceUnavailable, "La bandeja contiene solicitudes inválidas")
		}
	}
	return &respuesta.Data, nil
}

func ListarPazSalvosGrado(ctx context.Context, autorizacion, codigo, perfil string, limit, offset, periodoID, programaID, facultadID int, codigoEstudiante string) (*models.PaginaPazSalvosGrado, error) {
	codigoEstudiante = strings.TrimSpace(codigoEstudiante)
	if limit <= 0 || limit > 100 || offset < 0 || periodoID < 0 || programaID < 0 || facultadID < 0 || len([]rune(codigoEstudiante)) > 50 {
		return nil, falloGrado(http.StatusBadRequest, "Filtros de bandeja inválidos")
	}
	codigo = strings.ToUpper(strings.TrimSpace(codigo))
	actor, err := resolverIdentidadPazSalvoGrado(ctx, autorizacion)
	if err != nil {
		return nil, err
	}
	perfil, err = validarPerfilPazSalvoGrado(actor, perfil)
	if err != nil {
		return nil, err
	}
	if codigo != "" && tipoCheckPorPerfilGrado[perfil] != codigo {
		return nil, falloGrado(http.StatusForbidden, "El perfil no gestiona este Paz y Salvo")
	}
	if codigo == "" && tipoCheckPorPerfilGrado[perfil] != "" {
		return nil, falloGrado(http.StatusBadRequest, "El perfil gestor requiere indicar su Paz y Salvo")
	}
	dependencias, err := dependenciasPerfilPazSalvoGrado(actor, perfil)
	if err != nil {
		return nil, err
	}
	dependencias, err = filtrarDependenciasPorFacultad(actor.Ctx, dependencias, facultadID)
	if err != nil {
		return nil, err
	}
	tipos, estados, err := catalogoPazSalvoGrado(actor.Ctx)
	if err != nil {
		return nil, err
	}
	if facultadID > 0 && len(dependencias) == 0 {
		return &models.PaginaPazSalvosGrado{Solicitudes: []models.PazSalvosSolicitudGrado{}, TipoGestionado: codigo, TipoGestionadoId: tipos[codigo]}, nil
	}
	terceroID := 0
	if contextoPorPerfilGrado[perfil] == "propio" {
		terceroID = actor.TerceroID
		codigoEstudiante = ""
	}
	resultado, err := listarPazSalvosCRUD(actor.Ctx, tipos, estados["SG_DOC_APROBADA"], dependencias, limit, offset, periodoID, programaID, terceroID, codigoEstudiante)
	if err != nil {
		return nil, err
	}
	for _, solicitud := range resultado.Solicitudes {
		if err := autorizarAlcancePerfilPazSalvoGrado(actor, perfil, solicitud.Solicitud); err != nil {
			return nil, falloGrado(http.StatusServiceUnavailable, "El CRUD devolvió una solicitud fuera del alcance autorizado")
		}
	}
	resultado.TipoGestionado = codigo
	if codigo != "" {
		resultado.TipoGestionadoId = tipos[codigo]
	}
	for i := range resultado.Solicitudes {
		enriquecerPazSalvosGrado(&resultado.Solicitudes[i], tipos, estados)
		if err := enriquecerReferenciasPazSalvoGrado(actor.Ctx, &resultado.Solicitudes[i]); err != nil {
			return nil, err
		}
	}
	return resultado, nil
}

func ConsultarPazSalvosGrado(ctx context.Context, autorizacion string, id int, perfil string) (*models.PazSalvosSolicitudGrado, error) {
	actor, err := resolverIdentidadPazSalvoGrado(ctx, autorizacion)
	if err != nil {
		return nil, err
	}
	perfil, err = validarPerfilPazSalvoGrado(actor, perfil)
	if err != nil {
		return nil, err
	}
	tipos, estados, err := catalogoPazSalvoGrado(actor.Ctx)
	if err != nil {
		return nil, err
	}
	resultado, err := consultarPazSalvosCRUD(actor.Ctx, id, tipos, estados["SG_DOC_APROBADA"])
	if err != nil {
		return nil, err
	}
	if err := autorizarAlcancePerfilPazSalvoGrado(actor, perfil, resultado.Solicitud); err != nil {
		return nil, err
	}
	enriquecerPazSalvosGrado(resultado, tipos, estados)
	if err := enriquecerReferenciasPazSalvoGrado(actor.Ctx, resultado); err != nil {
		return nil, err
	}
	estudiante, err := resolverEstudianteRevisionGrado(actor.Ctx, resultado.Solicitud.TerceroId, true)
	if err != nil {
		return nil, err
	}
	resultado.Estudiante = &estudiante
	if err := enriquecerSoportesPazSalvoGrado(actor.Ctx, resultado, estados["SG_DOC_APROBADA"]); err != nil {
		return nil, err
	}
	return resultado, nil
}

func DescargarSoportePazSalvoGrado(ctx context.Context, autorizacion string, id int, codigoCheck, codigoSoporte, perfil string) (*models.ArchivoSoporteGrado, error) {
	if id <= 0 {
		return nil, falloGrado(http.StatusBadRequest, "Solicitud inválida")
	}
	if _, err := codigoDocumentoGrado(codigoSoporte); err != nil {
		return nil, err
	}
	codigoCheck = strings.ToUpper(strings.TrimSpace(codigoCheck))
	actor, err := resolverIdentidadPazSalvoGrado(ctx, autorizacion)
	if err != nil {
		return nil, err
	}
	perfil, err = validarPerfilPazSalvoGrado(actor, perfil)
	if err != nil {
		return nil, err
	}
	if tipoCheckPorPerfilGrado[perfil] != codigoCheck {
		return nil, falloGrado(http.StatusForbidden, "El funcionario no puede consultar este Paz y Salvo")
	}
	tipos, estados, err := catalogoPazSalvoGrado(actor.Ctx)
	if err != nil {
		return nil, err
	}
	actual, err := consultarPazSalvosCRUD(actor.Ctx, id, tipos, estados["SG_DOC_APROBADA"])
	if err != nil {
		return nil, err
	}
	if err := autorizarAlcancePerfilPazSalvoGrado(actor, perfil, actual.Solicitud); err != nil {
		return nil, err
	}
	borrador, _, err := detalleRevisionCRUD(actor.Ctx, id, actual.Solicitud.DependenciaOikosId, []int{estados["SG_DOC_APROBADA"]})
	if err != nil {
		return nil, err
	}
	tipoID, err := resolverParametroGrado(actor.Ctx, "TIP_SOP_GRADO", codigoSoporte)
	if err != nil {
		return nil, err
	}
	for _, soporte := range borrador.Soportes {
		if soporte.TipoDocumentoId == tipoID {
			documento, err := consultarDocumentoGrado(actor.Ctx, soporte.DocumentoId)
			if err != nil {
				return nil, err
			}
			archivo, _, err := archivoDocumentoGrado(actor.Ctx, documento)
			return archivo, err
		}
	}
	return nil, falloGrado(http.StatusNotFound, "Soporte no encontrado")
}

func DecidirPazSalvoGrado(ctx context.Context, autorizacion string, id int, codigo string, entrada models.DecidirPazSalvoGrado) (*models.PazSalvosSolicitudGrado, error) {
	codigo = strings.ToUpper(strings.TrimSpace(codigo))
	actor, err := resolverIdentidadPazSalvoGrado(ctx, autorizacion)
	if err != nil {
		return nil, err
	}
	perfil, err := validarPerfilPazSalvoGrado(actor, entrada.Perfil)
	if err != nil {
		return nil, err
	}
	if tipoCheckPorPerfilGrado[perfil] != codigo {
		return nil, falloGrado(http.StatusForbidden, "El funcionario no puede decidir este Paz y Salvo")
	}
	estado := strings.ToUpper(strings.TrimSpace(entrada.Estado))
	if estado != "PS_PENDIENTE" && estado != "PS_APROBADO" && estado != "PS_DESAPROBADO" {
		return nil, falloGrado(http.StatusBadRequest, "Estado de Paz y Salvo inválido")
	}
	if (estado == "PS_PENDIENTE" || estado == "PS_DESAPROBADO") && strings.TrimSpace(entrada.Justificacion) == "" {
		return nil, falloGrado(http.StatusBadRequest, "La reapertura o desaprobación requiere justificación")
	}
	tipos, estados, err := catalogoPazSalvoGrado(actor.Ctx)
	if err != nil {
		return nil, err
	}
	actual, err := consultarPazSalvosCRUD(actor.Ctx, id, tipos, estados["SG_DOC_APROBADA"])
	if err != nil {
		return nil, err
	}
	if err := autorizarAlcancePerfilPazSalvoGrado(actor, perfil, actual.Solicitud); err != nil {
		return nil, err
	}
	if err := validarVentanaAprobacionPazSalvoGrado(actor.Ctx, actual.Solicitud); err != nil {
		return nil, err
	}
	previos := make([]int, 0, len(tipos)-1)
	for tipoCodigo, tipoID := range tipos {
		if tipoCodigo != "TPS_SECRETARIA" {
			previos = append(previos, tipoID)
		}
	}
	sort.Ints(previos)
	body := map[string]interface{}{
		"TerceroId": actor.TerceroID, "TipoPazSalvoId": tipos[codigo], "EstadoPazSalvoId": estados[estado],
		"Justificacion": strings.TrimSpace(entrada.Justificacion), "EstadoDocumentacionAprobadaId": estados["SG_DOC_APROBADA"],
		"EstadoPendienteId": estados["PS_PENDIENTE"], "EstadoAprobadoId": estados["PS_APROBADO"],
		"EstadoDesaprobadoId": estados["PS_DESAPROBADO"], "TipoSecretariaId": tipos["TPS_SECRETARIA"], "TiposPreviosId": previos,
	}
	base, err := baseGrado("UrlCrudPazySalvos")
	if err != nil {
		return nil, err
	}
	var respuesta models.APIResponseData[models.PazSalvosSolicitudGrado]
	status, err := request.PostWithContext(actor.Ctx, base+"solicitud-grado/"+strconv.Itoa(id)+"/paz-salvos", body, &respuesta)
	if err != nil {
		if status >= 400 && status < 500 {
			return nil, falloGrado(status, "La decisión de Paz y Salvo fue rechazada")
		}
		return nil, falloGrado(http.StatusServiceUnavailable, "No se pudo registrar la decisión de Paz y Salvo")
	}
	if status != http.StatusOK || !respuesta.Success || respuesta.Status != http.StatusOK || respuesta.Data.Solicitud.Id != id {
		return nil, falloGrado(http.StatusServiceUnavailable, "Confirmación de Paz y Salvo inválida")
	}
	enriquecerPazSalvosGrado(&respuesta.Data, tipos, estados)
	if err := enriquecerReferenciasPazSalvoGrado(actor.Ctx, &respuesta.Data); err != nil {
		return nil, err
	}
	if err := enriquecerSoportesPazSalvoGrado(actor.Ctx, &respuesta.Data, estados["SG_DOC_APROBADA"]); err != nil {
		return nil, err
	}
	return &respuesta.Data, nil
}
