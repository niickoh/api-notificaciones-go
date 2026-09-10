package handlers

import (
	"fmt"
	"net/http"

	"github.com/niickoh/api-notificaciones-go/models"
	"github.com/niickoh/api-notificaciones-go/services"
)

// WhatsAppHandler agrupa los endpoints de WhatsApp.
type WhatsAppHandler struct {
	sender services.WhatsAppSender
}

// NewWhatsAppHandler crea un handler de WhatsApp.
func NewWhatsAppHandler(sender services.WhatsAppSender) *WhatsAppHandler {
	return &WhatsAppHandler{sender: sender}
}

// Bienvenida procesa POST /whatsapp/bienvenida.
func (h *WhatsAppHandler) Bienvenida(w http.ResponseWriter, r *http.Request) {
	h.handle(w, r, "bienvenida")
}

// Contacto procesa POST /whatsapp/contacto.
func (h *WhatsAppHandler) Contacto(w http.ResponseWriter, r *http.Request) {
	h.handle(w, r, "contacto")
}

// ContactoUPAC procesa POST /whatsapp/contacto-upac.
func (h *WhatsAppHandler) ContactoUPAC(w http.ResponseWriter, r *http.Request) {
	h.handle(w, r, "contacto-upac")
}

func (h *WhatsAppHandler) handle(w http.ResponseWriter, r *http.Request, messageType string) {
	if r.Method != http.MethodPost {
		WriteError(w, http.StatusMethodNotAllowed, "Método no permitido")
		return
	}

	var payload models.NotificationRequest
	if err := decodeRequest(r, &payload); err != nil {
		WriteError(w, http.StatusBadRequest, "JSON inválido")
		return
	}

	if err := services.ValidateNotification(payload, true); err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.sender.Send(messageType, payload)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Sprintf("Fallo en integración WhatsApp: %v", err))
		return
	}

	WriteJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Mensaje de WhatsApp enviado exitosamente",
		Data: models.SuccessData{
			MessageID: result.MessageID,
			Timestamp: result.Timestamp.UTC().Format("2006-01-02T15:04:05Z"),
		},
	})
}
