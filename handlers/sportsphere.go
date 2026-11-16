package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/niickoh/api-notificaciones-go/config"
	"github.com/niickoh/api-notificaciones-go/models"
	"github.com/niickoh/api-notificaciones-go/services"
)

const maxFieldLength = 5000

var hexColorRe = regexp.MustCompile(`^#[0-9a-fA-F]{3,8}$`)

// SportSphereMailHandler expone un endpoint por cada plantilla SportSphere
// (contacto, aviso, evento, usuario creado y asignación de equipo).
type SportSphereMailHandler struct {
	sender         services.MailSender
	templatesDir   string
	portalURL      string
	contactEmailTo string
}

// NewSportSphereMailHandler crea el handler de las plantillas SportSphere.
func NewSportSphereMailHandler(sender services.MailSender, cfg config.Config) *SportSphereMailHandler {
	return &SportSphereMailHandler{
		sender:         sender,
		templatesDir:   cfg.TemplatesDir,
		portalURL:      cfg.PortalURL,
		contactEmailTo: cfg.ContactEmailTo,
	}
}

// Contacto procesa POST /mail/contacto: el mensaje llega a CONTACT_EMAIL_TO con el correo del visitante en el cuerpo.
func (h *SportSphereMailHandler) Contacto(w http.ResponseWriter, r *http.Request) {
	var p models.ContactoRequest
	if !decodePost(w, r, &p) {
		return
	}
	if err := checkFields(map[string]string{"name": p.Name, "email": p.Email, "message": p.Message},
		p.Club, p.Telefono, p.Deportistas); err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if !validEmail(p.Email) {
		WriteError(w, http.StatusBadRequest, "El email no es válido")
		return
	}
	if h.contactEmailTo == "" {
		WriteError(w, http.StatusServiceUnavailable, "El envío de correos no está configurado")
		return
	}

	var extra []string
	for _, f := range [][2]string{{"Club", p.Club}, {"Teléfono", p.Telefono}, {"N° de deportistas", p.Deportistas}} {
		if v := strings.TrimSpace(f[1]); v != "" {
			extra = append(extra, f[0]+": "+v)
		}
	}
	name := strings.TrimSpace(p.Name)

	data := h.base()
	data["ETIQUETA"] = "Formulario de contacto"
	data["ASUNTO"] = "Nuevo mensaje de " + name
	data["PREHEADER"] = p.Message
	data["NOMBRE"] = name
	data["EMAIL"] = p.Email
	data["MENSAJE"] = strings.TrimSpace(strings.Join(append(extra, "", p.Message), "\n"))

	html, err := services.RenderMail(h.templatesDir, "contacto", data)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	text := append([]string{"Nombre: " + name, "Correo: " + p.Email}, extra...)
	text = append(text, "", "Mensaje:", p.Message)
	// Sin Reply-To: uno gratuito (Gmail, Hotmail...) distinto del From suma ~8 puntos de spam en Rspamd.
	// El correo del visitante ya va en el cuerpo.
	h.send(w, []models.MailMessage{{
		To:      models.Recipient{Email: h.contactEmailTo, Name: "Administrador"},
		Subject: "Nuevo mensaje de contacto de " + name,
		Text:    strings.Join(text, "\n"),
		HTML:    html,
	}})
}

// Aviso procesa POST /mail/aviso.
func (h *SportSphereMailHandler) Aviso(w http.ResponseWriter, r *http.Request) {
	var p models.AvisoRequest
	if !decodePost(w, r, &p) {
		return
	}
	if err := checkFields(map[string]string{"titulo": p.Titulo, "mensaje": p.Mensaje},
		p.Prioridad, p.Autor, p.Equipo, p.URL); err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	prioridad := strings.ToLower(strings.TrimSpace(p.Prioridad))
	color, ok := map[string]string{"normal": "#0075BE", "importante": "#f59e0b", "urgente": "#dc2626"}[prioridad]
	if !ok {
		prioridad, color = "normal", "#0075BE"
	}
	titulo := strings.TrimSpace(p.Titulo)
	asunto := titulo
	if prioridad != "normal" {
		asunto = "[" + strings.ToUpper(prioridad) + "] " + titulo
	}

	data := h.base()
	data["ETIQUETA"] = "Nuevo aviso"
	data["ASUNTO"] = asunto
	data["PREHEADER"] = p.Mensaje
	data["TITULO"] = titulo
	data["MENSAJE"] = p.Mensaje
	data["PRIORIDAD"] = prioridad
	data["PRIORIDAD_COLOR"] = color
	data["AUTOR"] = strings.TrimSpace(p.Autor)
	data["EQUIPO"] = strings.TrimSpace(p.Equipo)
	data["BOTON"] = "Ver aviso"
	data["URL"] = strings.TrimSpace(p.URL)
	h.sendToAll(w, "aviso", data, recipientsOf(p.Destinatarios, p.Email, p.Name))
}

