package services

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
	"github.com/udistrital/paz_y_salvos_mid/helpers"
	"github.com/udistrital/paz_y_salvos_mid/models"
	"github.com/udistrital/utils_oas/request"
	"github.com/udistrital/utils_oas/requestresponse"
)

func ConsultarEstudiante(codigo string, limit int, offset int) requestresponse.APIResponse {
	query := fmt.Sprintf("?query=CodigoEstudiante:%s,Activo:true&limit=%d&offset=%d", codigo, limit, offset)
	return obtenerSemaforos(query, "No se encontró información del estudiante.")
}

func ConsultarEstudiantes(limit int, offset int, codigo string, idFacultad int, idProyecto int, anio int, periodo int) requestresponse.APIResponse {
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

	// Construir el query
	query := fmt.Sprintf("?query=%s&limit=%d&offset=%d", queryString, limit, offset)

	return obtenerSemaforos(query, "No se encontraron estudiantes activos.")
}

func ConsultarEstudiantesProyecto(id_coordinador string, limit int, offset int, codigo string, anio int, periodo int) requestresponse.APIResponse {
	// 1. Consultar proyectos del coordinador
	urlCoord := beego.AppConfig.String("ProtocolAdmin") + "://" +
		beego.AppConfig.String("UrlcrudWSO2") +
		beego.AppConfig.String("NscrudAcademica") +
		"/coordinador_carrera_snies/" + id_coordinador

	var resCoord map[string]interface{}
	if err := request.GetJsonWSO2(urlCoord, &resCoord); err != nil {
		logs.Error("No se pudo obtener los proyectos del coordinador %s: %v", id_coordinador, err)
		return requestresponse.APIResponseDTO(false, 503, nil, "No se pudo consultar los proyectos del coordinador.")
	}

	var codigosCondor []string
	if collection, ok := resCoord["coordinadorCollection"].(map[string]interface{}); ok {
		if lista, ok := collection["coordinador"].([]interface{}); ok {
			for _, item := range lista {
				if proyecto, ok := item.(map[string]interface{}); ok {
					if cod, ok := proyecto["codigo_condor"].(string); ok {
						codigosCondor = append(codigosCondor, cod)
					}
				}
			}
		}
	}
	if len(codigosCondor) == 0 {
		return requestresponse.APIResponseDTO(false, 404, nil, "El coordinador no tiene proyectos asociados.")
	}

	// 2. Homologar con servicio de homologación
	var idsOikos []int
	for _, cod := range codigosCondor {
		urlHom := beego.AppConfig.String("ProtocolAdmin") + "://" +
			beego.AppConfig.String("UrlcrudWSO2") +
			beego.AppConfig.String("NscrudHomologacion") +
			"/proyecto_curricular_cod_proyecto/" + cod

		var resHom map[string]interface{}
		if err := request.GetJsonWSO2(urlHom, &resHom); err != nil {
			logs.Warn("Error al consultar homologación para proyecto %s: %v", cod, err)
			continue
		}

		if hom, ok := resHom["homologacion"].(map[string]interface{}); ok {
			if idStr, ok := hom["id_oikos"].(string); ok {
				if idInt, err := strconv.Atoi(idStr); err == nil {
					idsOikos = append(idsOikos, idInt)
				}
			}
		}
	}

	if len(idsOikos) == 0 {
		return requestresponse.APIResponseDTO(false, 404, nil, "No se encontraron proyectos oikos para el coordinador.")
	}

	// 3. Construir query con filtros
	var queryParts []string

	// Filtro de proyectos del coordinador con OR
	proyectosQuery := "IdProyectoOikos:"
	for i, id := range idsOikos {
		if i > 0 {
			proyectosQuery += "|"
		}
		proyectosQuery += fmt.Sprintf("%d", id)
	}
	queryParts = append(queryParts, proyectosQuery)
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

	return obtenerSemaforos(query, "No se encontraron estudiantes activos para los proyectos del coordinador.")
}

