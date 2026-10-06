package models

type DependenciaOikos struct {
	Id     int    `json:"Id"`
	Nombre string `json:"Nombre"`
}

type DependenciaPadreOikos struct {
	Padre DependenciaOikos `json:"Padre"`
	Hija  DependenciaOikos `json:"Hija"`
}
