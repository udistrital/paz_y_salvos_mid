package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	beego "github.com/beego/beego/v2/server/web"
	"github.com/udistrital/paz_y_salvos_mid/models"
	"github.com/udistrital/utils_oas/v2/request"
)

const rolSecretariaAcademica = "SECRETARIA_ACADEMICA"

var codigosCheckGrado = []string{
	"TPS_COORDINACION", "TPS_FINANCIERO", "TPS_BIBLIOTECA", "TPS_LABORATORIOS",
	"TPS_BIENESTAR", "TPS_URELINTER", "TPS_EXTENSION", "TPS_SECRETARIA",
}

type identidadSecretariaGrado struct {
	TerceroID    int
	Documento    string
	Dependencias []int
	Ctx          context.Context
}

type usuarioInfoGrado struct {
	Documento string          `json:"documento"`
	Role      json.RawMessage `json:"role"`
	Sub       string          `json:"sub"`
}

func rolesUsuarioGrado(raw json.RawMessage) ([]string, error) {
	var lista []string
	if err := json.Unmarshal(raw, &lista); err != nil {
		var texto string
		if json.Unmarshal(raw, &texto) != nil {
			return nil, falloGrado(http.StatusUnauthorized, "Roles de la sesión no verificables")
		}
		lista = strings.Split(texto, ",")
	}
	resultado := make([]string, 0, len(lista))
	for _, rol := range lista {
		rol = strings.ToUpper(strings.TrimSpace(rol))
		if rol != "" {
			resultado = append(resultado, rol)
		}
	}
	return resultado, nil
}

func contieneRolGrado(roles []string, esperado string) bool {
	for _, rol := range roles {
		if rol == esperado {
			return true
		}
	}
	return false
}

func urlUserInfoGrado() (string, error) {
	valor := strings.TrimSpace(beego.AppConfig.DefaultString("UrlUserInfo", ""))
	u, err := url.Parse(valor)
	if err != nil || u == nil {
		return "", falloGrado(http.StatusServiceUnavailable, "Servicio de identidad no configurado")
	}
	local := u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" || u.Hostname() == "::1")
	if (u.Scheme != "https" && !local) || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return "", falloGrado(http.StatusServiceUnavailable, "Servicio de identidad no configurado")
	}
	return u.String(), nil
}

func sesionAutenticadaGrado(ctx context.Context, autorizacion string) (*usuarioInfoGrado, error) {
	if !strings.HasPrefix(autorizacion, "Bearer ") || len(strings.TrimSpace(strings.TrimPrefix(autorizacion, "Bearer "))) < 10 {
		return nil, falloGrado(http.StatusUnauthorized, "Sesión no autenticada")
	}
	usuarioContexto, ok := ctx.Value("user").(string)
	if !ok || strings.TrimSpace(usuarioContexto) == "" {
		return nil, falloGrado(http.StatusUnauthorized, "La sesión no es válida")
	}
	endpoint, err := urlUserInfoGrado()
	if err != nil {
		return nil, err
	}
	var usuario usuarioInfoGrado
	if _, err := request.GetWithContext(ctx, endpoint, &usuario); err != nil {
		return nil, falloGrado(http.StatusUnauthorized, "No fue posible verificar la sesión")
	}
	usuario.Sub = strings.TrimSpace(usuario.Sub)
	usuario.Documento = strings.TrimSpace(usuario.Documento)
	if usuario.Sub == "" || usuario.Sub != strings.TrimSpace(usuarioContexto) || usuario.Documento == "" || len(usuario.Role) == 0 {
		return nil, falloGrado(http.StatusUnauthorized, "La identidad de la sesión no coincide")
	}
	return &usuario, nil
}

func terceroPorDocumentoGrado(ctx context.Context, documento string) (int, error) {
	base, err := baseGrado("UrlTercerosCrud")
	if err != nil {
		return 0, err
	}
	q := url.Values{"query": {"Numero:" + documento + ",Activo:true"}, "limit": {"0"}}
	var raw json.RawMessage
	if _, err := request.GetWithContext(ctx, base+"datos_identificacion?"+q.Encode(), &raw); err != nil {
		return 0, falloGrado(http.StatusServiceUnavailable, "No se pudo verificar la identidad del funcionario")
	}
	identificaciones, err := listaGrado[struct {
		Numero    string `json:"Numero"`
		Activo    bool   `json:"Activo"`
		TerceroId struct {
			Id int `json:"Id"`
		} `json:"TerceroId"`
	}](raw)
	if err != nil {
		return 0, falloGrado(http.StatusServiceUnavailable, "Respuesta de identidad inválida")
	}
	terceros := make(map[int]bool)
	for _, identificacion := range identificaciones {
		if identificacion.Activo && strings.TrimSpace(identificacion.Numero) == documento && identificacion.TerceroId.Id > 0 {
			terceros[identificacion.TerceroId.Id] = true
		}
	}
	if len(terceros) != 1 {
		return 0, falloGrado(http.StatusForbidden, "El documento autenticado no tiene un tercero único")
	}
	for id := range terceros {
		return id, nil
	}
	return 0, falloGrado(http.StatusForbidden, "Funcionario no verificable")
}

