package services

import (
	"fmt"
	"net/mail"
	"regexp"
	"strings"

	"github.com/niickoh/api-notificaciones-go/models"
)

var phoneRegex = regexp.MustCompile(`^\+\d{10,15}$`)

// ValidateNotification valida el payload base y puede exigir teléfono para WhatsApp.
func ValidateNotification(req models.NotificationRequest, requirePhone bool) error {
	if strings.TrimSpace(req.EmailFrom) == "" {
		return fmt.Errorf("email_from es obligatorio")
	}
	if _, err := mail.ParseAddress(req.EmailFrom); err != nil {
		return fmt.Errorf("email_from debe tener un formato válido")
	}

	if strings.TrimSpace(req.EmailTo) == "" {
		return fmt.Errorf("email_to es obligatorio")
	}
	if _, err := mail.ParseAddress(req.EmailTo); err != nil {
		return fmt.Errorf("email_to debe tener un formato válido")
	}

	phone := strings.TrimSpace(req.Telefono)
	if requirePhone && phone == "" {
		return fmt.Errorf("telefono es obligatorio para endpoints de whatsapp")
	}
	if phone != "" && !phoneRegex.MatchString(phone) {
		return fmt.Errorf("telefono debe tener formato +56912345678")
	}

	return nil
}