func ConsultarEstudiantesFacultad(id_secretario string, limit int, offset int, codigo string, idProyecto int, anio int, periodo int) requestresponse.APIResponse {
	// 1. Consultar facultades del secretario
	// urlSec := beego.AppConfig.String("ProtocolAdmin") + "://" +
	// 	beego.AppConfig.String("UrlcrudWSO2") +
	// 	beego.AppConfig.String("NscrudAcademica") +
	// 	"/facultad_secretaria/" + id_secretario

	urlSec := beego.AppConfig.String("ProtocolAdmin") + "://" +
		beego.AppConfig.String("UrlcrudWSO2") +
		"academica_pruebas" +
		"/facultad_secretaria/" + id_secretario

	var resSec map[string]interface{}
	if err := request.GetJsonWSO2(urlSec, &resSec); err != nil {
		logs.Error("No se pudo obtener las facultades del secretario %s: %v", id_secretario, err)
		return requestresponse.APIResponseDTO(false, 503, nil, "No se pudo consultar las facultades del secretario.")
	}

	var codigosCondor []string
	if facultades, ok := resSec["facultades"].(map[string]interface{}); ok {
		if secretaria, ok := facultades["secretaria"].([]interface{}); ok {
			for _, item := range secretaria {
				if fac, ok := item.(map[string]interface{}); ok {
					if cod, ok := fac["SEC_DEP_COD"].(string); ok {
						codigosCondor = append(codigosCondor, cod)
					}
				}
			}
		}
	}

	if len(codigosCondor) == 0 {
		return requestresponse.APIResponseDTO(false, 404, nil, "El secretario no tiene facultades asociadas.")
	}

	// 2. Homologar con servicio de homologación
	var idsOikos []int
	for _, cod := range codigosCondor {
		urlHom := beego.AppConfig.String("ProtocolAdmin") + "://" +
			beego.AppConfig.String("UrlcrudWSO2") +
			beego.AppConfig.String("NscrudHomologacion") +
			"/facultad_oikos_gedep/" + cod

		var resHom map[string]interface{}
		if err := request.GetJsonWSO2(urlHom, &resHom); err != nil {
			logs.Warn("Error al consultar homologación para facultad %s: %v", cod, err)
			continue
		}

		if hom, ok := resHom["homologacion"].(map[string]interface{}); ok {
			if idStr, ok := hom["id_oikos"].(string); ok {
				if idInt, err := strconv.Atoi(idStr); err == nil {
					idsOikos = append(idsOikos, idInt)
				}
			}
		}
	}

	if len(idsOikos) == 0 {
		return requestresponse.APIResponseDTO(false, 404, nil, "No se encontraron facultades oikos para el secretario.")
	}

	// 3. Construir query con OR para facultades
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

	// Obtener el primer ID de facultad (si hay múltiples, tomamos el primero)
	var idFacultadOikos int
	if len(idsOikos) > 0 {
		idFacultadOikos = idsOikos[0]
	}

	return obtenerSemaforosConFacultad(query, "No se encontraron estudiantes activos en las facultades del secretario.", idFacultadOikos)
}

