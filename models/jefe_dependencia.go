package models

import "time"

// JefeDependencia representa la estructura del objeto jefe_dependencia
type JefeDependencia struct {
	Id             int       `json:"Id"`
	TerceroId      int       `json:"TerceroId"`
	DependenciaId  int       `json:"DependenciaId"`
	FechaInicio    time.Time `json:"FechaInicio"`
	FechaFin       time.Time `json:"FechaFin"`
	ActaAprobacion string    `json:"ActaAprobacion"`
}