func facultadesSecretariaGrado(ctx context.Context, documento string) ([]int, error) {
	ws, err := baseGrado("UrlcrudWSO2")
	if err != nil {
		return nil, err
	}
	academica := strings.Trim(beego.AppConfig.DefaultString("NscrudAcademica", ""), "/")
	homologacion := strings.Trim(beego.AppConfig.DefaultString("NscrudHomologacion", ""), "/")
	if academica == "" || homologacion == "" {
		return nil, falloGrado(http.StatusServiceUnavailable, "Fuentes de alcance no configuradas")
	}
	ctxExterno := externalRequestContext(ctx)
	var respuesta models.FacultadesSecretariaResponse
	if _, err := request.GetWithContext(ctxExterno, ws+academica+"/facultad_secretaria/"+url.PathEscape(documento), &respuesta); err != nil {
		return nil, falloGrado(http.StatusServiceUnavailable, "No se pudieron consultar las facultades de Secretaría")
	}
	facultades := make(map[int]bool)
	for _, secretaria := range respuesta.Facultades.Secretarias {
		codigo := strings.TrimSpace(secretaria.Codigo)
		if codigo == "" {
			continue
		}
		var homologa models.HomologacionResponse
		if _, err := request.GetWithContext(ctxExterno, ws+homologacion+"/facultad_oikos_gedep/"+url.PathEscape(codigo), &homologa); err != nil {
			return nil, falloGrado(http.StatusServiceUnavailable, "No se pudo homologar una facultad de Secretaría")
		}
		if id := homologa.OikosId(); id > 0 {
			facultades[id] = true
		}
	}
	if len(facultades) == 0 {
		return nil, falloGrado(http.StatusForbidden, "El funcionario no tiene facultades asignadas")
	}
	resultado := make([]int, 0, len(facultades))
	for id := range facultades {
		resultado = append(resultado, id)
	}
	sort.Ints(resultado)
	return resultado, nil
}

func dependenciasFacultadesGrado(ctx context.Context, facultades []int) ([]int, error) {
	return resolverDependenciasFacultadesGrado(ctx, facultades, false)
}

func dependenciasFacultadesGradoPermitiendoVacio(ctx context.Context, facultades []int) ([]int, error) {
	return resolverDependenciasFacultadesGrado(ctx, facultades, true)
}

func resolverDependenciasFacultadesGrado(ctx context.Context, facultades []int, permitirVacio bool) ([]int, error) {
	base, err := baseOikosGradoV2()
	if err != nil {
		return nil, err
	}
	dependencias := make(map[int]bool)
	for _, facultad := range facultades {
		q := url.Values{"query": {"PadreId.Id:" + strconv.Itoa(facultad) + ",Activo:true"}, "limit": {"0"}}
		var raw json.RawMessage
		if _, err := request.GetWithContext(ctx, base+"dependencia_padre/?"+q.Encode(), &raw); err != nil {
			return nil, falloGrado(http.StatusServiceUnavailable, "No se pudo resolver el alcance Oikos de las facultades asignadas")
		}
		if permitirVacio && strings.TrimSpace(string(raw)) == "null" {
			continue
		}
		relaciones, err := listaGrado[models.DependenciaPadreOikosV2](raw)
		if err != nil {
			return nil, falloGrado(http.StatusServiceUnavailable, "Jerarquía Oikos no verificable")
		}
		for _, relacion := range relaciones {
			if relacion.Activo && relacion.PadreId.Id == facultad && relacion.PadreId.Activo &&
				relacion.HijaId.Id > 0 && relacion.HijaId.Activo {
				dependencias[relacion.HijaId.Id] = true
			}
		}
	}
	if len(dependencias) == 0 {
		if permitirVacio {
			return []int{}, nil
		}
		return nil, falloGrado(http.StatusForbidden, "Las facultades asignadas no tienen programas Oikos verificables")
	}
	resultado := make([]int, 0, len(dependencias))
	for id := range dependencias {
		resultado = append(resultado, id)
	}
	sort.Ints(resultado)
	return resultado, nil
}