func ConsultarEstudiantesFacultadLaboratorios(id_coordinador_lab string, limit int, offset int, codigo string, idProyecto int, anio int, periodo int) requestresponse.APIResponse {

	// 1. Consultar de que dependencias es jefe
	// Obtener la fecha actual en formato YYYY-MM-DD
	fechaActual := time.Now().Format("2006-01-02")

	urlJefe := beego.AppConfig.String("ProtocolAdmin") + "://" +
		beego.AppConfig.String("UrlcrudCore") +
		"/jefe_dependencia?query=TerceroId:" + id_coordinador_lab +
		",FechaFin__gte:" + fechaActual +
		",FechaInicio__lte:" + fechaActual

	var resJefe []models.JefeDependencia
	if err := request.GetJson(urlJefe, &resJefe); err != nil {
		logs.Error("No se pudo obtener las dependencias del jefe %s: %v", id_coordinador_lab, err)
		return requestresponse.APIResponseDTO(false, 503, nil, "No se pudo consultar las dependencias del jefe de laboratorios.")
	}

	var dependenciasConNombre []map[string]interface{}

	for _, jefe := range resJefe {
		urlDep := beego.AppConfig.String("ProtocolAdmin") + "://" +
			beego.AppConfig.String("UrlcrudOikos") +
			"dependencia/" + fmt.Sprintf("%d", jefe.DependenciaId)

		var resDep map[string]interface{}
		if err := request.GetJson(urlDep, &resDep); err != nil {
			logs.Warn("No se pudo obtener información de la dependencia %d: %v", jefe.DependenciaId, err)
			continue
		}

		nombre := ""
		if n, ok := resDep["Nombre"].(string); ok {
			nombre = n
		}

		dependenciasConNombre = append(dependenciasConNombre, map[string]interface{}{
			"DependenciaId": jefe.DependenciaId,
			"Nombre":        nombre,
		})
	}

	// Filtrar solo dependencias cuyo nombre contiene "laboratorio" (ignorando mayúsculas/minúsculas)
	var laboratorios []map[string]interface{}
	for _, dep := range dependenciasConNombre {
		if nombre, ok := dep["Nombre"].(string); ok {
			if len(nombre) > 0 && (helpers.ContainsIgnoreCase(nombre, "laboratorio") || helpers.ContainsIgnoreCase(nombre, "laboratorios")) {
				laboratorios = append(laboratorios, dep)
			}
		}
	}

	// Si no hay dependencias de laboratorios, probablemente es el decano de la facultad
	if len(laboratorios) == 0 {
		logs.Info("No se encontraron dependencias de laboratorios. Probablemente es el decano de la facultad.")
		// Se usa el id obtenido en dependencias con nombre para armar el query y obtener el semaforo
		var idsDependencias []int
		for _, dep := range dependenciasConNombre {
			if id, ok := dep["DependenciaId"].(int); ok {
				idsDependencias = append(idsDependencias, id)
			}
		}
		if len(idsDependencias) == 0 {
			return requestresponse.APIResponseDTO(false, 404, nil, "No se encontraron dependencias asociadas al decano.")
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

		return obtenerSemaforosConFacultad(query, "No se encontraron estudiantes activos en las facultades asociadas al decano.", idFacultadOikos)

	} else {
		// Se consulta la dependencia padre de las dependencias obtenidas
		logs.Info("Dependencias de laboratorios encontradas: %d", len(laboratorios))

		var facultadesOikos []int
		facultadesMap := make(map[int]bool) // Para rastrear facultades únicas

		for _, lab := range laboratorios {
			labId := 0
			if id, ok := lab["DependenciaId"].(int); ok {
				labId = id
			}

			if labId == 0 {
				continue
			}

			// Consultar información completa del laboratorio para obtener su padre
			urlLabDep := beego.AppConfig.String("ProtocolAdmin") + "://" +
				beego.AppConfig.String("UrlcrudOikos") +
				"dependencia_padre/?query=Hija:" + fmt.Sprintf("%d", labId)

			var resLabDep []interface{}
			if err := request.GetJson(urlLabDep, &resLabDep); err != nil {
				logs.Warn("No se pudo obtener información completa del laboratorio %d: %v", labId, err)
				continue
			}

			// La respuesta es un array, procesar cada elemento
			for _, item := range resLabDep {
				if relacion, ok := item.(map[string]interface{}); ok {
					// Extraer el ID de la facultad desde el campo Padre
					if padre, ok := relacion["Padre"].(map[string]interface{}); ok {
						if padreId, ok := padre["Id"].(float64); ok {
							facultadId := int(padreId)

							// Agregar a la lista si no existe
							if !facultadesMap[facultadId] {
								facultadesMap[facultadId] = true
								facultadesOikos = append(facultadesOikos, facultadId)

								nombreFacultad := ""
								if nombre, ok := padre["Nombre"].(string); ok {
									nombreFacultad = nombre
								}
								logs.Info("Facultad padre encontrada: ID %d (%s) para laboratorio ID %d", facultadId, nombreFacultad, labId)
							}
						}
					}
				}
			}
		}

		if len(facultadesOikos) == 0 {
			return requestresponse.APIResponseDTO(false, 404, nil, "No se encontraron facultades asociadas a los laboratorios del coordinador.")
		}

		// Alertar si se encontraron múltiples facultades diferentes
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

		return obtenerSemaforosConFacultad(query, "No se encontraron estudiantes activos en las facultades asociadas a los laboratorios.", idFacultadOikos)
	}
}

func obtenerSemaforos(query, notFoundMsg string) requestresponse.APIResponse {
	var res map[string]interface{}
	var semaforos []models.Semaforo

	url := beego.AppConfig.String("ProtocolAdmin") + "://" +
		beego.AppConfig.String("UrlCrudPazySalvos") + "/semaforo/" + query

	if err := request.GetJson(url, &res); err != nil {
		logs.Error("Error al consultar paz_y_salvos:", err)
		return requestresponse.APIResponseDTO(false, 503, nil, "Error al consultar los datos del semáforo.")
	}

	data, ok := res["Data"].([]interface{})
	if !ok || len(data) == 0 {
		return requestresponse.APIResponseDTO(false, 404, nil, notFoundMsg)
	}

	dataBytes, err := json.Marshal(data)
	if err != nil {
		logs.Error("Error al serializar datos:", err)
		return requestresponse.APIResponseDTO(false, 500, nil, "Error al procesar los datos del semáforo.")
	}

	if err := json.Unmarshal(dataBytes, &semaforos); err != nil {
		logs.Error("Error al convertir datos a estructura:", err)
		return requestresponse.APIResponseDTO(false, 500, nil, "Error interno al interpretar los datos del semáforo.")
	}

	// Consultar el total de registros (sin limit)
	totalCount := 0
	queryCount := query
	// Remover limit y offset del query para contar todos
	if strings.Contains(queryCount, "&limit=") {
		parts := strings.Split(queryCount, "&limit=")
		queryCount = parts[0] + "&limit=-1"
	}
	if strings.Contains(queryCount, "&offset=") {
		queryCount = strings.Split(queryCount, "&offset=")[0]
	}

	var resCount map[string]interface{}
	urlCount := beego.AppConfig.String("ProtocolAdmin") + "://" +
		beego.AppConfig.String("UrlCrudPazySalvos") + "/semaforo/" + queryCount

	if err := request.GetJson(urlCount, &resCount); err == nil {
		if dataCount, ok := resCount["Data"].([]interface{}); ok {
			totalCount = len(dataCount)
		}
	}

	// Reutiliza la lógica de enriquecimiento
	tabla := consultarDataSemaforo(semaforos)

	// Retornar con metadatos de paginación
	result := map[string]interface{}{
		"Data":       tabla,
		"TotalCount": totalCount,
		"Limit":      len(semaforos),
	}

	return requestresponse.APIResponseDTO(true, 200, result, "Consulta exitosa")
}

// obtenerSemaforosConFacultad es similar a obtenerSemaforos pero incluye el IdFacultadOikos en la respuesta
func obtenerSemaforosConFacultad(query, notFoundMsg string, idFacultadOikos int) requestresponse.APIResponse {
	var res map[string]interface{}
	var semaforos []models.Semaforo

	url := beego.AppConfig.String("ProtocolAdmin") + "://" +
		beego.AppConfig.String("UrlCrudPazySalvos") + "/semaforo/" + query

	if err := request.GetJson(url, &res); err != nil {
		logs.Error("Error al consultar paz_y_salvos:", err)
		return requestresponse.APIResponseDTO(false, 503, nil, "Error al consultar los datos del semáforo.")
	}

	data, ok := res["Data"].([]interface{})
	if !ok || len(data) == 0 {
		return requestresponse.APIResponseDTO(false, 404, nil, notFoundMsg)
	}

	dataBytes, err := json.Marshal(data)
	if err != nil {
		logs.Error("Error al serializar datos:", err)
		return requestresponse.APIResponseDTO(false, 500, nil, "Error al procesar los datos del semáforo.")
	}

	if err := json.Unmarshal(dataBytes, &semaforos); err != nil {
		logs.Error("Error al convertir datos a estructura:", err)
		return requestresponse.APIResponseDTO(false, 500, nil, "Error interno al interpretar los datos del semáforo.")
	}

	// Consultar el total de registros (sin limit)
	totalCount := 0
	queryCount := query
	// Remover limit y offset del query para contar todos
	if strings.Contains(queryCount, "&limit=") {
		parts := strings.Split(queryCount, "&limit=")
		queryCount = parts[0] + "&limit=-1"
	}
	if strings.Contains(queryCount, "&offset=") {
		queryCount = strings.Split(queryCount, "&offset=")[0]
	}

	var resCount map[string]interface{}
	urlCount := beego.AppConfig.String("ProtocolAdmin") + "://" +
		beego.AppConfig.String("UrlCrudPazySalvos") + "/semaforo/" + queryCount

	if err := request.GetJson(urlCount, &resCount); err == nil {
		if dataCount, ok := resCount["Data"].([]interface{}); ok {
			totalCount = len(dataCount)
		}
	}

	tabla := consultarDataSemaforo(semaforos)

	result := map[string]interface{}{
		"Data":            tabla,
		"TotalCount":      totalCount,
		"Limit":           len(semaforos),
		"IdFacultadOikos": idFacultadOikos,
	}

	return requestresponse.APIResponseDTO(true, 200, result, "Consulta exitosa")
}

func consultarDataSemaforo(semaforos []models.Semaforo) []models.SemaforoTable {
	var result []models.SemaforoTable

	for _, s := range semaforos {
		var nombreFacultad, nombreProyecto, nombreEstudiante string

		// 1. Nombre del estudiante
		urlEst := beego.AppConfig.String("ProtocolAdmin") + "://" +
			beego.AppConfig.String("UrlcrudWSO2") +
			beego.AppConfig.String("NscrudAcademica") +
			"/datos_basicos_estudiante/" + fmt.Sprintf("%0.f", s.CodigoEstudiante)

		var resEst map[string]interface{}
		if err := request.GetJsonWSO2(urlEst, &resEst); err != nil {
			logs.Warn("No se pudo obtener nombre del estudiante %0.f: %v", s.CodigoEstudiante, err)
		} else {
			if datosCollection, ok := resEst["datosEstudianteCollection"].(map[string]interface{}); ok {
				if lista, ok := datosCollection["datosBasicosEstudiante"].([]interface{}); ok && len(lista) == 1 {
					if estudiante, ok := lista[0].(map[string]interface{}); ok {
						if nombre, ok := estudiante["nombre"].(string); ok {
							nombreEstudiante = nombre
						}
					}
				}
			}
		}

		// 2. Nombre de la facultad
		urlFac := beego.AppConfig.String("ProtocolAdmin") + "://" +
			beego.AppConfig.String("UrlcrudOikos") +
			"dependencia/" + fmt.Sprintf("%d", s.IdFacultadOikos)

		var resFac map[string]interface{}
		if err := request.GetJson(urlFac, &resFac); err != nil {
			logs.Warn("No se pudo obtener nombre de la facultad %d: %v", s.IdFacultadOikos, err)
		} else {
			if nombre, ok := resFac["Nombre"].(string); ok {
				nombreFacultad = nombre
			}
		}

		// // 3. Nombre del proyecto
		urlProj := beego.AppConfig.String("ProtocolAdmin") + "://" +
			beego.AppConfig.String("UrlcrudOikos") +
			"dependencia/" + fmt.Sprintf("%d", s.IdProyectoOikos)

		var resProj map[string]interface{}
		if err := request.GetJson(urlProj, &resProj); err != nil {
			logs.Warn("No se pudo obtener nombre del proyecto %d: %v", s.IdProyectoOikos, err)
		} else {
			if nombre, ok := resProj["Nombre"].(string); ok {
				nombreProyecto = nombre
			}
		}

		result = append(result, models.SemaforoTable{
			Id:               s.Id,
			CodigoEstudiante: s.CodigoEstudiante,
			NombreEstudiante: nombreEstudiante,
			NombreFacultad:   nombreFacultad,
			NombreProyecto:   nombreProyecto,
			AnioInsGrado:     s.AnioInsGrado,
			PerInsGrado:      s.PerInsGrado,
			Academico:        s.Academico,
			Financiero:       s.Financiero,
			Biblioteca:       s.Biblioteca,
			Laboratorios:     s.Laboratorios,
			Bienestar:        s.Bienestar,
			Urelinter:        s.Urelinter,
			Orc:              s.Orc,
			Observacion:      s.Observacion,
		})
	}

	return result
}
