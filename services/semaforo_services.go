package services

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/beego/beego/v2/core/logs"
	"github.com/udistrital/paz_y_salvos_mid/helpers"
	"github.com/udistrital/paz_y_salvos_mid/models"
	"github.com/udistrital/utils_oas/v2/request"
)

const authorizationContextKey = "Authorization"

func externalRequestContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, authorizationContextKey, "")
}

// ConsultarSemaforosAsistente consulta los proyectos donde el usuario es asistente y retorna los semáforos de esos proyectos
func ConsultarSemaforosAsistente(ctx context.Context, cedula string, limit int, offset int, codigo string, idProyecto int, anio int, periodo int) models.APIResponse {
	ctx = externalRequestContext(ctx)
	// Obtiene los proyectos asignados al asistente desde Académica.
	urlAsistente :=
		helpers.ConfigString("UrlcrudWSO2") +
			helpers.ConfigString("NscrudAcademica") +
			"/asistente_proyecto/" + cedula

	var resAsistente models.AsistenteProyectoResponse
	if _, err := request.GetWithContext(ctx, urlAsistente, &resAsistente); err != nil {
		logs.Error("No se pudo obtener los proyectos del asistente %s: %v", cedula, err)
		return models.APIResponse{Success: false, Status: http.StatusServiceUnavailable, Message: "No se pudo consultar los proyectos del asistente.", Data: nil}
	}

	proyectos := make([]string, 0, len(resAsistente.Asistente.Proyectos))
	for _, proyecto := range resAsistente.Asistente.Proyectos {
		if proyecto.Proyecto != "" {
			proyectos = append(proyectos, proyecto.Proyecto)
		}
	}
	// La consulta requiere al menos un proyecto asignado.
	esAsistente := len(proyectos) > 0

	if !esAsistente {
		resp := models.SemaforosAsistenteResponse{
			EsAsistente: false,
			Semaforos:   []models.SemaforoTable{},
			Limit:       limit,
			TotalCount:  0,
		}
		return models.APIResponse{Success: false, Status: http.StatusNotFound, Message: "El asistente no tiene proyectos asignados.", Data: resp}
	}

	// Homologa cada código académico a la dependencia Oikos correspondiente.
	var idsOikos []int
	proyectosMap := make(map[int]models.ProyectoAsignado) // Evita repetir proyectos por ID Oikos.
	for _, cod := range proyectos {
		urlHom :=
			helpers.ConfigString("UrlcrudWSO2") +
				helpers.ConfigString("NscrudHomologacion") +
				"/proyecto_curricular_cod_proyecto/" + cod

		var resHom models.HomologacionResponse
		if _, err := request.GetWithContext(ctx, urlHom, &resHom); err != nil {
			logs.Warn("Error al consultar homologación para proyecto %s: %v", cod, err)
			continue
		}
		idOikos := resHom.OikosId()
		if idOikos > 0 {
			// Cada dependencia Oikos aparece una sola vez en el resultado.
			if _, existe := proyectosMap[idOikos]; !existe {
				idsOikos = append(idsOikos, idOikos)
				urlOikos := helpers.ConfigString("UrlcrudOikos") + "dependencia/" + fmt.Sprintf("%d", idOikos)
				var resOikos models.DependenciaOikos
				if _, err := request.GetWithContext(ctx, urlOikos, &resOikos); err != nil {
					logs.Warn("No se pudo obtener información del proyecto %d: %v", idOikos, err)
				}
				proyectosMap[idOikos] = models.ProyectoAsignado{
					IdOikos: idOikos,
					Codigo:  cod,
					Nombre:  strings.ToUpper(resOikos.Nombre),
				}
			}
		}
	}
	// Prepara la lista de proyectos incluidos en la respuesta.
	var proyectosAsignados []models.ProyectoAsignado
	for _, proyecto := range proyectosMap {
		proyectosAsignados = append(proyectosAsignados, proyecto)
	}

	if len(idsOikos) == 0 {
		return models.APIResponse{Success: false, Status: http.StatusNotFound, Message: "No se encontraron proyectos oikos para el asistente.", Data: nil}
	}

	// Acota la consulta del semáforo a los proyectos acreditados.
	var queryParts []string
	// Un proyecto solicitado debe pertenecer a las asignaciones del asistente.
	if idProyecto > 0 {
		// Impide consultar proyectos fuera del ámbito del asistente.
		proyectoValido := false
		for _, id := range idsOikos {
			if id == idProyecto {
				proyectoValido = true
				break
			}
		}
		if !proyectoValido {
			return models.APIResponse{Success: false, Status: http.StatusForbidden, Message: "El proyecto no está asignado al asistente.", Data: nil}
		}
		queryParts = append(queryParts, fmt.Sprintf("IdProyectoOikos:%d", idProyecto))
	} else {
		// Si no se proporciona proyecto, usar todos los asignados
		proyectosQuery := "IdProyectoOikos:"
		for i, id := range idsOikos {
			if i > 0 {
				proyectosQuery += "|"
			}
			proyectosQuery += fmt.Sprintf("%d", id)
		}
		queryParts = append(queryParts, proyectosQuery)
	}
	queryParts = append(queryParts, "Activo:true")
	if codigo != "" {
		queryParts = append(queryParts, fmt.Sprintf("CodigoEstudiante__contains:%s", codigo))
	}
	if anio > 0 {
		queryParts = append(queryParts, fmt.Sprintf("AnioInsGrado:%d", anio))
	}
	if periodo > 0 {
		queryParts = append(queryParts, fmt.Sprintf("PerInsGrado:%d", periodo))
	}
	queryString := strings.Join(queryParts, ",")
	query := fmt.Sprintf("?query=%s&limit=%d&offset=%d", queryString, limit, offset)

	// Obtener los semáforos normalmente
	resp := obtenerSemaforos(ctx, query, "No se encontraron estudiantes activos para los proyectos del asistente.")

	var semaforosTable []models.SemaforoTable
	var totalCount int

	if result, ok := resp.Data.(models.SemaforosResponse); ok {
		semaforosTable = result.Semaforos
		totalCount = result.TotalCount
	}

	if semaforosTable == nil {
		semaforosTable = []models.SemaforoTable{}
	}
	resp.Data = models.SemaforosAsistenteResponse{
		Semaforos:          semaforosTable,
		Limit:              limit,
		TotalCount:         totalCount,
		EsAsistente:        esAsistente,
		ProyectosAsignados: proyectosAsignados,
	}
	return resp
}