func resolverSecretariaGrado(ctx context.Context, autorizacion string) (*identidadSecretariaGrado, error) {
	ctxAutenticado := ctxAutorizado(ctx, autorizacion)
	usuario, err := sesionAutenticadaGrado(ctxAutenticado, autorizacion)
	if err != nil {
		return nil, err
	}
	roles, err := rolesUsuarioGrado(usuario.Role)
	if err != nil {
		return nil, err
	}
	if !contieneRolGrado(roles, rolSecretariaAcademica) {
		return nil, falloGrado(http.StatusForbidden, "La revisión documental requiere Secretaría Académica")
	}
	terceroID, err := terceroPorDocumentoGrado(ctxAutenticado, usuario.Documento)
	if err != nil {
		return nil, err
	}
	facultades, err := facultadesSecretariaGrado(ctxAutenticado, usuario.Documento)
	if err != nil {
		return nil, err
	}
	dependencias, err := dependenciasFacultadesGrado(ctxAutenticado, facultades)
	if err != nil {
		return nil, err
	}
	return &identidadSecretariaGrado{TerceroID: terceroID, Documento: usuario.Documento, Dependencias: dependencias, Ctx: ctxAutenticado}, nil
}

func estadosSolicitudRevisionGrado(ctx context.Context) (map[string]int, error) {
	codigos := []string{"SG_BORRADOR", "SG_RADICADA", "SG_OBSERVADA", "SG_DOC_APROBADA"}
	resultado := make(map[string]int, len(codigos))
	for _, codigo := range codigos {
		id, err := resolverParametroGrado(ctx, "EST_SOL_GRADO", codigo)
		if err != nil {
			return nil, err
		}
		resultado[codigo] = id
	}
	return resultado, nil
}

func validarVentanaAprobacionPazSalvoGrado(ctx context.Context, solicitud models.SolicitudGrado) error {
	if solicitud.CalendarioEventoAprobacionId <= 0 {
		return falloGrado(http.StatusConflict, "La solicitud no tiene asociado el evento de aprobación de Paz y Salvos")
	}
	base, err := baseGrado("UrlProyectoAcademico")
	if err != nil {
		return err
	}
	q := url.Values{"query": {"Id:" + strconv.Itoa(solicitud.ProgramaAcademicoId) + ",Activo:true"}, "limit": {"0"}}
	var raw json.RawMessage
	if _, err := request.GetWithContext(ctx, base+"proyecto_academico_institucion?"+q.Encode(), &raw); err != nil {
		return falloGrado(http.StatusServiceUnavailable, "No se pudo verificar el programa de la solicitud")
	}
	programas, err := listaGrado[programaGrado](raw)
	if err != nil || len(programas) != 1 || programas[0].Id != solicitud.ProgramaAcademicoId ||
		programas[0].DependenciaId != solicitud.DependenciaOikosId || !programas[0].Activo || strings.TrimSpace(programas[0].Nombre) == "" {
		return falloGrado(http.StatusServiceUnavailable, "Programa de la solicitud ausente o ambiguo")
	}
	inscripcion, aprobacion, err := eventosGrado(ctx, &programas[0], solicitud.PeriodoId)
	if err != nil {
		return err
	}
	if aprobacion.EventoId != solicitud.CalendarioEventoAprobacionId {
		return falloGrado(http.StatusConflict, "El evento de aprobación asociado a la solicitud no coincide con el calendario del periodo y programa")
	}
	if inscripcion.EventoId != solicitud.CalendarioEventoInscripcionId {
		return falloGrado(http.StatusConflict, "La configuración de inscripción cambió; la solicitud requiere revisión")
	}
	if err := validarVentana(inscripcion, aprobacion, false); err != nil {
		if fallo, ok := err.(*ErrorInscripcionGrado); ok && fallo.Status == http.StatusConflict &&
			fallo.Mensaje == "Los tiempos de aprobación se han cerrado" {
			return falloGrado(http.StatusConflict, "Los tiempos de aprobación para el programa académico "+strings.TrimSpace(programas[0].Nombre)+" se han cerrado")
		}
		return err
	}
	return nil
}

func idsEstadosRevisionGrado(estados map[string]int) []int {
	return []int{estados["SG_BORRADOR"], estados["SG_RADICADA"], estados["SG_OBSERVADA"], estados["SG_DOC_APROBADA"]}
}

