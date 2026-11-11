package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all runtime configuration for the API and the embedded web app.
type Config struct {
	Env            string
	Addr           string
	DatabasePath   string
	UploadDir      string
	StaticDir      string
	PublicURL      string
	AllowedOrigins []string
	SessionTTL     time.Duration
	SecureCookies  bool
	UploadMaxMB    int64
	AppKey         string
	SeedAdminName  string
	SeedAdminEmail string
	SeedAdminPass  string
	SeedDemoData   bool
	Timezone       string

	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPassword string
	SMTPFrom     string
}

// Load reads environment variables (optionally from a .env file) and returns a
// validated configuration. Every value has a safe local-development default.
func Load() (Config, error) {
	// .env is optional; production deployments normally inject env vars.
	_ = godotenv.Load()
	_ = godotenv.Load("../.env")

	cfg := Config{
		Env:            env("APP_ENV", "development"),
		Addr:           env("ADDR", ":8080"),
		DatabasePath:   env("DATABASE_PATH", "./data/app.db"),
		UploadDir:      env("UPLOAD_DIR", "./data/uploads"),
		StaticDir:      env("STATIC_DIR", "../frontend/dist"),
		PublicURL:      strings.TrimRight(env("PUBLIC_URL", "http://localhost:8080"), "/"),
		AllowedOrigins: splitCSV(env("ALLOWED_ORIGINS", "http://localhost:5173,http://localhost:8080")),
		SessionTTL:     time.Duration(envInt("SESSION_TTL_HOURS", 168)) * time.Hour,
		SecureCookies:  envBool("SECURE_COOKIES", env("APP_ENV", "development") == "production"),
		UploadMaxMB:    int64(envInt("UPLOAD_MAX_MB", 20)),
		AppKey:         env("APP_KEY", "dev-only-change-me-in-production-0123456789"),
		SeedAdminName:  env("SEED_ADMIN_NAME", "Administrador"),
		SeedAdminEmail: env("SEED_ADMIN_EMAIL", "admin@example.com"),
		SeedAdminPass:  env("SEED_ADMIN_PASSWORD", "password"),
		SeedDemoData:   envBool("SEED_DEMO_DATA", true),
		Timezone:       env("APP_TIMEZONE", "America/Sao_Paulo"),
		SMTPHost:       env("SMTP_HOST", ""),
		SMTPPort:       envInt("SMTP_PORT", 587),
		SMTPUser:       env("SMTP_USER", ""),
		SMTPPassword:   env("SMTP_PASSWORD", ""),
		SMTPFrom:       env("SMTP_FROM", "no-reply@documentation.local"),
	}

	if cfg.SessionTTL <= 0 {
		cfg.SessionTTL = 7 * 24 * time.Hour
	}
	if cfg.UploadMaxMB <= 0 {
		cfg.UploadMaxMB = 20
	}
	if len(cfg.AllowedOrigins) == 0 {
		cfg.AllowedOrigins = []string{"http://localhost:5173", "http://localhost:8080"}
	}

	if cfg.SeedAdminPass == "" {
		return cfg, fmt.Errorf("SEED_ADMIN_PASSWORD must not be empty")
	}
	return cfg, nil
}

func env(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value := env(key, "")
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		log.Printf("config: invalid integer for %s=%q, using %d", key, value, fallback)
		return fallback
	}
	return parsed
}

func envBool(key string, fallback bool) bool {
	value := strings.ToLower(env(key, ""))
	if value == "" {
		return fallback
	}
	return value == "1" || value == "true" || value == "yes" || value == "on"
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