func ConsultarEstudiante(ctx context.Context, codigo string, limit int, offset int) models.APIResponse {
	ctx = externalRequestContext(ctx)
	query := fmt.Sprintf("?query=CodigoEstudiante:%s,Activo:true&limit=%d&offset=%d", codigo, limit, offset)
	return obtenerSemaforos(ctx, query, "No se encontró información del estudiante.")
}

func ConsultarEstudiantes(ctx context.Context, limit int, offset int, codigo string, idFacultad int, idProyecto int, anio int, periodo int) models.APIResponse {
	ctx = externalRequestContext(ctx)
	// Construir query dinámicamente con filtros activos
	var queryParts []string
	queryParts = append(queryParts, "Activo:true")

	if codigo != "" {
		queryParts = append(queryParts, fmt.Sprintf("CodigoEstudiante__contains:%s", codigo))
	}
	if idFacultad > 0 {
		queryParts = append(queryParts, fmt.Sprintf("IdFacultadOikos:%d", idFacultad))
	}
	if idProyecto > 0 {
		queryParts = append(queryParts, fmt.Sprintf("IdProyectoOikos:%d", idProyecto))
	}
	if anio > 0 {
		queryParts = append(queryParts, fmt.Sprintf("AnioInsGrado:%d", anio))
	}
	if periodo > 0 {
		queryParts = append(queryParts, fmt.Sprintf("PerInsGrado:%d", periodo))
	}

	queryString := strings.Join(queryParts, ",")

	// Aplica los filtros y la paginación solicitados.
	query := fmt.Sprintf("?query=%s&limit=%d&offset=%d", queryString, limit, offset)

	return obtenerSemaforos(ctx, query, "No se encontraron estudiantes activos.")
}