func codigoEstadoRevisionGrado(estados map[string]int, id int) string {
	for codigo, estadoID := range estados {
		if estadoID == id {
			return codigo
		}
	}
	return ""
}

func expedienteRevisionGrado(ctx context.Context, borrador models.BorradorGrado, estados map[string]int, detallado bool) (*models.SolicitudRevisionGrado, error) {
	estudiante, programa, periodo, formulario, err := enriquecerExpedienteRevisionGrado(ctx, borrador, detallado)
	if err != nil {
		return nil, err
	}
	soportes := make([]models.SoporteBorradorGrado, 0, len(borrador.Soportes))
	if detallado {
		tipos := make(map[int]string, len(tiposSoporteGrado))
		for _, definicion := range tiposSoporteGrado {
			id, err := resolverParametroGrado(ctx, "TIP_SOP_GRADO", definicion.Codigo)
			if err != nil {
				return nil, err
			}
			tipos[id] = definicion.Codigo
		}
		for _, soporte := range borrador.Soportes {
			codigo, ok := tipos[soporte.TipoDocumentoId]
			if !ok || soporte.Id <= 0 || soporte.DocumentoId <= 0 || soporte.FormularioSolicitudGradoId != borrador.Formulario.Id {
				return nil, falloGrado(http.StatusServiceUnavailable, "Expediente documental inconsistente")
			}
			documento, err := consultarDocumentoGrado(ctx, soporte.DocumentoId)
			if err != nil {
				return nil, err
			}
			soportes = append(soportes, models.SoporteBorradorGrado{
				Id: soporte.Id, FormularioId: borrador.Formulario.Id, TipoSoporte: codigo,
				DocumentoId: soporte.DocumentoId, Nombre: documento.Nombre,
			})
		}
	}
	estado := codigoEstadoRevisionGrado(estados, borrador.Historial.EstadoSolicitudId)
	if estado == "" {
		return nil, falloGrado(http.StatusServiceUnavailable, "Estado del expediente no verificable")
	}
	return &models.SolicitudRevisionGrado{
		Solicitud: borrador.Solicitud, Estudiante: estudiante, Programa: programa, Periodo: periodo, Formulario: formulario,
		Version: borrador.Formulario.Version, Estado: estado,
		Comentario: borrador.Historial.Justificacion, RadicadaEn: borrador.Formulario.FechaRadicacion,
		Soportes: soportes,
	}, nil
}

func consultarRevisionCRUD(ctx context.Context, dependenciaID int, estados []int, limit, offset, periodoID, programaID int) ([]models.BorradorGrado, error) {
	base, err := baseGrado("UrlCrudPazySalvos")
	if err != nil {
		return nil, err
	}
	valores := make([]string, len(estados))
	for i, estado := range estados {
		valores[i] = strconv.Itoa(estado)
	}
	q := url.Values{
		"dependencia_id": {strconv.Itoa(dependenciaID)}, "estados": {strings.Join(valores, ",")},
		"limit": {strconv.Itoa(limit)}, "offset": {strconv.Itoa(offset)},
	}
	if periodoID > 0 {
		q.Set("periodo_id", strconv.Itoa(periodoID))
	}
	if programaID > 0 {
		q.Set("programa_id", strconv.Itoa(programaID))
	}
	var respuesta models.APIResponseData[[]models.BorradorGrado]
	status, err := request.GetWithContext(ctx, base+"solicitud-grado/revision?"+q.Encode(), &respuesta)
	if err != nil || status != http.StatusOK || !respuesta.Success || respuesta.Status != http.StatusOK {
		return nil, falloGrado(http.StatusServiceUnavailable, "No se pudo consultar la bandeja documental")
	}
	for _, expediente := range respuesta.Data {
		if expediente.Solicitud.DependenciaOikosId != dependenciaID {
			return nil, falloGrado(http.StatusServiceUnavailable, "El CRUD devolvió una solicitud fuera del alcance solicitado")
		}
	}
	return respuesta.Data, nil
}

