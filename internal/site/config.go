package site

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"
)

// Config holds runtime settings for the site and CMS.
type Config struct {
	ContentDir    string
	PublicBaseURL string
	AdminUser     string
	AdminPassword string
	SessionSecret string
}

// ConfigFromEnv loads configuration from environment variables.
func ConfigFromEnv() Config {
	cfg := Config{
		ContentDir:    envOr("CONTENT_DIR", "content"),
		PublicBaseURL: strings.TrimSpace(os.Getenv("PUBLIC_BASE_URL")),
		AdminUser:     strings.TrimSpace(os.Getenv("ADMIN_USER")),
		AdminPassword: os.Getenv("ADMIN_PASSWORD"),
		SessionSecret: strings.TrimSpace(os.Getenv("ADMIN_SESSION_SECRET")),
	}
	if cfg.SessionSecret == "" && cfg.AdminPassword != "" {
		sum := sha256.Sum256([]byte("neonaicloud-session:" + cfg.AdminPassword))
		cfg.SessionSecret = hex.EncodeToString(sum[:])
	}
	return cfg
}

func (c Config) AdminEnabled() bool {
	return c.AdminUser != "" && c.AdminPassword != "" && c.SessionSecret != ""
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
