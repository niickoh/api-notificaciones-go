package services

import (
	"bytes"
	"fmt"
	"html/template"
	"path/filepath"
	"strings"
	"time"
)

// templateKeys son las variables que usan _base.html y las plantillas SportSphere;
// se inicializan en "" para que una clave ausente nunca se pinte como "<no value>".
var templateKeys = []string{
	"ETIQUETA", "ASUNTO", "PREHEADER", "URL_PORTAL", "LOGO", "URL", "BOTON", "NOTA", "EMPRESA",
	"NOMBRE", "EMAIL", "MENSAJE", "TITULO", "PRIORIDAD", "PRIORIDAD_COLOR", "AUTOR", "EQUIPO",
	"INTRO", "TIPO", "ESTADO_COLOR", "FECHA_INICIO", "FECHA_FIN", "UBICACION", "DESCRIPCION",
	"USUARIO", "PASSWORD", "ROL", "CLUB", "ENTRENADOR", "COLOR",
}

// RenderMail renderiza templates/<name>.html junto con templates/_base.html.
func RenderMail(dir, name string, data map[string]any) (string, error) {
	tmpl, err := template.New(name+".html").Funcs(template.FuncMap{"fecha": FechaES}).
		ParseFiles(filepath.Join(dir, "_base.html"), filepath.Join(dir, name+".html"))
	if err != nil {
		return "", fmt.Errorf("plantilla %s: %w", name, err)
	}

	full := make(map[string]any, len(templateKeys)+len(data))
	for _, k := range templateKeys {
		full[k] = ""
	}
	for k, v := range data {
		full[k] = v
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, full); err != nil {
		return "", fmt.Errorf("plantilla %s: %w", name, err)
	}
	return buf.String(), nil
}

var (
	diasES  = []string{"domingo", "lunes", "martes", "miércoles", "jueves", "viernes", "sábado"}
	mesesES = []string{"enero", "febrero", "marzo", "abril", "mayo", "junio", "julio", "agosto", "septiembre", "octubre", "noviembre", "diciembre"}
)

// FechaES formatea una fecha ISO como "martes 17 de noviembre de 2026, 18:30" (hora de Chile si trae zona).
// Si no se puede leer, la devuelve tal cual.
func FechaES(s string) string {
	s = strings.TrimSpace(s)
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02T15:04", "2006-01-02 15:04:05", "2006-01-02 15:04", "2006-01-02"} {
		t, err := time.Parse(layout, s)
		if err != nil {
			continue
		}
		if layout == time.RFC3339 {
			if loc, err := time.LoadLocation("America/Santiago"); err == nil {
				t = t.In(loc)
			}
		}
		out := fmt.Sprintf("%s %d de %s de %d", diasES[t.Weekday()], t.Day(), mesesES[t.Month()-1], t.Year())
		if layout != "2006-01-02" {
			out += t.Format(", 15:04")
		}
		return out
	}
	return s
}