func consultarPaginasRevisionCRUD(ctx context.Context, dependencia, periodoID, programaID int, estados []int) ([]models.BorradorGrado, error) {
	resultado := make([]models.BorradorGrado, 0)
	vistos := make(map[int]bool)
	for offset := 0; ; offset += 100 {
		pagina, err := consultarRevisionCRUD(ctx, dependencia, estados, 100, offset, periodoID, programaID)
		if err != nil {
			return nil, err
		}
		for _, expediente := range pagina {
			if expediente.Solicitud.Id <= 0 || vistos[expediente.Solicitud.Id] {
				return nil, falloGrado(http.StatusServiceUnavailable, "Paginación documental inconsistente")
			}
			vistos[expediente.Solicitud.Id] = true
		}
		resultado = append(resultado, pagina...)
		if len(pagina) < 100 {
			return resultado, nil
		}
	}
}

func detalleRevisionCRUD(ctx context.Context, id, dependenciaID int, estados []int) (*models.BorradorGrado, int, error) {
	base, err := baseGrado("UrlCrudPazySalvos")
	if err != nil {
		return nil, 0, err
	}
	valores := make([]string, len(estados))
	for i, estado := range estados {
		valores[i] = strconv.Itoa(estado)
	}
	q := url.Values{"dependencia_id": {strconv.Itoa(dependenciaID)}, "estados": {strings.Join(valores, ",")}}
	var respuesta models.APIResponseData[models.BorradorGrado]
	status, err := request.GetWithContext(ctx, base+"solicitud-grado/revision/"+strconv.Itoa(id)+"?"+q.Encode(), &respuesta)
	if err != nil {
		return nil, status, err
	}
	if status != http.StatusOK || !respuesta.Success || respuesta.Status != http.StatusOK || respuesta.Data.Solicitud.DependenciaOikosId != dependenciaID {
		return nil, status, falloGrado(http.StatusServiceUnavailable, "Respuesta de expediente inválida")
	}
	return &respuesta.Data, status, nil
}

func buscarRevisionEnAlcance(ctx context.Context, id int, dependencias, estados []int) (*models.BorradorGrado, int, error) {
	for _, dependencia := range dependencias {
		expediente, status, err := detalleRevisionCRUD(ctx, id, dependencia, estados)
		if err == nil {
			return expediente, dependencia, nil
		}
		if status != http.StatusNotFound {
			return nil, 0, err
		}
	}
	return nil, 0, falloGrado(http.StatusNotFound, "Solicitud no encontrada en las facultades autorizadas")
}

func ListarRevisionDocumentalGrado(ctx context.Context, auth string, limit, offset, periodoID, programaID int, estado, texto string) (*models.PaginaRevisionGrado, error) {
	texto = strings.TrimSpace(texto)
	estado = strings.TrimSpace(estado)
	if limit <= 0 || limit > 100 || offset < 0 || periodoID < 0 || programaID < 0 || len(texto) > 200 {
		return nil, falloGrado(http.StatusBadRequest, "Paginación inválida")
	}
	secretaria, err := resolverSecretariaGrado(ctx, auth)
	if err != nil {
		return nil, err
	}
	estados, err := estadosSolicitudRevisionGrado(secretaria.Ctx)
	if err != nil {
		return nil, err
	}
	idsEstados := idsEstadosRevisionGrado(estados)
	if estado != "" {
		estadoID, ok := estados[estado]
		if !ok {
			return nil, falloGrado(http.StatusBadRequest, "Estado documental inválido")
		}
		idsEstados = []int{estadoID}
	}
	porID := make(map[int]models.BorradorGrado)
	for _, dependencia := range secretaria.Dependencias {
		expedientes, err := consultarPaginasRevisionCRUD(secretaria.Ctx, dependencia, periodoID, programaID, idsEstados)
		if err != nil {
			return nil, err
		}
		for _, expediente := range expedientes {
			if !coincideFiltrosRevisionGrado(expediente, periodoID, programaID) || expediente.Solicitud.Id <= 0 {
				return nil, falloGrado(http.StatusServiceUnavailable, "El CRUD devolvió una solicitud fuera de los filtros autorizados")
			}
			if _, repetido := porID[expediente.Solicitud.Id]; repetido {
				return nil, falloGrado(http.StatusServiceUnavailable, "Paginación documental inconsistente")
			}
			porID[expediente.Solicitud.Id] = expediente
		}
	}
	lista := make([]models.BorradorGrado, 0, len(porID))
	for _, expediente := range porID {
		if !coincideFiltrosRevisionGrado(expediente, periodoID, programaID) {
			continue
		}
		lista = append(lista, expediente)
	}
	sort.Slice(lista, func(i, j int) bool { return lista[i].Solicitud.Id > lista[j].Solicitud.Id })
	if texto == "" {
		total := len(lista)
		if offset >= total {
			return &models.PaginaRevisionGrado{Solicitudes: []models.SolicitudRevisionGrado{}, Total: total}, nil
		}
		fin := offset + limit
		if fin > total {
			fin = total
		}
		pagina := make([]models.SolicitudRevisionGrado, 0, fin-offset)
		for _, borrador := range lista[offset:fin] {
			expediente, err := expedienteRevisionGrado(secretaria.Ctx, borrador, estados, false)
			if err != nil {
				return nil, err
			}
			pagina = append(pagina, *expediente)
		}
		return &models.PaginaRevisionGrado{Solicitudes: pagina, Total: total}, nil
	}
	resultado := make([]models.SolicitudRevisionGrado, 0, len(lista))
	for _, borrador := range lista {
		expediente, err := expedienteRevisionGrado(secretaria.Ctx, borrador, estados, false)
		if err != nil {
			return nil, err
		}
		if coincideTextoRevisionGrado(*expediente, texto) {
			resultado = append(resultado, *expediente)
		}
	}
	total := len(resultado)
	if offset >= total {
		return &models.PaginaRevisionGrado{Solicitudes: []models.SolicitudRevisionGrado{}, Total: total}, nil
	}
	fin := offset + limit
	if fin > total {
		fin = total
	}
	return &models.PaginaRevisionGrado{Solicitudes: resultado[offset:fin], Total: total}, nil
}

