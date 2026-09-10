package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/niickoh/api-notificaciones-go/models"
	"github.com/niickoh/api-notificaciones-go/services"
)

// MailHandler agrupa los endpoints de correo.
type MailHandler struct {
	sender services.MailSender
}

// NewMailHandler crea un handler de correo.
func NewMailHandler(sender services.MailSender) *MailHandler {
	return &MailHandler{sender: sender}
}

// Bienvenida procesa POST /mail/bienvenida.
func (h *MailHandler) Bienvenida(w http.ResponseWriter, r *http.Request) {
	h.handle(w, r, "bienvenida")
}

// Contacto procesa POST /mail/contacto.
func (h *MailHandler) Contacto(w http.ResponseWriter, r *http.Request) {
	h.handle(w, r, "contacto")
}

// ContactoUPAC procesa POST /mail/contacto-upac.
func (h *MailHandler) ContactoUPAC(w http.ResponseWriter, r *http.Request) {
	h.handle(w, r, "contacto-upac")
}

func (h *MailHandler) handle(w http.ResponseWriter, r *http.Request, templateName string) {
	if r.Method != http.MethodPost {
		WriteError(w, http.StatusMethodNotAllowed, "Método no permitido")
		return
	}

	var payload models.NotificationRequest
	if err := decodeRequest(r, &payload); err != nil {
		WriteError(w, http.StatusBadRequest, "JSON inválido")
		return
	}

	if err := services.ValidateNotification(payload, false); err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.sender.Send(templateName, payload)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Sprintf("Fallo en integración Mailjet: %v", err))
		return
	}

	WriteJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Correo enviado exitosamente",
		Data: models.SuccessData{
			MessageID: result.MessageID,
			Timestamp: result.Timestamp.UTC().Format("2006-01-02T15:04:05Z"),
		},
	})
}

func decodeRequest(r *http.Request, destination any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if decoder.More() {
		return fmt.Errorf("unexpected trailing data")
	}
	return nil
}