func ConsultarEstudiantesProyecto(ctx context.Context, id_coordinador string, limit int, offset int, codigo string, idProyecto int, anio int, periodo int) models.APIResponse {
	ctx = externalRequestContext(ctx)
	// Consulta los códigos académicos asignados al coordinador.
	urlCoord :=
		helpers.ConfigString("UrlcrudWSO2") +
			helpers.ConfigString("NscrudAcademica") +
			"/coordinador_carrera_snies/" + id_coordinador

	var resCoord models.CoordinadorCarreraResponse
	if _, err := request.GetWithContext(ctx, urlCoord, &resCoord); err != nil {
		logs.Error("No se pudo obtener los proyectos del coordinador %s: %v", id_coordinador, err)
		return models.APIResponse{Success: false, Status: http.StatusServiceUnavailable, Message: "No se pudo consultar los proyectos del coordinador.", Data: nil}
	}

	var codigosCondor []string
	for _, coordinador := range resCoord.CoordinadorCollection.Coordinadores {
		if coordinador.CodigoCondor != "" {
			codigosCondor = append(codigosCondor, coordinador.CodigoCondor)
		}
	}

	tieneProyectos := len(codigosCondor) > 0

	if !tieneProyectos {
		resp := models.SemaforoCoordinadorResponse{
			Semaforos:          []models.SemaforoTable{},
			Limit:              limit,
			TotalCount:         0,
			ProyectosAsignados: []models.ProyectoAsignado{},
		}
		return models.APIResponse{Success: false, Status: http.StatusNotFound, Message: "El coordinador no tiene proyectos asociados.", Data: resp}
	}

	// Asocia los proyectos académicos con dependencias Oikos únicas.
	var idsOikos []int
	proyectosMap := make(map[int]models.ProyectoAsignado)

	for _, cod := range codigosCondor {
		urlHom :=
			helpers.ConfigString("UrlcrudWSO2") +
				helpers.ConfigString("NscrudHomologacion") +
				"/proyecto_curricular_cod_proyecto/" + cod

		var resHom models.HomologacionResponse
		if _, err := request.GetWithContext(ctx, urlHom, &resHom); err != nil {
			logs.Warn("Error al consultar homologación para proyecto %s: %v", cod, err)
			continue
		}

		idOikos := resHom.OikosId()
		if idOikos > 0 {
			if _, existe := proyectosMap[idOikos]; !existe {
				idsOikos = append(idsOikos, idOikos)
				urlOikos := helpers.ConfigString("UrlcrudOikos") + "dependencia/" + fmt.Sprintf("%d", idOikos)
				var resOikos models.DependenciaOikos
				if _, err := request.GetWithContext(ctx, urlOikos, &resOikos); err != nil {
					logs.Warn("No se pudo obtener información del proyecto %d: %v", idOikos, err)
				}
				proyectosMap[idOikos] = models.ProyectoAsignado{
					IdOikos: idOikos,
					Codigo:  cod,
					Nombre:  strings.ToUpper(resOikos.Nombre),
				}
			}
		}
	}

	// Incluye los proyectos autorizados en la respuesta.
	var proyectosAsignados []models.ProyectoAsignado
	for _, proyecto := range proyectosMap {
		proyectosAsignados = append(proyectosAsignados, proyecto)
	}

	if len(idsOikos) == 0 {
		return models.APIResponse{Success: false, Status: http.StatusNotFound, Message: "No se encontraron proyectos oikos para el coordinador.", Data: nil}
	}

	// Restringe el semáforo a los proyectos asignados al coordinador.
	var queryParts []string

	// Un proyecto solicitado debe pertenecer a las asignaciones del coordinador.
	if idProyecto > 0 {
		// Impide consultar proyectos fuera de su ámbito.
		proyectoValido := false
		for _, id := range idsOikos {
			if id == idProyecto {
				proyectoValido = true
				break
			}
		}
		if !proyectoValido {
			return models.APIResponse{Success: false, Status: http.StatusForbidden, Message: "El proyecto no está asignado al coordinador.", Data: nil}
		}
		queryParts = append(queryParts, fmt.Sprintf("IdProyectoOikos:%d", idProyecto))
	} else {
		// Sin proyecto específico, consulta todos los proyectos acreditados.
		proyectosQuery := "IdProyectoOikos:"
		for i, id := range idsOikos {
			if i > 0 {
				proyectosQuery += "|"
			}
			proyectosQuery += fmt.Sprintf("%d", id)
		}
		queryParts = append(queryParts, proyectosQuery)
	}
	queryParts = append(queryParts, "Activo:true")

	// Agregar filtros adicionales si están presentes
	if codigo != "" {
		queryParts = append(queryParts, fmt.Sprintf("CodigoEstudiante__contains:%s", codigo))
	}
	if anio > 0 {
		queryParts = append(queryParts, fmt.Sprintf("AnioInsGrado:%d", anio))
	}
	if periodo > 0 {
		queryParts = append(queryParts, fmt.Sprintf("PerInsGrado:%d", periodo))
	}

	queryString := strings.Join(queryParts, ",")
	query := fmt.Sprintf("?query=%s&limit=%d&offset=%d", queryString, limit, offset)

	// Conserva el total filtrado y los registros de la página solicitada.
	resp := obtenerSemaforos(ctx, query, "No se encontraron estudiantes activos para los proyectos del coordinador.")

	var semaforosTable []models.SemaforoTable
	var totalCount int

	if result, ok := resp.Data.(models.SemaforosResponse); ok {
		semaforosTable = result.Semaforos
		totalCount = result.TotalCount
	}

	if semaforosTable == nil {
		semaforosTable = []models.SemaforoTable{}
	}

	resp.Data = models.SemaforoCoordinadorResponse{
		Semaforos:          semaforosTable,
		Limit:              limit,
		TotalCount:         totalCount,
		ProyectosAsignados: proyectosAsignados,
	}

	return resp
}