func coincideTextoRevisionGrado(expediente models.SolicitudRevisionGrado, texto string) bool {
	busqueda := strings.ToLower(strings.TrimSpace(texto))
	contenido := strings.ToLower(strings.Join([]string{
		expediente.Estudiante.NombreCompleto, expediente.Estudiante.NumeroIdentificacion,
		expediente.Solicitud.CodigoEstudiante, expediente.Programa, expediente.Periodo,
		expediente.Estado, expediente.Formulario.TrabajoGrado,
	}, " "))
	return busqueda == "" || strings.Contains(contenido, busqueda)
}

func coincideFiltrosRevisionGrado(expediente models.BorradorGrado, periodoID, programaID int) bool {
	return (periodoID == 0 || expediente.Solicitud.PeriodoId == periodoID) &&
		(programaID == 0 || expediente.Solicitud.ProgramaAcademicoId == programaID)
}

// Los programas del filtro provienen del catálogo institucional de todas las
// dependencias de las facultades del funcionario, no de solicitudes existentes.
func FiltrosRevisionDocumentalGrado(ctx context.Context, auth string) (*models.FiltrosRevisionGrado, error) {
	secretaria, err := resolverSecretariaGrado(ctx, auth)
	if err != nil {
		return nil, err
	}
	return consultarFiltrosRevisionGrado(secretaria)
}

