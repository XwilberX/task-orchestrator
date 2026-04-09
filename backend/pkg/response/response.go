package response

import (
	"encoding/json"
	"net/http"
)

type Envelope struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Message string      `json:"message,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func write(w http.ResponseWriter, status int, body Envelope) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

func OK(w http.ResponseWriter, data interface{}, message string) {
	write(w, http.StatusOK, Envelope{Success: true, Data: data, Message: message})
}

func Created(w http.ResponseWriter, data interface{}) {
	write(w, http.StatusCreated, Envelope{Success: true, Data: data})
}

func BadRequest(w http.ResponseWriter, err error, message string) {
	write(w, http.StatusBadRequest, Envelope{Success: false, Message: message, Error: err.Error()})
}

func NotFound(w http.ResponseWriter, message string) {
	write(w, http.StatusNotFound, Envelope{Success: false, Message: message})
}

func Conflict(w http.ResponseWriter, message string) {
	write(w, http.StatusConflict, Envelope{Success: false, Message: message})
}

func Unauthorized(w http.ResponseWriter) {
	write(w, http.StatusUnauthorized, Envelope{Success: false, Message: "API key inválida o ausente"})
}

func InternalError(w http.ResponseWriter, err error) {
	write(w, http.StatusInternalServerError, Envelope{Success: false, Message: "Error interno del servidor", Error: err.Error()})
}
