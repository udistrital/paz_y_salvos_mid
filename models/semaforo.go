package models

import (
	"time"
)

type Semaforo struct {
	Id                int
	CodigoEstudiante  float64
	IdFacultadOikos   int16
	IdProyectoOikos   int16
	IdFacultadGedep   int16
	IdProyectoAccra   int16
	AnioInsGrado      float64
	PerInsGrado       float64
	Academico         bool
	Financiero        bool
	Biblioteca        bool
	Laboratorios      bool
	Bienestar         bool
	Urelinter         bool
	Orc               bool
	Observacion       string
	Activo            bool
	FechaCreacion     time.Time
	FechaModificacion time.Time
}

type SemaforoTable struct {
	Id               int
	CodigoEstudiante float64
	NombreEstudiante string
	NombreFacultad   string
	NombreProyecto   string
	AnioInsGrado     float64
	PerInsGrado      float64
	Academico        bool
	Financiero       bool
	Biblioteca       bool
	Laboratorios     bool
	Bienestar        bool
	Urelinter        bool
	Orc              bool
	Observacion      string
}
