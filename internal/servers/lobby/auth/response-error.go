package auth

import "encoding/json"

type ResponseError struct {
	ErrorMessage string `json:"error_message"`
}

func (re ResponseError) ToJSON() []byte {
	data, _ := json.Marshal(re)
	return data
}

func NewResponseError(message string) ResponseError {
	return ResponseError{
		ErrorMessage: message,
	}
}