func consultarFiltrosRevisionGrado(secretaria *identidadSecretariaGrado) (*models.FiltrosRevisionGrado, error) {
	parametros, err := baseGrado("UrlcrudParametros")
	if err != nil {
		return nil, err
	}
	var raw json.RawMessage
	periodosQuery := url.Values{"query": {"CodigoAbreviacion:PA,Activo:true"}, "limit": {"0"}, "sortby": {"Id"}, "order": {"desc"}}
	if _, err := request.GetWithContext(secretaria.Ctx, parametros+"periodo?"+periodosQuery.Encode(), &raw); err != nil {
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
	resultado := &models.FiltrosRevisionGrado{Periodos: make([]models.OpcionFiltroRevisionGrado, 0, len(periodos)),
		Programas: []models.ProgramaFiltroRevisionGrado{}}
	periodosVistos := make(map[int]bool)
	for _, periodo := range periodos {
		if periodo.Id <= 0 || !periodo.Activo || strings.TrimSpace(periodo.Nombre) == "" || periodosVistos[periodo.Id] {
			return nil, falloGrado(http.StatusServiceUnavailable, "Catálogo de periodos inconsistente")
		}
		periodosVistos[periodo.Id] = true
		resultado.Periodos = append(resultado.Periodos, models.OpcionFiltroRevisionGrado{Id: periodo.Id, Nombre: strings.TrimSpace(periodo.Nombre)})
	}
	proyectos, err := baseGrado("UrlProyectoAcademico")
	if err != nil {
		return nil, err
	}
	programasVistos := make(map[int]bool)
	for _, dependencia := range secretaria.Dependencias {
		q := url.Values{"query": {"DependenciaId:" + strconv.Itoa(dependencia) + ",Activo:true"}, "limit": {"0"}}
		var respuesta json.RawMessage
		if _, err := request.GetWithContext(secretaria.Ctx, proyectos+"proyecto_academico_institucion?"+q.Encode(), &respuesta); err != nil {
			return nil, falloGrado(http.StatusServiceUnavailable, "No se pudieron consultar los programas de la facultad")
		}
		programas, err := listaGrado[struct {
			Id            int    `json:"Id"`
			Nombre        string `json:"Nombre"`
			DependenciaId int    `json:"DependenciaId"`
			Activo        bool   `json:"Activo"`
		}](respuesta)
		if err != nil {
			return nil, falloGrado(http.StatusServiceUnavailable, "Catálogo de programas inválido")
		}
		for _, programa := range programas {
			// Proyecto Académico puede incluir proyectos ajenos en una respuesta de
			// catálogo. Solo los de la dependencia autorizada se exponen al MF.
			if programa.DependenciaId != dependencia || !programa.Activo {
				continue
			}
			if programa.Id <= 0 || strings.TrimSpace(programa.Nombre) == "" {
				return nil, falloGrado(http.StatusServiceUnavailable, "Programa académico inválido en el ámbito autorizado")
			}
			if programasVistos[programa.Id] {
				continue
			}
			programasVistos[programa.Id] = true
			resultado.Programas = append(resultado.Programas, models.ProgramaFiltroRevisionGrado{
				Id: programa.Id, Nombre: strings.TrimSpace(programa.Nombre), DependenciaId: dependencia,
			})
		}
	}
	sort.Slice(resultado.Programas, func(i, j int) bool { return resultado.Programas[i].Nombre < resultado.Programas[j].Nombre })
	return resultado, nil
}

func ConsultarRevisionDocumentalGrado(ctx context.Context, auth string, id int) (*models.SolicitudRevisionGrado, error) {
	if id <= 0 {
		return nil, falloGrado(http.StatusBadRequest, "Solicitud inválida")
	}
	secretaria, err := resolverSecretariaGrado(ctx, auth)
	if err != nil {
		return nil, err
	}
	estados, err := estadosSolicitudRevisionGrado(secretaria.Ctx)
	if err != nil {
		return nil, err
	}
	borrador, _, err := buscarRevisionEnAlcance(secretaria.Ctx, id, secretaria.Dependencias, idsEstadosRevisionGrado(estados))
	if err != nil {
		return nil, err
	}
	return expedienteRevisionGrado(secretaria.Ctx, *borrador, estados, true)
}

func DescargarSoporteRevisionGrado(ctx context.Context, auth string, id int, codigo string) (*models.ArchivoSoporteGrado, error) {
	if _, err := codigoDocumentoGrado(codigo); err != nil {
		return nil, err
	}
	secretaria, err := resolverSecretariaGrado(ctx, auth)
	if err != nil {
		return nil, err
	}
	estados, err := estadosSolicitudRevisionGrado(secretaria.Ctx)
	if err != nil {
		return nil, err
	}
	borrador, _, err := buscarRevisionEnAlcance(secretaria.Ctx, id, secretaria.Dependencias, idsEstadosRevisionGrado(estados))
	if err != nil {
		return nil, err
	}
	tipoID, err := resolverParametroGrado(secretaria.Ctx, "TIP_SOP_GRADO", codigo)
	if err != nil {
		return nil, err
	}
	for _, soporte := range borrador.Soportes {
		if soporte.TipoDocumentoId == tipoID {
			documento, err := consultarDocumentoGrado(secretaria.Ctx, soporte.DocumentoId)
			if err != nil {
				return nil, err
			}
			archivo, _, err := archivoDocumentoGrado(secretaria.Ctx, documento)
			return archivo, err
		}
	}
	return nil, falloGrado(http.StatusNotFound, "Soporte no encontrado")
}

func RevisarDocumentacionSecretariaGrado(ctx context.Context, auth string, id int, entrada models.RevisarDocumentacionGrado) (*models.SolicitudRevisionGrado, error) {
	if id <= 0 || entrada.FormularioId <= 0 || len(entrada.Soportes) < 3 || len(entrada.Soportes) > 4 {
		return nil, falloGrado(http.StatusBadRequest, "Solicitud, versión y decisiones de soportes son obligatorias")
	}
	secretaria, err := resolverSecretariaGrado(ctx, auth)
	if err != nil {
		return nil, err
	}
	estados, err := estadosSolicitudRevisionGrado(secretaria.Ctx)
	if err != nil {
		return nil, err
	}
	borrador, dependencia, err := buscarRevisionEnAlcance(secretaria.Ctx, id, secretaria.Dependencias, []int{estados["SG_RADICADA"]})
	if err != nil {
		return nil, err
	}
	if borrador.Formulario.Id != entrada.FormularioId || borrador.Formulario.FechaRadicacion == nil ||
		len(borrador.Soportes) != len(entrada.Soportes) {
		return nil, falloGrado(http.StatusConflict, "La versión radicada cambió")
	}
	if err := validarVentanaAprobacionPazSalvoGrado(secretaria.Ctx, borrador.Solicitud); err != nil {
		return nil, err
	}
	estadoObservado, err := resolverParametroGrado(secretaria.Ctx, "EST_SOP_GRADO", "SD_OBSERVADO")
	if err != nil {
		return nil, err
	}
	estadoAprobado, err := resolverParametroGrado(secretaria.Ctx, "EST_SOP_GRADO", "SD_APROBADO")
	if err != nil {
		return nil, err
	}
	soportesActuales := make(map[int]bool, len(borrador.Soportes))
	for _, soporte := range borrador.Soportes {
		soportesActuales[soporte.Id] = true
	}
	decisiones := make([]map[string]interface{}, 0, len(entrada.Soportes))
	vistos := make(map[int]bool, len(entrada.Soportes))
	for _, decision := range entrada.Soportes {
		if !soportesActuales[decision.SoporteId] || vistos[decision.SoporteId] || (entrada.Aprobada && decision.Observado) {
			return nil, falloGrado(http.StatusBadRequest, "Las decisiones no corresponden a los soportes vigentes")
		}
		vistos[decision.SoporteId] = true
		estado := estadoAprobado
		if decision.Observado {
			estado = estadoObservado
		}
		decisiones = append(decisiones, map[string]interface{}{
			"SoporteId": decision.SoporteId, "EstadoSoporteId": estado, "Observacion": strings.TrimSpace(decision.Observacion),
		})
	}
	body := map[string]interface{}{
		"FormularioId": entrada.FormularioId, "TerceroId": secretaria.TerceroID,
		"Aprobada": entrada.Aprobada, "Justificacion": strings.TrimSpace(entrada.Justificacion),
		"EstadoRadicadaId": estados["SG_RADICADA"], "EstadoObservadaId": estados["SG_OBSERVADA"],
		"EstadoDocumentacionAprobadaId": estados["SG_DOC_APROBADA"],
		"EstadoSoporteObservadoId":      estadoObservado, "EstadoSoporteAprobadoId": estadoAprobado,
		"EstadoPazSalvoPendienteId": 0, "TiposPazSalvoId": []int{}, "Soportes": decisiones,
	}
	if entrada.Aprobada {
		pendiente, err := resolverParametroGrado(secretaria.Ctx, "EST_PAZ_SALVO", "PS_PENDIENTE")
		if err != nil {
			return nil, err
		}
		tipos := make([]int, 0, len(codigosCheckGrado))
		for _, codigo := range codigosCheckGrado {
			tipo, err := resolverParametroGrado(secretaria.Ctx, "TIP_PAZ_SALVO", codigo)
			if err != nil {
				return nil, err
			}
			tipos = append(tipos, tipo)
		}
		body["EstadoPazSalvoPendienteId"] = pendiente
		body["TiposPazSalvoId"] = tipos
	}
	base, err := baseGrado("UrlCrudPazySalvos")
	if err != nil {
		return nil, err
	}
	q := url.Values{"dependencia_id": {strconv.Itoa(dependencia)}}
	var respuesta models.APIResponseData[models.BorradorGrado]
	status, err := request.PostWithContext(secretaria.Ctx, base+"solicitud-grado/revision/"+strconv.Itoa(id)+"?"+q.Encode(), body, &respuesta)
	if err != nil {
		if status == http.StatusConflict {
			return nil, falloGrado(status, "La solicitud cambió; recarga antes de decidir")
		}
		if status >= 400 && status < 500 {
			return nil, falloGrado(status, "La revisión documental fue rechazada")
		}
		return nil, falloGrado(http.StatusServiceUnavailable, "No se pudo registrar la revisión documental")
	}
	if status != http.StatusOK || !respuesta.Success || respuesta.Status != http.StatusOK || respuesta.Data.Solicitud.Id != id {
		return nil, falloGrado(http.StatusServiceUnavailable, "Confirmación de revisión inválida")
	}
	return expedienteRevisionGrado(secretaria.Ctx, respuesta.Data, estados, true)
}
