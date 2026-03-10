package models

import (
	"time"
)

// ProyectoAsignado representa un proyecto curricular homologado y su nombre
type ProyectoAsignado struct {
	IdOikos int    `json:"IdOikos"`
	Codigo  string `json:"Codigo"`
	Nombre  string `json:"Nombre"`
}

type Semaforo struct {
	Id                      int
	CodigoEstudiante        float64
	IdFacultadOikos         int16
	IdProyectoOikos         int16
	IdFacultadGedep         int16
	IdProyectoAccra         int16
	AnioInsGrado            float64
	PerInsGrado             float64
	Academico               bool
	Financiero              bool
	Biblioteca              bool
	Laboratorios            bool
	Bienestar               bool
	Urelinter               bool
	Orc                     *bool
	ObservacionCoordinacion string
	ObservacionBiblioteca   string
	ObservacionLaboratorios string
	ObservacionBienestar    string
	ObservacionUrelinter    string
	ObservacionOrc          string
	ObservacionFinanciera   string
	Activo                  bool
	FechaCreacion           time.Time
	FechaModificacion       time.Time
}

type SemaforoTable struct {
	Id                      int
	CodigoEstudiante        float64
	NombreEstudiante        string
	NombreFacultad          string
	NombreProyecto          string
	AnioInsGrado            float64
	PerInsGrado             float64
	Academico               bool
	Financiero              bool
	Biblioteca              bool
	Laboratorios            bool
	Bienestar               bool
	Urelinter               bool
	Orc                     *bool
	ObservacionCoordinacion string
	ObservacionBiblioteca   string
	ObservacionLaboratorios string
	ObservacionBienestar    string
	ObservacionUrelinter    string
	ObservacionOrc          string
	ObservacionFinanciera   string
}

type SemaforosAsistenteResponse struct {
	EsAsistente        bool               `json:"EsAsistente"`
	Semaforos          []SemaforoTable    `json:"Semaforos"`
	Limit              int                `json:"Limit"`
	TotalCount         int                `json:"TotalCount"`
	ProyectosAsignados []ProyectoAsignado `json:"ProyectosAsignados"`
}

type SemaforoCoordinadorResponse struct {
	Semaforos          []SemaforoTable    `json:"Semaforos"`
	Limit              int                `json:"Limit"`
	TotalCount         int                `json:"TotalCount"`
	ProyectosAsignados []ProyectoAsignado `json:"ProyectosAsignados"`
}
