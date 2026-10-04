package models

import "encoding/json"

type CrearBorradorGrado struct {
	TerceroId           int             `json:"TerceroId"`
	PeriodoId           int             `json:"PeriodoId"`
	ProgramaAcademicoId int             `json:"ProgramaAcademicoId"`
	Contenido           json.RawMessage `json:"Contenido"`
}

type GuardarBorradorGrado struct {
	TerceroId int             `json:"TerceroId"`
	Contenido json.RawMessage `json:"Contenido"`
}

type RadicarGrado struct {
	TerceroId    int             `json:"TerceroId"`
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
	Id                int     `json:"Id"`
	Nombre            string  `json:"Nombre"`
	CodigoAbreviacion string  `json:"CodigoAbreviacion"`
	Activo            bool    `json:"Activo"`
	NumeroOrden       float64 `json:"NumeroOrden"`
}

type PaisExpedicionGrado struct {
	Id     int    `json:"Id"`
	Nombre string `json:"Nombre"`
}

type DepartamentoExpedicionGrado struct {
	Id         int    `json:"Id"`
	Nombre     string `json:"Nombre"`
	PaisId     int    `json:"PaisId"`
	PaisNombre string `json:"PaisNombre"`
}

type LugarExpedicionGrado struct {
	Id                 int    `json:"Id"`
	Nombre             string `json:"Nombre"`
	DepartamentoId     int    `json:"DepartamentoId"`
	DepartamentoNombre string `json:"DepartamentoNombre"`
	PaisId             int    `json:"PaisId"`
	PaisNombre         string `json:"PaisNombre"`
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
