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
