package models

type DependenciaOikos struct {
	Id     int    `json:"Id"`
	Nombre string `json:"Nombre"`
}

type DependenciaPadreOikos struct {
	Padre DependenciaOikos `json:"Padre"`
	Hija  DependenciaOikos `json:"Hija"`
}

type DependenciaOikosV2 struct {
	Id     int    `json:"Id"`
	Nombre string `json:"Nombre"`
	Activo bool   `json:"Activo"`
}

type DependenciaPadreOikosV2 struct {
	Id      int                `json:"Id"`
	PadreId DependenciaOikosV2 `json:"PadreId"`
	HijaId  DependenciaOikosV2 `json:"HijaId"`
	Activo  bool               `json:"Activo"`
}
