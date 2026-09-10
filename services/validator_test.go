package services

import (
	"testing"

	"github.com/niickoh/api-notificaciones-go/models"
)

func TestValidateNotificationRejectsInvalidEmail(t *testing.T) {
	err := ValidateNotification(models.NotificationRequest{EmailFrom: "invalid", EmailTo: "dest@test.com"}, false)
	if err == nil {
		t.Fatal("expected invalid email error")
	}
}

func TestValidateNotificationRequiresWhatsAppPhone(t *testing.T) {
	err := ValidateNotification(models.NotificationRequest{EmailFrom: "from@test.com", EmailTo: "dest@test.com"}, true)
	if err == nil {
		t.Fatal("expected missing phone error")
	}
}

func TestValidateNotificationAcceptsValidPhone(t *testing.T) {
	err := ValidateNotification(models.NotificationRequest{EmailFrom: "from@test.com", EmailTo: "dest@test.com", Telefono: "+56912345678"}, true)
	if err != nil {
		t.Fatalf("expected valid payload, got %v", err)
	}
}
