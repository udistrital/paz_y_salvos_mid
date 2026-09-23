package models

// APIResponse is the response envelope exposed by the MID endpoints.
type APIResponse struct {
	Success bool        `json:"Success"`
	Status  int         `json:"Status"`
	Message interface{} `json:"Message"`
	Data    interface{} `json:"Data"`
}

// APIResponseData tipa el contenido de respuestas consumidas desde otras APIs.
type APIResponseData[T any] struct {
	Success bool        `json:"Success"`
	Status  int         `json:"Status"`
	Message interface{} `json:"Message"`
	Data    T           `json:"Data"`
}