func ConsultarEstudiantesFacultad(ctx context.Context, id_secretario string, limit int, offset int, codigo string, idProyecto int, anio int, periodo int) models.APIResponse {
	ctx = externalRequestContext(ctx)
	// Consulta las facultades asociadas al secretario académico.

	urlSec :=
		helpers.ConfigString("UrlcrudWSO2") +
			"academica_pruebas" +
			"/facultad_secretaria/" + id_secretario

	var resSec models.FacultadesSecretariaResponse
	if _, err := request.GetWithContext(ctx, urlSec, &resSec); err != nil {
		logs.Error("No se pudo obtener las facultades del secretario %s: %v", id_secretario, err)
		return models.APIResponse{Success: false, Status: http.StatusServiceUnavailable, Message: "No se pudo consultar las facultades del secretario.", Data: nil}
	}

	var codigosCondor []string
	for _, facultad := range resSec.Facultades.Secretarias {
		if facultad.Codigo != "" {
			codigosCondor = append(codigosCondor, facultad.Codigo)
		}
	}

	if len(codigosCondor) == 0 {
		return models.APIResponse{Success: false, Status: http.StatusNotFound, Message: "El secretario no tiene facultades asociadas.", Data: nil}
	}

	// Homologa los códigos de facultad a las dependencias Oikos.
	var idsOikos []int
	for _, cod := range codigosCondor {
		urlHom :=
			helpers.ConfigString("UrlcrudWSO2") +
				helpers.ConfigString("NscrudHomologacion") +
				"/facultad_oikos_gedep/" + cod

		var resHom models.HomologacionResponse
		if _, err := request.GetWithContext(ctx, urlHom, &resHom); err != nil {
			logs.Warn("Error al consultar homologación para facultad %s: %v", cod, err)
			continue
		}

		if idOikos := resHom.OikosId(); idOikos > 0 {
			idsOikos = append(idsOikos, idOikos)
		}
	}

	if len(idsOikos) == 0 {
		return models.APIResponse{Success: false, Status: http.StatusNotFound, Message: "No se encontraron facultades oikos para el secretario.", Data: nil}
	}

	// Incluye las facultades acreditadas en el filtro de la consulta.
	var queryParts []string
	queryParts = append(queryParts, "IdFacultadOikos:")
	for i, id := range idsOikos {
		if i > 0 {
			queryParts[0] += "|"
		}
		queryParts[0] += fmt.Sprintf("%d", id)
	}
	queryParts = append(queryParts, "Activo:true")

	// Agregar filtros adicionales si están presentes
	if codigo != "" {
		queryParts = append(queryParts, fmt.Sprintf("CodigoEstudiante__contains:%s", codigo))
	}
	if idProyecto > 0 {
		queryParts = append(queryParts, fmt.Sprintf("IdProyectoOikos:%d", idProyecto))
	}
	if anio > 0 {
		queryParts = append(queryParts, fmt.Sprintf("AnioInsGrado:%d", anio))
	}
	if periodo > 0 {
		queryParts = append(queryParts, fmt.Sprintf("PerInsGrado:%d", periodo))
	}

	queryString := strings.Join(queryParts, ",")
	query := fmt.Sprintf("?query=%s&limit=%d&offset=%d", queryString, limit, offset)

	// La respuesta identifica la primera facultad acreditada.
	var idFacultadOikos int
	if len(idsOikos) > 0 {
		idFacultadOikos = idsOikos[0]
	}

	return obtenerSemaforosConFacultad(ctx, query, "No se encontraron estudiantes activos en las facultades del secretario.", idFacultadOikos)
}

