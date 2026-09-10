package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/niickoh/api-notificaciones-go/config"
	"github.com/niickoh/api-notificaciones-go/models"
)

const whatsappEndpointBase = "https://graph.instagram.com/v18.0"

// WhatsAppSender abstrae el envío de mensajes de WhatsApp.
type WhatsAppSender interface {
	Send(messageType string, payload models.NotificationRequest) (models.NotificationResult, error)
}

// WhatsAppService integra con Meta WhatsApp.
type WhatsAppService struct {
	token        string
	phoneID      string
	endpointBase string
	client       *http.Client
}

// NewWhatsAppService construye el servicio de WhatsApp.
func NewWhatsAppService(cfg config.Config) *WhatsAppService {
	return &WhatsAppService{
		token:        cfg.MetaWhatsAppToken,
		phoneID:      cfg.MetaWhatsAppPhone,
		endpointBase: whatsappEndpointBase,
		client:       &http.Client{Timeout: 10 * time.Second},
	}
}

// Send envía un texto plano al endpoint de Meta.
func (s *WhatsAppService) Send(messageType string, payload models.NotificationRequest) (models.NotificationResult, error) {
	if s.token == "" || s.phoneID == "" {
		return models.NotificationResult{}, errors.New("credenciales de WhatsApp no configuradas")
	}

	requestBody := map[string]any{
		"messaging_product": "whatsapp",
		"to":                payload.Telefono,
		"type":              "text",
		"text": map[string]string{
			"body": buildWhatsAppMessage(messageType, payload),
		},
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return models.NotificationResult{}, err
	}

	endpoint := fmt.Sprintf("%s/%s/messages", strings.TrimRight(s.endpointBase, "/"), s.phoneID)
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return models.NotificationResult{}, err
	}
	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return models.NotificationResult{}, err
	}
	defer resp.Body.Close()

	responseBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return models.NotificationResult{}, fmt.Errorf("whatsapp devolvió status %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	var parsed struct {
		Messages []struct {
			ID string `json:"id"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(responseBody, &parsed); err != nil {
		return models.NotificationResult{}, err
	}

	messageID := fmt.Sprintf("%d", time.Now().UnixNano())
	if len(parsed.Messages) > 0 && parsed.Messages[0].ID != "" {
		messageID = parsed.Messages[0].ID
	}

	return models.NotificationResult{MessageID: messageID, Timestamp: time.Now().UTC()}, nil
}

func buildWhatsAppMessage(messageType string, payload models.NotificationRequest) string {
	lines := []string{fmt.Sprintf("Notificación %s", messageType)}
	if payload.Nombres != "" || payload.Apellidos != "" {
		lines = append(lines, strings.TrimSpace(payload.Nombres+" "+payload.Apellidos))
	}
	if payload.Mensaje != "" {
		lines = append(lines, "Mensaje: "+payload.Mensaje)
	}
	if payload.Direccion != "" {
		lines = append(lines, "Dirección: "+payload.Direccion)
	}
	if payload.Comuna != "" {
		lines = append(lines, "Comuna: "+payload.Comuna)
	}
	if payload.EmailFrom != "" {
		lines = append(lines, "Contacto: "+payload.EmailFrom)
	}

	return strings.Join(lines, "\n")
}
