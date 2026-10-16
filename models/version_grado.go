package models

import "encoding/json"

type VersionGradoResumen struct {
	FormularioId      int     `json:"FormularioId"`
	Version           int     `json:"Version"`
	FechaRadicacion   string  `json:"FechaRadicacion"`
	EstadoSolicitudId int     `json:"EstadoSolicitudId"`
	EstadoCodigo      string  `json:"EstadoCodigo"`
	Justificacion     *string `json:"Justificacion"`
}

type ActuacionSolicitudVersion struct {
	Id                int     `json:"Id"`
	FormularioId      *int    `json:"FormularioSolicitudGradoId"`
	TerceroId         int     `json:"TerceroId"`
	EstadoSolicitudId int     `json:"EstadoSolicitudId"`
	EstadoCodigo      string  `json:"EstadoCodigo"`
	Justificacion     *string `json:"Justificacion"`
	FechaCreacion     string  `json:"FechaCreacion"`
}

type ActuacionSoporteVersion struct {
	Id              int    `json:"Id"`
	SoporteGradoId  int    `json:"SoporteGradoId"`
	TerceroId       int    `json:"TerceroId"`
	EstadoSoporteId int    `json:"EstadoSoporteId"`
	EstadoCodigo    string `json:"EstadoCodigo"`
	Observacion     string `json:"Observacion"`
	FechaCreacion   string `json:"FechaCreacion"`
}

type SoporteVersionGrado struct {
	Soporte struct {
		Id                         int `json:"Id"`
		SolicitudGradoId           int `json:"SolicitudGradoId"`
		FormularioSolicitudGradoId int `json:"FormularioSolicitudGradoId"`
		DocumentoId                int `json:"DocumentoId"`
		TipoDocumentoId            int `json:"TipoDocumentoId"`
	} `json:"Soporte"`
	TipoSoporte  string                    `json:"TipoSoporte"`
	EstadoActual ActuacionSoporteVersion   `json:"EstadoActual"`
	Historial    []ActuacionSoporteVersion `json:"Historial"`
}

type VersionGradoDetalle struct {
	Solicitud  SolicitudGrado `json:"Solicitud"`
	Formulario struct {
		Id              int             `json:"Id"`
		Version         int             `json:"Version"`
		Contenido       json.RawMessage `json:"Contenido"`
		FechaRadicacion *string         `json:"FechaRadicacion"`
	} `json:"Formulario"`
	Estado      string `json:"Estado"`
	Referencias struct {
		Modalidad string `json:"Modalidad"`
		Director1 string `json:"Director1"`
		Director2 string `json:"Director2"`
	} `json:"Referencias"`
	Historial []ActuacionSolicitudVersion `json:"Historial"`
	Soportes  []SoporteVersionGrado       `json:"Soportes"`
}
