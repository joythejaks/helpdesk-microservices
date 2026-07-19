package response

import (
	"encoding/json"
	"net/http"
)

// Response matches the {success, message, data, error} shape the other
// three services already return, so the frontend parses one shape
// regardless of which service answered.
type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func Success(w http.ResponseWriter, data interface{}) {
	write(w, http.StatusOK, Response{Success: true, Message: "success", Data: data})
}

func Error(w http.ResponseWriter, code int, message, errCode string) {
	write(w, code, Response{Success: false, Message: message, Error: errCode})
}

func write(w http.ResponseWriter, code int, body Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(body)
}
