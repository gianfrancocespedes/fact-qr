package main

import (
	"log"
	"os"
	"time"

	"github.com/gianfrancocespedes/fact-qr/qr-api/internal/app"
	"github.com/gianfrancocespedes/fact-qr/qr-api/internal/auth"
	"github.com/gianfrancocespedes/fact-qr/qr-api/internal/client"
)

const (
	defaultPort         = "8080"
	defaultStatsBaseURL = "http://localhost:3000"
)

func main() {
	// Valida la configuración de autenticación antes de iniciar el servidor.
	if err := auth.RequireSecret(); err != nil {
		log.Fatalf("[qr-api] configuración inválida: %v", err)
	}

	statsClient := client.New(
		envOrDefault("STATS_API_URL", defaultStatsBaseURL),
		parseTimeout(os.Getenv("STATS_API_TIMEOUT")),
	)

	server := app.New(app.Config{Stats: statsClient})

	port := envOrDefault("QR_API_PORT", defaultPort)
	log.Printf("[qr-api] escuchando en el puerto %s", port)

	// Quien decide el dual-stack es Network en internal/app; [::] lo hace explícito aquí,
	// donde se abre el puerto, porque de otro modo nada en este archivo delataría que el
	// modo de red es una elección deliberada y no el defecto de Fiber.
	if err := server.Listen("[::]:" + port); err != nil {
		log.Fatalf("[qr-api] no se pudo iniciar el servidor: %v", err)
	}
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// parseTimeout interpreta la duración configurada y recurre al valor por defecto si el
// formato es inválido: un timeout mal escrito no debe impedir que el servicio arranque.
func parseTimeout(raw string) time.Duration {
	if raw == "" {
		return client.DefaultTimeout
	}

	parsed, err := time.ParseDuration(raw)
	if err != nil || parsed <= 0 {
		log.Printf("[qr-api] STATS_API_TIMEOUT inválido (%q), se usa %s", raw, client.DefaultTimeout)
		return client.DefaultTimeout
	}
	return parsed
}
