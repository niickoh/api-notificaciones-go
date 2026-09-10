package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/niickoh/api-notificaciones-go/config"
	"github.com/niickoh/api-notificaciones-go/models"
)

type stubMailSender struct{}

func (stubMailSender) Send(string, models.NotificationRequest) (models.NotificationResult, error) {
	return models.NotificationResult{MessageID: "mail-123", Timestamp: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)}, nil
}

type stubWhatsAppSender struct{}

func (stubWhatsAppSender) Send(string, models.NotificationRequest) (models.NotificationResult, error) {
	return models.NotificationResult{MessageID: "wa-123", Timestamp: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)}, nil
}

func TestHealthEndpointDoesNotRequireAuth(t *testing.T) {
	handler := newHandler(testConfig(), stubMailSender{}, stubWhatsAppSender{})
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if body["status"] != "ok" {
		t.Fatalf("expected status ok, got %q", body["status"])
	}
}

func TestProtectedEndpointRequiresJWT(t *testing.T) {
	handler := newHandler(testConfig(), stubMailSender{}, stubWhatsAppSender{})
	req := httptest.NewRequest(http.MethodPost, "/mail/bienvenida", bytes.NewBufferString(`{"email_from":"a@test.com","email_to":"b@test.com"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.Code)
	}
}

func TestMailEndpointReturnsSuccessPayload(t *testing.T) {
	handler := newHandler(testConfig(), stubMailSender{}, stubWhatsAppSender{})
	body := bytes.NewBufferString(`{"email_from":"a@test.com","email_to":"b@test.com","mensaje":"hola"}`)
	req := httptest.NewRequest(http.MethodPost, "/mail/bienvenida", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", bearerToken(t, "bascodelab-0317"))
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	var parsed struct {
		Success bool               `json:"success"`
		Message string             `json:"message"`
		Data    models.SuccessData `json:"data"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !parsed.Success || parsed.Message != "Correo enviado exitosamente" || parsed.Data.MessageID != "mail-123" {
		t.Fatalf("unexpected response: %+v", parsed)
	}
}

func TestWhatsAppEndpointRequiresPhone(t *testing.T) {
	handler := newHandler(testConfig(), stubMailSender{}, stubWhatsAppSender{})
	body := bytes.NewBufferString(`{"email_from":"a@test.com","email_to":"b@test.com"}`)
	req := httptest.NewRequest(http.MethodPost, "/whatsapp/contacto", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", bearerToken(t, "bascodelab-0317"))
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestRateLimitByIP(t *testing.T) {
	cfg := testConfig()
	cfg.RateLimitIP = 1
	handler := newHandler(cfg, stubMailSender{}, stubWhatsAppSender{})

	makeRequest := func() *httptest.ResponseRecorder {
		body := bytes.NewBufferString(`{"email_from":"a@test.com","email_to":"b@test.com","telefono":"+56912345678"}`)
		req := httptest.NewRequest(http.MethodPost, "/whatsapp/bienvenida", body)
		req.RemoteAddr = "127.0.0.1:12345"
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", bearerToken(t, "bascodelab-0317"))
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, req)
		return resp
	}

	first := makeRequest()
	if first.Code != http.StatusOK {
		t.Fatalf("expected first request 200, got %d body=%s", first.Code, first.Body.String())
	}

	second := makeRequest()
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("expected second request 429, got %d body=%s", second.Code, second.Body.String())
	}
}

func testConfig() config.Config {
	return config.Config{
		Port:            "4000",
		JWTSecret:       "bascodelab-0317",
		RateLimitIP:     100,
		RateLimitAPIKey: 1000,
		CORSOrigin:      "*",
		TemplatesDir:    "templates",
	}
}

func bearerToken(t *testing.T, secret string) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": "test", "exp": time.Now().Add(time.Hour).Unix()})
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}
	return "Bearer " + signed
}
