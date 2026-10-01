package models

type SoporteGrado struct {
	Id                         int  `json:"Id"`
	SolicitudGradoId           int  `json:"SolicitudGradoId"`
	FormularioSolicitudGradoId int  `json:"FormularioSolicitudGradoId"`
	TipoDocumentoId            int  `json:"TipoDocumentoId"`
	DocumentoId                int  `json:"DocumentoId"`
	Activo                     bool `json:"Activo"`
}

type CargarSoporteGrado struct {
	FormularioId    int    `json:"FormularioId"`
	SoporteActualId int    `json:"SoporteActualId"`
	Nombre          string `json:"Nombre"`
	MimeType        string `json:"MimeType"`
	Archivo         string `json:"Archivo"`
}

type SoporteBorradorGrado struct {
	Id           int    `json:"Id"`
	FormularioId int    `json:"FormularioId"`
	TipoSoporte  string `json:"TipoSoporte"`
	DocumentoId  int    `json:"DocumentoId"`
	Nombre       string `json:"Nombre"`
}

type ArchivoSoporteGrado struct {
	Nombre   string `json:"Nombre"`
	MimeType string `json:"MimeType"`
	Archivo  string `json:"Archivo"`
}
