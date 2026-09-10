package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/niickoh/api-notificaciones-go/models"
)

// WriteJSON escribe una respuesta JSON con el status especificado.
func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// WriteError retorna el formato estándar de error.
func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, models.APIResponse{
		Success: false,
		Message: message,
		Data:    nil,
	})
}
