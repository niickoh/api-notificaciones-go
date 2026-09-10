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
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
