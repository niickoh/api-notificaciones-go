package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/niickoh/api-notificaciones-go/models"
)

// recordingMailSender guarda los correos que recibiría Mailjet.
type recordingMailSender struct {
	stubMailSender
	sent []models.MailMessage
}

func (s *recordingMailSender) SendMessages(m []models.MailMessage) (models.NotificationResult, error) {
	s.sent = append(s.sent, m...)
	return models.NotificationResult{MessageID: "mail-123", Timestamp: time.Now()}, nil
}

func postMail(t *testing.T, sender *recordingMailSender, path, body string, withJWT bool) *httptest.ResponseRecorder {
	t.Helper()
	cfg := testConfig()
	cfg.PortalURL = "https://dev.sportspherecenter.com/"
	cfg.ContactEmailTo = "admin@sportsphere.test"
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if withJWT {
		req.Header.Set("Authorization", bearerToken(t, "bascodelab-0317"))
	}
	resp := httptest.NewRecorder()
	newHandler(cfg, sender, stubWhatsAppSender{}).ServeHTTP(resp, req)
	return resp
}

// Un caso por plantilla, con los cuerpos de ejemplo del artifact.
func TestSportSphereTemplatesRender(t *testing.T) {
	cases := []struct {
		name, path, body string
		jwt              bool
		wantTo           []string
		wantSubject      string
		wantHTML         []string
	}{
		{
			name: "contacto", path: "/mail/contacto/", jwt: true,
			body:        `{"name":"Sofía <b>","email":"sofia@example.com","message":"Hola,\nquiero una demo.","club":"UPAC","telefono":"+56 9 1234 5678","deportistas":"30"}`,
			wantTo:      []string{"admin@sportsphere.test"},
			wantSubject: "Nuevo mensaje de contacto de Sofía <b>",
			wantHTML:    []string{"Formulario de contacto", "Sofía &lt;b&gt;", "Club: UPAC", "N° de deportistas: 30", "quiero una demo.", "sphere-mail-logo.png"},
		},
		{
			name: "aviso", path: "/mail/aviso", jwt: true,
			body:        `{"destinatarios":[{"email":"a@example.com","name":"Ana"},{"email":"b@example.com"}],"titulo":"Cambio de horario","mensaje":"El jueves entrenamos a las 19:00.","prioridad":"importante","autor":"Coach Camila","equipo":"Galaxy","url":"https://dev.sportspherecenter.com/"}`,
			wantTo:      []string{"a@example.com", "b@example.com"},
			wantSubject: "[IMPORTANTE] Cambio de horario",
			wantHTML:    []string{"Nuevo aviso", "Coach Camila", "Galaxy", "#f59e0b", "Ver aviso", "El jueves entrenamos"},
		},
		{
			name: "evento", path: "/mail/evento/", jwt: true,
			body:        `{"email":"atleta@example.com","name":"Sofía","titulo":"Entrenamiento técnico","tipo":"ENTRENAMIENTO","estado":"cancelado","fecha_inicio":"2026-11-17T18:30:00","fecha_fin":"2026-11-17T20:30:00","ubicacion":"Gimnasio UPAC","equipo":"Galaxy","descripcion":"Traer botella de agua.","url":"https://dev.sportspherecenter.com/"}`,
			wantTo:      []string{"atleta@example.com"},
			wantSubject: "Entrenamiento técnico · martes 17 de noviembre de 2026, 18:30",
			wantHTML:    []string{"Evento cancelado", "line-through", "martes 17 de noviembre de 2026, 20:30", "Gimnasio UPAC", "Traer botella de agua.", "Ver calendario"},
		},
		{
			name: "usuario creado", path: "/mail/bienvenidaNew/", jwt: true,
			body:        `{"email":"sofia@example.com","name":"Sofía","username":"sofia.r","password":"Temp2026!","rol":"Atleta","club":"UPAC All Stars"}`,
			wantTo:      []string{"sofia@example.com"},
			wantSubject: "Bienvenido/a a SportSphere Pro",
			wantHTML:    []string{"Cuenta creada", "sofia.r", "Temp2026!", "UPAC All Stars", "Atleta", "Iniciar sesión", "Si no esperabas este correo"},
		},
		{
			name: "asignación de equipo", path: "/mail/asignacionEquipo", jwt: true,
			body:        `{"email":"sofia@example.com","name":"Sofía","equipo":"Galaxy","rol":"Atleta","entrenador":"Camila Rojas","club":"UPAC","color":"#E91E63"}`,
			wantTo:      []string{"sofia@example.com"},
			wantSubject: "Ahora eres parte de Galaxy",
			wantHTML:    []string{"Asignación de equipo", "Galaxy", "Camila Rojas", "#E91E63", "Ver mi equipo"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sender := &recordingMailSender{}
			resp := postMail(t, sender, tc.path, tc.body, tc.jwt)
			if resp.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
			}
			if len(sender.sent) != len(tc.wantTo) {
				t.Fatalf("expected %d messages, got %d", len(tc.wantTo), len(sender.sent))
			}
			for i, m := range sender.sent {
				if m.To.Email != tc.wantTo[i] {
					t.Errorf("message %d to %q, want %q", i, m.To.Email, tc.wantTo[i])
				}
				if m.Subject != tc.wantSubject {
					t.Errorf("subject %q, want %q", m.Subject, tc.wantSubject)
				}
				for _, want := range tc.wantHTML {
					if !strings.Contains(m.HTML, want) {
						t.Errorf("HTML missing %q", want)
					}
				}
				if strings.Contains(m.HTML, "<no value>") || strings.Contains(m.HTML, "ZgotmplZ") {
					t.Errorf("HTML has unrendered/unsafe values")
				}
			}
		})
	}
}

