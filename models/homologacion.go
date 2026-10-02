package models

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type homologacionId int

func (value *homologacionId) UnmarshalJSON(data []byte) error {
	var number int
	if err := json.Unmarshal(data, &number); err == nil {
		*value = homologacionId(number)
		return nil
	}

	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return fmt.Errorf("el identificador de Oikos debe ser entero o cadena numérica: %w", err)
	}
	number, err := strconv.Atoi(text)
	if err != nil {
		return fmt.Errorf("el identificador de Oikos %q no es entero: %w", text, err)
	}
	*value = homologacionId(number)
	return nil
}

type HomologacionResponse struct {
	Homologacion struct {
		IdOikos homologacionId `json:"id_oikos"`
	} `json:"homologacion"`
}

func (response HomologacionResponse) OikosId() int {
	return int(response.Homologacion.IdOikos)
}