func ConsultarEstudiantesFacultadLaboratorios(ctx context.Context, id_coordinador_lab string, limit int, offset int, codigo string, idProyecto int, anio int, periodo int) models.APIResponse {
	ctx = externalRequestContext(ctx)

	// Consulta las dependencias bajo jefatura vigente en la fecha actual.
	fechaActual := time.Now().Format("2006-01-02")

	urlJefe :=
		helpers.ConfigString("UrlcrudCore") +
			"/jefe_dependencia?query=TerceroId:" + id_coordinador_lab +
			",FechaFin__gte:" + fechaActual +
			",FechaInicio__lte:" + fechaActual

	var resJefe []models.JefeDependencia
	if _, err := request.GetWithContext(ctx, urlJefe, &resJefe); err != nil {
		logs.Error("No se pudo obtener las dependencias del jefe %s: %v", id_coordinador_lab, err)
		return models.APIResponse{Success: false, Status: http.StatusServiceUnavailable, Message: "No se pudo consultar las dependencias del jefe de laboratorios.", Data: nil}
	}

	var dependenciasConNombre []models.DependenciaConNombre

	for _, jefe := range resJefe {
		urlDep :=
			helpers.ConfigString("UrlcrudOikos") +
				"dependencia/" + fmt.Sprintf("%d", jefe.DependenciaId)

		var resDep models.DependenciaOikos
		if _, err := request.GetWithContext(ctx, urlDep, &resDep); err != nil {
			logs.Warn("No se pudo obtener información de la dependencia %d: %v", jefe.DependenciaId, err)
			continue
		}

		dependenciasConNombre = append(dependenciasConNombre, models.DependenciaConNombre{
			DependenciaId: jefe.DependenciaId,
			Nombre:        resDep.Nombre,
		})
	}

	// Identifica laboratorios por nombre, sin distinguir mayúsculas.
	var laboratorios []models.DependenciaConNombre
	for _, dep := range dependenciasConNombre {
		if dep.Nombre != "" && helpers.ContainsIgnoreCase(dep.Nombre, "laboratorio") {
			laboratorios = append(laboratorios, dep)
		}
	}

	// Sin laboratorios identificados, consulta las dependencias asociadas al jefe.
	if len(laboratorios) == 0 {
		logs.Info("No se encontraron dependencias de laboratorios. Probablemente es el decano de la facultad.")
		// Acota el semáforo a las dependencias asociadas al jefe.
		var idsDependencias []int
		for _, dep := range dependenciasConNombre {
			idsDependencias = append(idsDependencias, dep.DependenciaId)
		}
		if len(idsDependencias) == 0 {
			return models.APIResponse{Success: false, Status: http.StatusNotFound, Message: "No se encontraron dependencias asociadas al decano.", Data: nil}
		}

		// Construir query con filtros adicionales
		var queryParts []string
		queryParts = append(queryParts, "IdFacultadOikos:")
		for i, id := range idsDependencias {
			if i > 0 {
				queryParts[0] += "|"
			}
			queryParts[0] += fmt.Sprintf("%d", id)
		}
		queryParts = append(queryParts, "Activo:true")

		// Agregar filtros adicionales si están presentes
		if codigo != "" {
			queryParts = append(queryParts, fmt.Sprintf("CodigoEstudiante__contains:%s", codigo))
		}
		if idProyecto > 0 {
			queryParts = append(queryParts, fmt.Sprintf("IdProyectoOikos:%d", idProyecto))
		}
		if anio > 0 {
			queryParts = append(queryParts, fmt.Sprintf("AnioInsGrado:%d", anio))
		}
		if periodo > 0 {
			queryParts = append(queryParts, fmt.Sprintf("PerInsGrado:%d", periodo))
		}

		queryString := strings.Join(queryParts, ",")
		query := fmt.Sprintf("?query=%s&limit=%d&offset=%d", queryString, limit, offset)

		// Obtener el primer ID de facultad
		var idFacultadOikos int
		if len(idsDependencias) > 0 {
			idFacultadOikos = idsDependencias[0]
		}

		return obtenerSemaforosConFacultad(ctx, query, "No se encontraron estudiantes activos en las facultades asociadas al decano.", idFacultadOikos)

	} else {
		// Resuelve la facultad padre de cada laboratorio identificado.
		logs.Info("Dependencias de laboratorios encontradas: %d", len(laboratorios))

		var facultadesOikos []int
		facultadesMap := make(map[int]bool) // Deduplica las facultades padre.

		for _, lab := range laboratorios {
			labId := lab.DependenciaId

			if labId == 0 {
				continue
			}

			// Consulta la relación padre de la dependencia de laboratorio.
			urlLabDep :=
				helpers.ConfigString("UrlcrudOikos") +
					"dependencia_padre/?query=Hija:" + fmt.Sprintf("%d", labId)

			var resLabDep []models.DependenciaPadreOikos
			if _, err := request.GetWithContext(ctx, urlLabDep, &resLabDep); err != nil {
				logs.Warn("No se pudo obtener información completa del laboratorio %d: %v", labId, err)
				continue
			}

			for _, relacion := range resLabDep {
				facultadId := relacion.Padre.Id
				if facultadId > 0 && !facultadesMap[facultadId] {
					facultadesMap[facultadId] = true
					facultadesOikos = append(facultadesOikos, facultadId)
					logs.Info("Facultad padre encontrada: ID %d (%s) para laboratorio ID %d", facultadId, relacion.Padre.Nombre, labId)
				}
			}
		}

		if len(facultadesOikos) == 0 {
			return models.APIResponse{Success: false, Status: http.StatusNotFound, Message: "No se encontraron facultades asociadas a los laboratorios del coordinador.", Data: nil}
		}

		// Registra cuando los laboratorios pertenecen a varias facultades.
		if len(facultadesOikos) > 1 {
			logs.Warn("ALERTA: El coordinador de laboratorios tiene laboratorios en %d facultades diferentes: %v", len(facultadesOikos), facultadesOikos)
			fmt.Printf("ALERTA: Se encontraron laboratorios en %d facultades diferentes: %v\n", len(facultadesOikos), facultadesOikos)
		}

		// Construir query con las facultades encontradas y filtros adicionales
		var queryParts []string
		queryParts = append(queryParts, "IdFacultadOikos:")
		for i, id := range facultadesOikos {
			if i > 0 {
				queryParts[0] += "|"
			}
			queryParts[0] += fmt.Sprintf("%d", id)
		}
		queryParts = append(queryParts, "Activo:true")

		// Agregar filtros adicionales si están presentes
		if codigo != "" {
			queryParts = append(queryParts, fmt.Sprintf("CodigoEstudiante__contains:%s", codigo))
		}
		if idProyecto > 0 {
			queryParts = append(queryParts, fmt.Sprintf("IdProyectoOikos:%d", idProyecto))
		}
		if anio > 0 {
			queryParts = append(queryParts, fmt.Sprintf("AnioInsGrado:%d", anio))
		}
		if periodo > 0 {
			queryParts = append(queryParts, fmt.Sprintf("PerInsGrado:%d", periodo))
		}

		queryString := strings.Join(queryParts, ",")
		query := fmt.Sprintf("?query=%s&limit=%d&offset=%d", queryString, limit, offset)

		// Obtener el primer ID de facultad
		var idFacultadOikos int
		if len(facultadesOikos) > 0 {
			idFacultadOikos = facultadesOikos[0]
		}

		return obtenerSemaforosConFacultad(ctx, query, "No se encontraron estudiantes activos en las facultades asociadas a los laboratorios.", idFacultadOikos)
	}
}

