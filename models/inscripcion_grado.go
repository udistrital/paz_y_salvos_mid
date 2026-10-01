package models

import "encoding/json"

// Las escrituras públicas solo incluyen selección y contenido del formulario;
// identidad, dependencia, eventos y estado se resuelven en el MID.
type CrearBorradorGrado struct {
	PeriodoId           int             `json:"PeriodoId"`
	ProgramaAcademicoId int             `json:"ProgramaAcademicoId"`
	Contenido           json.RawMessage `json:"Contenido"`
}

type GuardarBorradorGrado struct {
	Contenido json.RawMessage `json:"Contenido"`
}

type RadicarGrado struct {
	FormularioId int             `json:"FormularioId"`
	Contenido    json.RawMessage `json:"Contenido"`
}

type DirectorGrado struct {
	Identificacion string `json:"DIR_NRO_IDEN"`
	Nombre         string `json:"DIR_NOMBRE"`
	Apellido       string `json:"DIR_APELLIDO"`
	Estado         string `json:"DIR_ESTADO"`
}

type ModalidadGrado struct {
	Codigo      int64  `json:"AMG_COD"`
	Nombre      string `json:"AMG_NOMBRE"`
	Abreviatura string `json:"AMG_ABREVIATURA"`
	Estado      string `json:"AMG_ESTADO"`
}

type SolicitudGrado struct {
	Id                            int    `json:"Id"`
	TerceroId                     int    `json:"TerceroId"`
	CodigoEstudiante              string `json:"CodigoEstudiante"`
	PeriodoId                     int    `json:"PeriodoId"`
	ProgramaAcademicoId           int    `json:"ProgramaAcademicoId"`
	DependenciaOikosId            int    `json:"DependenciaOikosId"`
	CalendarioEventoInscripcionId int    `json:"CalendarioEventoInscripcionId"`
	CalendarioEventoAprobacionId  int    `json:"CalendarioEventoAprobacionId"`
}

type BorradorGrado struct {
	Solicitud  SolicitudGrado `json:"Solicitud"`
	Soportes   []SoporteGrado `json:"Soportes"`
	Formulario struct {
		Id              int             `json:"Id"`
		Contenido       json.RawMessage `json:"Contenido"`
		FechaRadicacion *string         `json:"FechaRadicacion"`
	} `json:"Formulario"`
}
