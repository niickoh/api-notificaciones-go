package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config centraliza la configuración de la aplicación.
type Config struct {
	Port              string
	JWTSecret         string
	MailjetAPIKey     string
	MailjetAPISecret  string
	MailjetFromEmail  string
	MailjetFromName   string
	MetaWhatsAppToken string
	MetaWhatsAppPhone string
	RateLimitIP       int
	RateLimitAPIKey   int
	CORSOrigin        string
	TemplatesDir      string
}

// Load obtiene la configuración desde variables de entorno y aplica defaults.
func Load() Config {
	_ = godotenv.Load()

	return Config{
		Port:              getEnv("PORT", "4000"),
		JWTSecret:         getEnv("JWT_SECRET", "bascodelab-0317"),
		MailjetAPIKey:     getEnv("MAILJET_API_KEY", ""),
		MailjetAPISecret:  getEnv("MAILJET_API_SECRET", ""),
		MailjetFromEmail:  getEnv("MAILJET_FROM_EMAIL", "noreply@tudominio.com"),
		MailjetFromName:   getEnv("MAILJET_FROM_NAME", "Tu Nombre"),
		MetaWhatsAppToken: getEnv("META_WHATSAPP_TOKEN", ""),
		MetaWhatsAppPhone: getEnv("META_WHATSAPP_PHONE_ID", ""),
		RateLimitIP:       getEnvInt("RATE_LIMIT_IP", 100),
		RateLimitAPIKey:   getEnvInt("RATE_LIMIT_API_KEY", 1000),
		CORSOrigin:        getEnv("CORS_ORIGIN", "*"),
		TemplatesDir:      getEnv("TEMPLATES_DIR", "templates"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}
