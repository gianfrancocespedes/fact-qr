package middleware

import (
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

// NewCORS configura CORS a partir de la lista de orígenes permitidos.
func NewCORS() fiber.Handler {
	configured := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS"))
	if configured == "" {
		configured = "*"
	}

	return cors.New(cors.Config{
		AllowOrigins: configured,
		AllowMethods: strings.Join([]string{fiber.MethodGet, fiber.MethodPost, fiber.MethodOptions}, ","),
		AllowHeaders: strings.Join([]string{fiber.HeaderContentType, fiber.HeaderAuthorization}, ","),
	})
}
