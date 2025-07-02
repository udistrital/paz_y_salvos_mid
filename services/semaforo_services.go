package services

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
	"github.com/udistrital/paz_y_salvos_mid/models"
	"github.com/udistrital/utils_oas/request"
	"github.com/udistrital/utils_oas/requestresponse"
)

func ConsultarEstudiante(codigo string) requestresponse.APIResponse {
	query := "?query=CodigoEstudiante:" + codigo + ",Activo:true&limit=-1"
	return obtenerSemaforos(query, "No se encontró información del estudiante.")
}

func ConsultarEstudiantes() requestresponse.APIResponse {
	query := "?query=Activo:true&limit=-1"
	return obtenerSemaforos(query, "No se encontraron estudiantes activos.")
}

func ConsultarEstudiantesProyecto(id_coordinador string) requestresponse.APIResponse {
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

	// 3. Construir query con OR
	query := "?query=IdProyectoOikos:"
	for i, id := range idsOikos {
		if i > 0 {
			query += "|"
		}
		query += fmt.Sprintf("%d", id)
	}
	query += ",Activo:true&limit=-1"
	return obtenerSemaforos(query, "No se encontraron estudiantes activos para los proyectos del coordinador.")
}

func ConsultarEstudiantesFacultad(id_secretario string) requestresponse.APIResponse {
	// 1. Consultar facultades del secretario
	urlSec := beego.AppConfig.String("ProtocolAdmin") + "://" +
		beego.AppConfig.String("UrlcrudWSO2") +
		beego.AppConfig.String("NscrudAcademica") +
		"/secretario_facultad/" + id_secretario

	var resSec map[string]interface{}
	if err := request.GetJsonWSO2(urlSec, &resSec); err != nil {
		logs.Error("No se pudo obtener las facultades del secretario %s: %v", id_secretario, err)
		return requestresponse.APIResponseDTO(false, 503, nil, "No se pudo consultar las facultades del secretario.")
	}

	var codigosCondor []string
	if collection, ok := resSec["secretarioCollection"].(map[string]interface{}); ok {
		if lista, ok := collection["secretario"].([]interface{}); ok {
			for _, item := range lista {
				if facultad, ok := item.(map[string]interface{}); ok {
					if cod, ok := facultad["codigo_condor"].(string); ok {
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

	// 3. Construir query con OR
	query := "?query=IdFacultadOikos:"
	for i, id := range idsOikos {
		if i > 0 {
			query += "|"
		}
		query += fmt.Sprintf("%d", id)
	}
	query += ",Activo:true&limit=-1"

	return obtenerSemaforos(query, "No se encontraron estudiantes activos en las facultades del secretario.")
}

func obtenerSemaforos(query, notFoundMsg string) requestresponse.APIResponse {
	var res map[string]interface{}
	var semaforos []models.Semaforo

	url := beego.AppConfig.String("ProtocolAdmin") + "://" +
		beego.AppConfig.String("UrlCrudPazySalvos") + "/semaforo/" + query

	fmt.Println("URL de consulta:", url)

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

	// Reutiliza la lógica de enriquecimiento
	tabla := consultarDataSemaforo(semaforos)
	return requestresponse.APIResponseDTO(true, 200, tabla, "Consulta exitosa")
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
