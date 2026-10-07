package models

type PazSalvoGrado struct {
	Id               int    `json:"Id"`
	SolicitudGradoId int    `json:"SolicitudGradoId"`
	TipoPazSalvoId   int    `json:"TipoPazSalvoId"`
	TipoCodigo       string `json:"TipoCodigo"`
}

type HistorialPazSalvoGrado struct {
	Id                int    `json:"Id"`
	PazSalvoId        int    `json:"PazSalvoId"`
	TerceroId         int    `json:"TerceroId"`
	EstadoPazSalvoId  int    `json:"EstadoPazSalvoId"`
	EstadoCodigo      string `json:"EstadoCodigo"`
	Justificacion     string `json:"Justificacion"`
	FechaCreacion     string `json:"FechaCreacion"`
	FechaModificacion string `json:"FechaModificacion"`
}

type CheckPazSalvoGrado struct {
	PazSalvo     PazSalvoGrado            `json:"PazSalvo"`
	EstadoActual HistorialPazSalvoGrado   `json:"EstadoActual"`
	Historial    []HistorialPazSalvoGrado `json:"Historial"`
}

type PazSalvosSolicitudGrado struct {
	Solicitud  SolicitudGrado           `json:"Solicitud"`
	Estudiante *EstudianteRevisionGrado `json:"Estudiante,omitempty"`
	Programa   string                   `json:"Programa"`
	Periodo    string                   `json:"Periodo"`
	Checks     []CheckPazSalvoGrado     `json:"Checks"`
	Soportes   []SoporteBorradorGrado   `json:"Soportes"`
}

type PaginaPazSalvosGrado struct {
	Solicitudes      []PazSalvosSolicitudGrado `json:"Solicitudes"`
	Total            int                       `json:"Total"`
	TipoGestionado   string                    `json:"TipoGestionado"`
	TipoGestionadoId int                       `json:"TipoGestionadoId"`
}

type OpcionFiltroPazSalvoGrado struct {
	Id     int    `json:"Id"`
	Nombre string `json:"Nombre"`
}

type ProgramaFiltroPazSalvoGrado struct {
	Id            int    `json:"Id"`
	Nombre        string `json:"Nombre"`
	DependenciaId int    `json:"DependenciaId"`
	FacultadId    int    `json:"FacultadId"`
}

type FiltrosPazSalvoGrado struct {
	Periodos   []OpcionFiltroPazSalvoGrado   `json:"Periodos"`
	Facultades []OpcionFiltroPazSalvoGrado   `json:"Facultades"`
	Programas  []ProgramaFiltroPazSalvoGrado `json:"Programas"`
}

type DecidirPazSalvoGrado struct {
	Estado        string `json:"Estado"`
	Justificacion string `json:"Justificacion"`
	Perfil        string `json:"Perfil"`
}
