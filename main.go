package main

import (
	"log"
	"net/http"
	"time"

	"github.com/niickoh/api-notificaciones-go/config"
	"github.com/niickoh/api-notificaciones-go/handlers"
	"github.com/niickoh/api-notificaciones-go/services"
)

func newHandler(cfg config.Config, mailSender services.MailSender, whatsappSender services.WhatsAppSender) http.Handler {
	jwtService := services.NewJWTService(cfg.JWTSecret)
	ipLimiter := services.NewFixedWindowLimiter(cfg.RateLimitIP, time.Hour)
	apiKeyLimiter := services.NewFixedWindowLimiter(cfg.RateLimitAPIKey, time.Hour)
	contactoLimiter := services.NewFixedWindowLimiter(cfg.RateLimitContacto, time.Hour)

	mailHandler := handlers.NewMailHandler(mailSender)
	sportsphere := handlers.NewSportSphereMailHandler(mailSender, cfg)
	whatsappHandler := handlers.NewWhatsAppHandler(whatsappSender)

	// withSlash registra la ruta con y sin "/" final (el front llama p. ej. /mail/contacto/).
	withSlash := func(mux *http.ServeMux, path string, h http.Handler) {
		mux.Handle(path, h)
		mux.Handle(path+"/", h)
	}

	protected := http.NewServeMux()
	protected.HandleFunc("/mail/bienvenida", mailHandler.Bienvenida)
	protected.HandleFunc("/mail/contacto-upac", mailHandler.ContactoUPAC)
	// Plantillas SportSphere: un endpoint por plantilla (/mail/contacto es público, ver más abajo)
	withSlash(protected, "/mail/aviso", http.HandlerFunc(sportsphere.Aviso))
	withSlash(protected, "/mail/evento", http.HandlerFunc(sportsphere.Evento))
	withSlash(protected, "/mail/bienvenidaNew", http.HandlerFunc(sportsphere.UsuarioCreado))
	withSlash(protected, "/mail/asignacionEquipo", http.HandlerFunc(sportsphere.AsignacionEquipo))
	protected.HandleFunc("/whatsapp/bienvenida", whatsappHandler.Bienvenida)
	protected.HandleFunc("/whatsapp/contacto", whatsappHandler.Contacto)
	protected.HandleFunc("/whatsapp/contacto-upac", whatsappHandler.ContactoUPAC)

	root := http.NewServeMux()
	root.HandleFunc("/health", handlers.Health)

	// ✅ Servir el archivo OpenAPI
	root.HandleFunc("/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		w.Header().Set("Cache-Control", "no-cache") // evita que el navegador siga usando un spec viejo
		http.ServeFile(w, r, "openapi.yaml")
	})

	// ✅ Servir Swagger UI
root.HandleFunc("/docs", func(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	html := `<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<title>API de Notificaciones</title>
	<link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/swagger-ui/5.9.0/swagger-ui.min.css">
	<style>
		html { box-sizing: border-box; overflow: -moz-scrollbars-vertical; overflow-y: scroll; }
		*, *:before, *:after { box-sizing: inherit; }
		body { margin:0; background: #fafafa; }
	</style>
</head>
<body>
	<div id="swagger-ui"></div>
	<script src="https://cdnjs.cloudflare.com/ajax/libs/swagger-ui/5.9.0/swagger-ui-bundle.min.js"></script>
	<script src="https://cdnjs.cloudflare.com/ajax/libs/swagger-ui/5.9.0/swagger-ui-standalone-preset.min.js"></script>
	<script>
		const ui = SwaggerUIBundle({
			url: "openapi.yaml", // relativo a /docs, así respeta el dominio y un posible prefijo de nginx
			dom_id: '#swagger-ui',
			presets: [
				SwaggerUIBundle.presets.apis,
				SwaggerUIBundle.SwaggerUIStandalonePreset
			],
			layout: "BaseLayout"
		})
	</script>
</body>
</html>`
	w.Write([]byte(html))
})

	protectedChain := handlers.Chain(
		protected,
		handlers.RateLimitMiddleware(ipLimiter, apiKeyLimiter),
		handlers.AuthMiddleware(jwtService),
	)
	root.Handle("/mail/", protectedChain)
	// Formulario de contacto de la landing: lo usan visitantes sin sesión, así que no pide JWT.
	// El destinatario es fijo (CONTACT_EMAIL_TO) y se limita por IP para frenar el spam.
	withSlash(root, "/mail/contacto", handlers.Chain(
		http.HandlerFunc(sportsphere.Contacto),
		handlers.RateLimitMiddleware(contactoLimiter, apiKeyLimiter),
	))
	root.Handle("/whatsapp/", protectedChain)

	return handlers.Chain(
		root,
		handlers.CORSMiddleware(cfg.CORSOrigin),
		handlers.RecoverMiddleware,
	)
}

func main() {
	cfg := config.Load()
	mailSender := services.NewMailjetService(cfg)
	whatsappSender := services.NewWhatsAppService(cfg)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           newHandler(cfg, mailSender, whatsappSender),
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Printf("api de notificaciones escuchando en :%s", cfg.Port)
	log.Printf("/mail/contacto envía a: %q (CONTACT_EMAIL_TO)", cfg.ContactEmailTo)
	log.Printf("📚 Documentación Swagger en: http://localhost:%s/docs", cfg.Port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}