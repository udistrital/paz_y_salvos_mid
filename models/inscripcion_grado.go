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

type RegistrarLugarExpedicionGrado struct {
	TerceroId int `json:"TerceroId" validate:"required,gt=0"`
	LugarId   int `json:"LugarId" validate:"required,gt=0"`
}

type RadicarGrado struct {
	TerceroId    int             `json:"TerceroId"`
	FormularioId int             `json:"FormularioId"`
	Contenido    json.RawMessage `json:"Contenido"`
}

type SubsanarGrado struct {
	TerceroId    int `json:"TerceroId"`
	FormularioId int `json:"FormularioId"`
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

type LugarExpedicionIdentificacionGrado struct {
	Registrado bool                  `json:"Registrado"`
	Lugar      *LugarExpedicionGrado `json:"Lugar"`
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
	Solicitud       SolicitudGrado       `json:"Solicitud"`
	Soportes        []SoporteGrado       `json:"Soportes"`
	EstadosSoportes []EstadoSoporteGrado `json:"EstadosSoportes"`
	Estado          string               `json:"Estado"`
	Comentario      *string              `json:"Comentario"`
	Formulario      struct {
		Id              int             `json:"Id"`
		Version         int             `json:"Version"`
		Contenido       json.RawMessage `json:"Contenido"`
		FechaRadicacion *string         `json:"FechaRadicacion"`
	} `json:"Formulario"`
	Historial struct {
		Id                int     `json:"Id"`
		EstadoSolicitudId int     `json:"EstadoSolicitudId"`
		Justificacion     *string `json:"Justificacion"`
	} `json:"Historial"`
}

type EstadoSoporteGrado struct {
	SoporteGradoId  int    `json:"SoporteGradoId"`
	EstadoSoporteId int    `json:"EstadoSoporteId"`
	Observacion     string `json:"Observacion"`
}

type SolicitudRevisionGrado struct {
	Solicitud  SolicitudGrado          `json:"Solicitud"`
	Estudiante EstudianteRevisionGrado `json:"Estudiante"`
	Programa   string                  `json:"Programa"`
	Periodo    string                  `json:"Periodo"`
	Formulario FormularioRevisionGrado `json:"Formulario"`
	Version    int                     `json:"Version"`
	Estado     string                  `json:"Estado"`
	Comentario *string                 `json:"Comentario"`
	RadicadaEn *string                 `json:"RadicadaEn"`
	Soportes   []SoporteBorradorGrado  `json:"Soportes"`
}

type PaginaRevisionGrado struct {
	Solicitudes []SolicitudRevisionGrado `json:"Solicitudes"`
	Total       int                      `json:"Total"`
}

type OpcionFiltroRevisionGrado struct {
	Id     int    `json:"Id"`
	Nombre string `json:"Nombre"`
}

type ProgramaFiltroRevisionGrado struct {
	Id            int    `json:"Id"`
	Nombre        string `json:"Nombre"`
	DependenciaId int    `json:"DependenciaId"`
}

type FiltrosRevisionGrado struct {
	Periodos  []OpcionFiltroRevisionGrado   `json:"Periodos"`
	Programas []ProgramaFiltroRevisionGrado `json:"Programas"`
}

type EstudianteRevisionGrado struct {
	NombreCompleto       string  `json:"NombreCompleto"`
	TipoIdentificacion   string  `json:"TipoIdentificacion"`
	NumeroIdentificacion string  `json:"NumeroIdentificacion"`
	FechaNacimiento      *string `json:"FechaNacimiento"`
	FechaExpedicion      *string `json:"FechaExpedicion"`
	Genero               string  `json:"Genero"`
	Telefono             string  `json:"Telefono"`
	TelefonoAlterno      string  `json:"TelefonoAlterno"`
	Correo               string  `json:"Correo"`
	Direccion            string  `json:"Direccion"`
}

type FormularioRevisionGrado struct {
	TrabajoGrado           string `json:"TrabajoGrado"`
	Modalidad              string `json:"Modalidad"`
	DirectorPrincipal      string `json:"DirectorPrincipal"`
	DirectorSecundario     string `json:"DirectorSecundario"`
	CiudadExpedicion       string `json:"CiudadExpedicion"`
	DepartamentoExpedicion string `json:"DepartamentoExpedicion"`
	PaisExpedicion         string `json:"PaisExpedicion"`
	NumeroActaSustentacion string `json:"NumeroActaSustentacion"`
	NumeroRegistroSNP      string `json:"NumeroRegistroSnp"`
	TrabajaActualmente     *bool  `json:"TrabajaActualmente"`
	Empresa                string `json:"Empresa"`
	DireccionEmpresa       string `json:"DireccionEmpresa"`
	TelefonoEmpresa        string `json:"TelefonoEmpresa"`
}

type DecisionSoporteRevisionGrado struct {
	SoporteId   int    `json:"SoporteId"`
	Observado   bool   `json:"Observado"`
	Observacion string `json:"Observacion"`
}

type RevisarDocumentacionGrado struct {
	FormularioId  int                            `json:"FormularioId"`
	Aprobada      bool                           `json:"Aprobada"`
	Justificacion string                         `json:"Justificacion"`
	Soportes      []DecisionSoporteRevisionGrado `json:"Soportes"`
}