// Evento procesa POST /mail/evento (entrenamientos, competencias, reuniones...).
func (h *SportSphereMailHandler) Evento(w http.ResponseWriter, r *http.Request) {
	var p models.EventoRequest
	if !decodePost(w, r, &p) {
		return
	}
	if err := checkFields(map[string]string{"titulo": p.Titulo, "fecha_inicio": p.FechaInicio},
		p.Tipo, p.Estado, p.FechaFin, p.Ubicacion, p.Equipo, p.Descripcion, p.URL); err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	var etiqueta, intro, color string
	estado := strings.ToLower(strings.TrimSpace(p.Estado))
	switch estado {
	case "modificado":
		etiqueta, intro, color = "Evento modificado", "Se actualizó un evento de tu calendario:", "#f59e0b"
	case "cancelado":
		etiqueta, intro, color = "Evento cancelado", "Se canceló el siguiente evento:", "#dc2626"
	default:
		etiqueta, intro, color = "Nuevo evento en tu calendario", "Se agendó un nuevo evento en tu calendario:", "#0075BE"
	}
	titulo := strings.TrimSpace(p.Titulo)

	data := h.base()
	data["ETIQUETA"] = etiqueta
	data["ASUNTO"] = titulo + " · " + services.FechaES(p.FechaInicio)
	data["PREHEADER"] = etiqueta + ": " + titulo
	data["INTRO"] = intro
	data["TITULO"] = titulo
	data["TIPO"] = orDefault(p.Tipo, "ENTRENAMIENTO")
	data["ESTADO_COLOR"] = color
	data["CANCELADO"] = estado == "cancelado"
	data["FECHA_INICIO"] = strings.TrimSpace(p.FechaInicio)
	data["FECHA_FIN"] = strings.TrimSpace(p.FechaFin)
	data["UBICACION"] = strings.TrimSpace(p.Ubicacion)
	data["EQUIPO"] = strings.TrimSpace(p.Equipo)
	data["DESCRIPCION"] = p.Descripcion
	data["BOTON"] = "Ver calendario"
	data["URL"] = strings.TrimSpace(p.URL)
	h.sendToAll(w, "evento", data, recipientsOf(p.Destinatarios, p.Email, p.Name))
}

// UsuarioCreado procesa POST /mail/bienvenidaNew: datos de acceso de un usuario nuevo.
func (h *SportSphereMailHandler) UsuarioCreado(w http.ResponseWriter, r *http.Request) {
	var p models.UsuarioCreadoRequest
	if !decodePost(w, r, &p) {
		return
	}
	if err := checkFields(map[string]string{"email": p.Email, "username": p.Username},
		p.Name, p.Password, p.Rol, p.Club, p.URL); err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	data := h.base()
	data["ETIQUETA"] = "Cuenta creada"
	data["ASUNTO"] = "Bienvenido/a a SportSphere Pro"
	data["PREHEADER"] = "Tus datos de acceso a SportSphere Pro"
	data["USUARIO"] = strings.TrimSpace(p.Username)
	data["PASSWORD"] = p.Password
	data["ROL"] = strings.TrimSpace(p.Rol)
	data["CLUB"] = strings.TrimSpace(p.Club)
	data["BOTON"] = "Iniciar sesión"
	data["URL"] = orDefault(p.URL, h.portalURL)
	data["NOTA"] = "Si no esperabas este correo, puedes ignorarlo."
	h.sendToAll(w, "usuario-creado", data, recipientsOf(nil, p.Email, p.Name))
}