func obtenerSemaforos(ctx context.Context, query, notFoundMsg string) models.APIResponse {
	var res models.APIResponseData[[]models.Semaforo]

	url :=
		helpers.ConfigString("UrlCrudPazySalvos") + "/semaforo/" + query

	if _, err := request.GetWithContext(ctx, url, &res); err != nil {
		logs.Error("Error al consultar paz_y_salvos:", err)
		return models.APIResponse{Success: false, Status: http.StatusServiceUnavailable, Message: "Error al consultar los datos del semáforo.", Data: nil}
	}

	if !res.Success {
		return models.APIResponse{Success: false, Status: res.Status, Message: res.Message, Data: res.Data}
	}

	semaforos := res.Data
	if len(semaforos) == 0 {
		return models.APIResponse{Success: false, Status: http.StatusNotFound, Message: notFoundMsg, Data: nil}
	}

	// Cuenta todos los registros del filtro, independientemente de la página.
	totalCount := 0
	queryCount := query
	// Retira la paginación de la consulta de conteo.
	if strings.Contains(queryCount, "&limit=") {
		parts := strings.Split(queryCount, "&limit=")
		queryCount = parts[0] + "&limit=-1"
	}
	if strings.Contains(queryCount, "&offset=") {
		queryCount = strings.Split(queryCount, "&offset=")[0]
	}

	var resCount models.APIResponseData[[]models.Semaforo]
	urlCount :=
		helpers.ConfigString("UrlCrudPazySalvos") + "/semaforo/" + queryCount

	if _, err := request.GetWithContext(ctx, urlCount, &resCount); err == nil && resCount.Success {
		totalCount = len(resCount.Data)
	}

	// Completa los nombres de estudiante, facultad y proyecto de la página.
	tabla := consultarDataSemaforo(ctx, semaforos)

	// Adjunta el total y el tamaño de la página al listado.
	result := models.SemaforosResponse{
		Semaforos:  tabla,
		TotalCount: totalCount,
		Limit:      len(semaforos),
	}

	return models.APIResponse{Success: true, Status: http.StatusOK, Message: "Consulta exitosa", Data: result}
}

