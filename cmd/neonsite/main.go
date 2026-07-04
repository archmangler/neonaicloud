package main

import (
	"log"
	"net/http"
	"os"

	"neonaicloud/internal/site"
)

func main() {
	addr := envOr("HTTP_ADDR", ":8080")

	srv, err := site.New()
	if err != nil {
		log.Fatalf("site: %v", err)
	}

	log.Printf("Neon AI Cloud listening on %s", addr)
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