func TestContactoSetsReplyToSender(t *testing.T) {
	sender := &recordingMailSender{}
	resp := postMail(t, sender, "/mail/contacto", `{"name":"Sofía","email":"sofia@example.com","message":"Hola"}`, true)
	if resp.Code != http.StatusOK || len(sender.sent) != 1 {
		t.Fatalf("expected 200 with 1 message, got %d body=%s", resp.Code, resp.Body.String())
	}
	if r := sender.sent[0].ReplyTo; r == nil || r.Email != "sofia@example.com" {
		t.Fatalf("unexpected ReplyTo %+v", r)
	}
}

func TestStaffTemplatesRequireJWT(t *testing.T) {
	for _, path := range []string{"/mail/contacto", "/mail/contacto/", "/mail/aviso", "/mail/evento/", "/mail/bienvenidaNew", "/mail/asignacionEquipo/"} {
		resp := postMail(t, &recordingMailSender{}, path, `{}`, false)
		if resp.Code != http.StatusUnauthorized {
			t.Errorf("%s: expected 401, got %d", path, resp.Code)
		}
	}
}

func TestSportSphereValidation(t *testing.T) {
	cases := map[string]struct{ path, body string }{
		"missing fields":      {"/mail/aviso", `{"email":"a@example.com","name":"Ana"}`},
		"invalid recipient":   {"/mail/aviso", `{"destinatarios":[{"email":"nope"}],"titulo":"t","mensaje":"m"}`},
		"empty destinatarios": {"/mail/evento", `{"destinatarios":[],"titulo":"t","fecha_inicio":"2026-11-17"}`},
		"unknown field":       {"/mail/bienvenidaNew", `{"email":"a@example.com","username":"u","extra":1}`},
		"contacto bad email":  {"/mail/contacto", `{"name":"n","email":"x","message":"m"}`},
		"too long":            {"/mail/aviso", `{"email":"a@example.com","titulo":"t","mensaje":"` + strings.Repeat("a", 5001) + `"}`},
	}
	for name, tc := range cases {
		sender := &recordingMailSender{}
		resp := postMail(t, sender, tc.path, tc.body, true)
		if resp.Code != http.StatusBadRequest || len(sender.sent) != 0 {
			t.Errorf("%s: expected 400 and nothing sent, got %d (%d sent) body=%s", name, resp.Code, len(sender.sent), resp.Body.String())
		}
	}
}

func TestAsignacionEquipoRejectsNonHexColor(t *testing.T) {
	sender := &recordingMailSender{}
	resp := postMail(t, sender, "/mail/asignacionEquipo", `{"email":"a@example.com","equipo":"Galaxy","color":"red;background:url(x)"}`, true)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	if html := sender.sent[0].HTML; strings.Contains(html, "url(x)") || !strings.Contains(html, "#359FD8") {
		t.Fatalf("color not sanitized")
	}
}