// AsignacionEquipo procesa POST /mail/asignacionEquipo.
func (h *SportSphereMailHandler) AsignacionEquipo(w http.ResponseWriter, r *http.Request) {
	var p models.AsignacionEquipoRequest
	if !decodePost(w, r, &p) {
		return
	}
	if err := checkFields(map[string]string{"email": p.Email, "equipo": p.Equipo},
		p.Name, p.Rol, p.Entrenador, p.Club, p.Color, p.URL); err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	color := strings.TrimSpace(p.Color)
	if !hexColorRe.MatchString(color) { // va dentro de style="": solo colores hex
		color = "#359FD8"
	}
	equipo := strings.TrimSpace(p.Equipo)

	data := h.base()
	data["ETIQUETA"] = "Asignación de equipo"
	data["ASUNTO"] = "Ahora eres parte de " + equipo
	data["PREHEADER"] = "Fuiste asignado/a al equipo " + equipo
	data["EQUIPO"] = equipo
	data["ROL"] = strings.TrimSpace(p.Rol)
	data["ENTRENADOR"] = strings.TrimSpace(p.Entrenador)
	data["CLUB"] = strings.TrimSpace(p.Club)
	data["COLOR"] = color
	data["BOTON"] = "Ver mi equipo"
	data["URL"] = orDefault(p.URL, h.portalURL)
	h.sendToAll(w, "asignacion-equipo", data, recipientsOf(nil, p.Email, p.Name))
}

func (h *SportSphereMailHandler) base() map[string]any {
	return map[string]any{
		"EMPRESA":    "SportSphere Pro",
		"URL_PORTAL": h.portalURL,
		"LOGO":       strings.TrimRight(h.portalURL, "/") + "/assets/images/sphere-mail-logo.png",
	}
}

// sendToAll renderiza la plantilla una vez por destinatario (saludo personalizado) y envía.
func (h *SportSphereMailHandler) sendToAll(w http.ResponseWriter, template string, data map[string]any, to []models.Recipient) {
	if len(to) == 0 {
		WriteError(w, http.StatusBadRequest, "destinatarios no puede estar vacío")
		return
	}
	asunto, preheader := fmt.Sprint(data["ASUNTO"]), fmt.Sprint(data["PREHEADER"])
	messages := make([]models.MailMessage, 0, len(to))
	for _, r := range to {
		if !validEmail(r.Email) {
			WriteError(w, http.StatusBadRequest, "Email no válido: "+r.Email)
			return
		}
		if utf8.RuneCountInString(r.Name) > maxFieldLength {
			WriteError(w, http.StatusBadRequest, "El mensaje es demasiado largo")
			return
		}
		data["NOMBRE"] = r.Name
		html, err := services.RenderMail(h.templatesDir, template, data)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		messages = append(messages, models.MailMessage{
			To:      r,
			Subject: asunto,
			Text:    asunto + "\n\n" + preheader + "\n\n" + h.portalURL,
			HTML:    html,
		})
	}
	h.send(w, messages)
}

func (h *SportSphereMailHandler) send(w http.ResponseWriter, messages []models.MailMessage) {
	result, err := h.sender.SendMessages(messages)
	if err != nil {
		WriteError(w, http.StatusBadGateway, fmt.Sprintf("Fallo en integración Mailjet: %v", err))
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

func decodePost(w http.ResponseWriter, r *http.Request, dst any) bool {
	if r.Method != http.MethodPost {
		WriteError(w, http.StatusMethodNotAllowed, "Método no permitido")
		return false
	}
	if err := decodeRequest(r, dst); err != nil {
		WriteError(w, http.StatusBadRequest, "JSON inválido")
		return false
	}
	return true
}

// checkFields valida obligatorios y el largo máximo de todos los campos de texto.
func checkFields(required map[string]string, optional ...string) error {
	var missing []string
	for name, v := range required {
		if strings.TrimSpace(v) == "" {
			missing = append(missing, name)
		}
		optional = append(optional, v)
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("%s son obligatorios", strings.Join(missing, ", "))
	}
	for _, v := range optional {
		if utf8.RuneCountInString(v) > maxFieldLength {
			return errors.New("El mensaje es demasiado largo")
		}
	}
	return nil
}

// recipientsOf usa destinatarios si viene; si no, email + name sueltos.
func recipientsOf(list []models.Recipient, email, name string) []models.Recipient {
	if list == nil {
		list = []models.Recipient{{Email: email, Name: name}}
	}
	out := make([]models.Recipient, 0, len(list))
	for _, r := range list {
		r.Email = strings.TrimSpace(r.Email)
		r.Name = orDefault(r.Name, r.Email)
		out = append(out, r)
	}
	return out
}

func validEmail(email string) bool {
	addr, err := mail.ParseAddress(email)
	return err == nil && addr.Address == email
}

func orDefault(v, def string) string {
	if v = strings.TrimSpace(v); v != "" {
		return v
	}
	return def
}