// obtenerSemaforosConFacultad devuelve una página de semáforos con su facultad Oikos y el total filtrado.
func obtenerSemaforosConFacultad(ctx context.Context, query, notFoundMsg string, idFacultadOikos int) models.APIResponse {
	var res models.APIResponseData[[]models.Semaforo]

	url :=
		helpers.ConfigString("UrlCrudPazySalvos") + "/semaforo/" + query

	if _, err := request.GetWithContext(ctx, url, &res); err != nil {
		logs.Error("Error al consultar paz_y_salvos:", err)
		return models.APIResponse{Success: false, Status: http.StatusServiceUnavailable, Message: "Error al consultar los datos del semáforo.", Data: nil}
	}

	if !res.Success {
		return models.APIResponse{Success: false, Status: res.Status, Message: res.Message, Data: res.Data}
	}

	semaforos := res.Data
	if len(semaforos) == 0 {
		return models.APIResponse{Success: false, Status: http.StatusNotFound, Message: notFoundMsg, Data: nil}
	}

	// Cuenta todos los registros del filtro, independientemente de la página.
	totalCount := 0
	queryCount := query
	// Retira la paginación de la consulta de conteo.
	if strings.Contains(queryCount, "&limit=") {
		parts := strings.Split(queryCount, "&limit=")
		queryCount = parts[0] + "&limit=-1"
	}
	if strings.Contains(queryCount, "&offset=") {
		queryCount = strings.Split(queryCount, "&offset=")[0]
	}

	var resCount models.APIResponseData[[]models.Semaforo]
	urlCount :=
		helpers.ConfigString("UrlCrudPazySalvos") + "/semaforo/" + queryCount

	if _, err := request.GetWithContext(ctx, urlCount, &resCount); err == nil && resCount.Success {
		totalCount = len(resCount.Data)
	}

	tabla := consultarDataSemaforo(ctx, semaforos)

	result := models.SemaforosFacultadResponse{
		SemaforosResponse: models.SemaforosResponse{
			Semaforos:  tabla,
			TotalCount: totalCount,
			Limit:      len(semaforos),
		},
		IdFacultadOikos: idFacultadOikos,
	}

	return models.APIResponse{Success: true, Status: http.StatusOK, Message: "Consulta exitosa", Data: result}
}

