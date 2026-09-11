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

	mailHandler := handlers.NewMailHandler(mailSender)
	whatsappHandler := handlers.NewWhatsAppHandler(whatsappSender)

	protected := http.NewServeMux()
	protected.HandleFunc("/mail/bienvenida", mailHandler.Bienvenida)
	protected.HandleFunc("/mail/contacto", mailHandler.Contacto)
	protected.HandleFunc("/mail/contacto-upac", mailHandler.ContactoUPAC)
	protected.HandleFunc("/whatsapp/bienvenida", whatsappHandler.Bienvenida)
	protected.HandleFunc("/whatsapp/contacto", whatsappHandler.Contacto)
	protected.HandleFunc("/whatsapp/contacto-upac", whatsappHandler.ContactoUPAC)

	root := http.NewServeMux()
	root.HandleFunc("/health", handlers.Health)

	// ✅ Servir el archivo OpenAPI
	root.HandleFunc("/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		http.ServeFile(w, r, "openapi.yaml")
	})

	// ✅ Servir Swagger UI
	// ✅ Servir Swagger UI
	// ✅ Servir Swagger UI - Versión FUNCIONAL
	// ✅ Servir Swagger UI - Versión LOCAL que FUNCIONA
root.HandleFunc("/docs", func(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
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
	<script src="https://cdnjs.cloudflare.com/ajax/libs/swagger-ui/5.9.0/swagger-ui.min.js"></script>
	<script src="https://cdnjs.cloudflare.com/ajax/libs/swagger-ui/5.9.0/swagger-ui-bundle.min.js"></script>
	<script src="https://cdnjs.cloudflare.com/ajax/libs/swagger-ui/5.9.0/swagger-ui-standalone-preset.min.js"></script>
	<script>
		const ui = SwaggerUIBundle({
			url: "/openapi.yaml",
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
	log.Printf("📚 Documentación Swagger en: http://localhost:%s/docs", cfg.Port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}