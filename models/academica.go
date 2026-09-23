package models

import (
	"bytes"
	"encoding/json"
)

// academicaOneOrMany tolera la representación variable de colecciones de
// Académica: objeto para un resultado y arreglo para múltiples resultados.
type academicaOneOrMany[T any] []T

func (items *academicaOneOrMany[T]) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if bytes.Equal(data, []byte("null")) || len(data) == 0 {
		*items = nil
		return nil
	}
	if data[0] == '[' {
		return json.Unmarshal(data, (*[]T)(items))
	}

	var item T
	if err := json.Unmarshal(data, &item); err != nil {
		return err
	}
	*items = []T{item}
	return nil
}

type AsistenteProyectoResponse struct {
	Asistente struct {
		Proyectos academicaOneOrMany[ProyectoAcademica] `json:"proyectos"`
	} `json:"asistente"`
}

type ProyectoAcademica struct {
	Proyecto string `json:"proyecto"`
}

type CoordinadorCarreraResponse struct {
	CoordinadorCollection struct {
		Coordinadores academicaOneOrMany[CoordinadorCarrera] `json:"coordinador"`
	} `json:"coordinadorCollection"`
}

type CoordinadorCarrera struct {
	CodigoCondor string `json:"codigo_condor"`
}

type FacultadesSecretariaResponse struct {
	Facultades struct {
		Secretarias academicaOneOrMany[FacultadSecretaria] `json:"secretaria"`
	} `json:"facultades"`
}

type FacultadSecretaria struct {
	Codigo string `json:"SEC_DEP_COD"`
}

type DatosEstudianteResponse struct {
	DatosEstudianteCollection struct {
		DatosBasicos academicaOneOrMany[DatosBasicosEstudiante] `json:"datosBasicosEstudiante"`
	} `json:"datosEstudianteCollection"`
}

type DatosBasicosEstudiante struct {
	Nombre string `json:"nombre"`
}
