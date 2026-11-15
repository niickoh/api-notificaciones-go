package models

import "time"

// NotificationRequest representa el cuerpo compartido por todos los endpoints.
type NotificationRequest struct {
	EmailFrom string `json:"email_from"`
	EmailTo   string `json:"email_to"`
	Mensaje   string `json:"mensaje"`
	Telefono  string `json:"telefono"`
	Direccion string `json:"direccion"`
	Nombres   string `json:"nombres"`
	Apellidos string `json:"apellidos"`
	Comuna    string `json:"comuna"`
}

// Recipient es un destinatario de las plantillas SportSphere.
type Recipient struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

// ContactoRequest es el cuerpo de POST /mail/contacto (formulario de la landing).
type ContactoRequest struct {
	Name        string `json:"name"`
	Email       string `json:"email"`
	Message     string `json:"message"`
	Club        string `json:"club"`
	Telefono    string `json:"telefono"`
	Deportistas string `json:"deportistas"`
}

// AvisoRequest es el cuerpo de POST /mail/aviso.
// Acepta destinatarios (lista) o email + name sueltos.
type AvisoRequest struct {
	Destinatarios []Recipient `json:"destinatarios"`
	Email         string      `json:"email"`
	Name          string      `json:"name"`
	Titulo        string      `json:"titulo"`
	Mensaje       string      `json:"mensaje"`
	Prioridad     string      `json:"prioridad"`
	Autor         string      `json:"autor"`
	Equipo        string      `json:"equipo"`
	URL           string      `json:"url"`
}

// EventoRequest es el cuerpo de POST /mail/evento.
// Acepta destinatarios (lista) o email + name sueltos.
type EventoRequest struct {
	Destinatarios []Recipient `json:"destinatarios"`
	Email         string      `json:"email"`
	Name          string      `json:"name"`
	Titulo        string      `json:"titulo"`
	Tipo          string      `json:"tipo"`
	Estado        string      `json:"estado"`
	FechaInicio   string      `json:"fecha_inicio"`
	FechaFin      string      `json:"fecha_fin"`
	Ubicacion     string      `json:"ubicacion"`
	Equipo        string      `json:"equipo"`
	Descripcion   string      `json:"descripcion"`
	URL           string      `json:"url"`
}

// UsuarioCreadoRequest es el cuerpo de POST /mail/bienvenidaNew.
type UsuarioCreadoRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Username string `json:"username"`
	Password string `json:"password"`
	Rol      string `json:"rol"`
	Club     string `json:"club"`
	URL      string `json:"url"`
}

// AsignacionEquipoRequest es el cuerpo de POST /mail/asignacionEquipo.
type AsignacionEquipoRequest struct {
	Email      string `json:"email"`
	Name       string `json:"name"`
	Equipo     string `json:"equipo"`
	Rol        string `json:"rol"`
	Entrenador string `json:"entrenador"`
	Club       string `json:"club"`
	Color      string `json:"color"`
	URL        string `json:"url"`
}

// MailMessage es un correo ya renderizado listo para Mailjet.
type MailMessage struct {
	To      Recipient
	ReplyTo *Recipient
	Subject string
	Text    string
	HTML    string
}

// NotificationResult contiene el identificador devuelto por una integración externa.
type NotificationResult struct {
	MessageID string
	Timestamp time.Time
}

// SuccessData define el bloque data para respuestas exitosas.
type SuccessData struct {
	MessageID string `json:"message_id"`
	Timestamp string `json:"timestamp"`
}

// APIResponse define la respuesta estándar de la API.
type APIResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

// HealthResponse define la respuesta del healthcheck.
type HealthResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}
