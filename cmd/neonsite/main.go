package main

import (
	"log"
	"net/http"
	"os"

	"neonaicloud/internal/site"
)

func main() {
	addr := envOr("HTTP_ADDR", ":8080")
	cfg := site.ConfigFromEnv()

	srv, err := site.New(cfg)
	if err != nil {
		log.Fatalf("site: %v", err)
	}

	log.Printf("Neon AI Cloud listening on %s", addr)
	log.Printf("content dir: %s", srv.ContentDir())
	if cfg.AdminEnabled() {
		log.Printf("admin CMS enabled for user %q", cfg.AdminUser)
	} else {
		log.Printf("admin CMS disabled (set ADMIN_USER and ADMIN_PASSWORD to enable)")
	}

	if err := http.ListenAndServe(addr, srv.Handler()); err != nil {
		log.Fatalf("listen: %v", err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
