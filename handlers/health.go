package handlers

import (
	"net/http"

	"github.com/niickoh/api-notificaciones-go/models"
)

// Health expone el endpoint de salud sin autenticación.
func Health(w http.ResponseWriter, _ *http.Request) {
	WriteJSON(w, http.StatusOK, models.HealthResponse{
		Status:  "ok",
		Message: "API de notificaciones está funcionando correctamente",
	})
}