func consultarDataSemaforo(ctx context.Context, semaforos []models.Semaforo) []models.SemaforoTable {
	var result []models.SemaforoTable

	for _, s := range semaforos {
		var nombreFacultad, nombreProyecto, nombreEstudiante string

		// 1. Nombre del estudiante
		urlEst :=
			helpers.ConfigString("UrlcrudWSO2") +
				helpers.ConfigString("NscrudAcademica") +
				"/datos_basicos_estudiante/" + fmt.Sprintf("%0.f", s.CodigoEstudiante)
		var resEst models.DatosEstudianteResponse
		if _, err := request.GetWithContext(ctx, urlEst, &resEst); err != nil {
			logs.Warn("No se pudo obtener nombre del estudiante %0.f: %v", s.CodigoEstudiante, err)
		} else if datos := resEst.DatosEstudianteCollection.DatosBasicos; len(datos) == 1 {
			nombreEstudiante = datos[0].Nombre
		}

		// 2. Nombre de la facultad
		urlFac :=
			helpers.ConfigString("UrlcrudOikos") +
				"dependencia/" + fmt.Sprintf("%d", s.IdFacultadOikos)

		var resFac models.DependenciaOikos
		if _, err := request.GetWithContext(ctx, urlFac, &resFac); err != nil {
			logs.Warn("No se pudo obtener nombre de la facultad %d: %v", s.IdFacultadOikos, err)
		} else {
			nombreFacultad = resFac.Nombre
		}

		// Obtiene el nombre del proyecto curricular desde Oikos.
		urlProj :=
			helpers.ConfigString("UrlcrudOikos") +
				"dependencia/" + fmt.Sprintf("%d", s.IdProyectoOikos)

		var resProj models.DependenciaOikos
		if _, err := request.GetWithContext(ctx, urlProj, &resProj); err != nil {
			logs.Warn("No se pudo obtener nombre del proyecto %d: %v", s.IdProyectoOikos, err)
		} else {
			nombreProyecto = resProj.Nombre
		}

		result = append(result, models.SemaforoTable{
			Id:                      s.Id,
			CodigoEstudiante:        s.CodigoEstudiante,
			NombreEstudiante:        nombreEstudiante,
			NombreFacultad:          nombreFacultad,
			NombreProyecto:          nombreProyecto,
			AnioInsGrado:            s.AnioInsGrado,
			PerInsGrado:             s.PerInsGrado,
			Academico:               s.Academico,
			Financiero:              s.Financiero,
			Biblioteca:              s.Biblioteca,
			Laboratorios:            s.Laboratorios,
			Bienestar:               s.Bienestar,
			Urelinter:               s.Urelinter,
			Orc:                     s.Orc,
			ObservacionCoordinacion: s.ObservacionCoordinacion,
			ObservacionBiblioteca:   s.ObservacionBiblioteca,
			ObservacionLaboratorios: s.ObservacionLaboratorios,
			ObservacionBienestar:    s.ObservacionBienestar,
			ObservacionUrelinter:    s.ObservacionUrelinter,
			ObservacionOrc:          s.ObservacionOrc,
		})
	}

	return result
}
