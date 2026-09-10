package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/niickoh/api-notificaciones-go/config"
	"github.com/niickoh/api-notificaciones-go/models"
)

const mailjetEndpoint = "https://api.mailjet.com/v3.1/send"

// MailSender abstrae el envío de correos.
type MailSender interface {
	Send(templateName string, payload models.NotificationRequest) (models.NotificationResult, error)
}

// MailjetService integra con Mailjet y renderiza templates HTML.
type MailjetService struct {
	apiKey       string
	apiSecret    string
	fromEmail    string
	fromName     string
	templatesDir string
	endpoint     string
	client       *http.Client
}

// NewMailjetService construye el servicio Mailjet.
func NewMailjetService(cfg config.Config) *MailjetService {
	return &MailjetService{
		apiKey:       cfg.MailjetAPIKey,
		apiSecret:    cfg.MailjetAPISecret,
		fromEmail:    cfg.MailjetFromEmail,
		fromName:     cfg.MailjetFromName,
		templatesDir: cfg.TemplatesDir,
		endpoint:     mailjetEndpoint,
		client:       &http.Client{Timeout: 10 * time.Second},
	}
}

// Send renderiza la plantilla y envía el correo a Mailjet.
func (s *MailjetService) Send(templateName string, payload models.NotificationRequest) (models.NotificationResult, error) {
	htmlBody, err := s.renderTemplate(templateName, payload)
	if err != nil {
		return models.NotificationResult{}, err
	}

	if s.apiKey == "" || s.apiSecret == "" {
		return models.NotificationResult{}, errors.New("credenciales de Mailjet no configuradas")
	}

	requestBody := map[string]any{
		"Messages": []map[string]any{{
			"From": map[string]string{
				"Email": s.fromEmail,
				"Name":  s.fromName,
			},
			"To": []map[string]string{{
				"Email": payload.EmailTo,
			}},
			"ReplyTo": map[string]string{
				"Email": payload.EmailFrom,
			},
			"Subject":  mailSubject(templateName),
			"HTMLPart": htmlBody,
		}},
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return models.NotificationResult{}, err
	}

	req, err := http.NewRequest(http.MethodPost, s.endpoint, bytes.NewReader(body))
	if err != nil {
		return models.NotificationResult{}, err
	}
	req.SetBasicAuth(s.apiKey, s.apiSecret)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return models.NotificationResult{}, err
	}
	defer resp.Body.Close()

	responseBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return models.NotificationResult{}, fmt.Errorf("mailjet devolvió status %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	var parsed struct {
		Messages []struct {
			To []struct {
				MessageID any `json:"MessageID"`
			} `json:"To"`
		} `json:"Messages"`
	}
	if err := json.Unmarshal(responseBody, &parsed); err != nil {
		return models.NotificationResult{}, err
	}

	messageID := fmt.Sprintf("%d", time.Now().UnixNano())
	if len(parsed.Messages) > 0 && len(parsed.Messages[0].To) > 0 && parsed.Messages[0].To[0].MessageID != nil {
		messageID = fmt.Sprintf("%v", parsed.Messages[0].To[0].MessageID)
	}

	return models.NotificationResult{MessageID: messageID, Timestamp: time.Now().UTC()}, nil
}

func (s *MailjetService) renderTemplate(templateName string, payload models.NotificationRequest) (string, error) {
	filePath := filepath.Join(s.templatesDir, templateName+".html")
	tmpl, err := template.ParseFiles(filePath)
	if err != nil {
		return "", err
	}

	var rendered bytes.Buffer
	if err := tmpl.Execute(&rendered, payload); err != nil {
		return "", err
	}

	return rendered.String(), nil
}

func mailSubject(templateName string) string {
	switch templateName {
	case "bienvenida":
		return "Bienvenida"
	case "contacto":
		return "Contacto"
	case "contacto-upac":
		return "Contacto UPAC"
	default:
		return "Notificación"
	}
}
