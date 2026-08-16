package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/gianfrancocespedes/fact-qr/qr-api/internal/apperror"
	"github.com/gianfrancocespedes/fact-qr/qr-api/internal/auth"
)

// bearerPrefix es el esquema de autenticación esperado en la cabecera Authorization.
const bearerPrefix = "Bearer "

// RequireAuth rechaza las peticiones sin un token válido.
func RequireAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		token, err := extractToken(c)
		if err != nil {
			return err
		}

		claims, err := auth.Verify(token)
		if err != nil {
			return err
		}

		// El sujeto queda disponible para los handlers posteriores; hoy no se usa, pero es
		// el punto natural de extensión si más adelante hiciera falta autorizar por usuario.
		c.Locals("subject", claims.Subject)
		return c.Next()
	}
}

// extractToken obtiene el token de la cabecera Authorization.
func extractToken(c *fiber.Ctx) (string, error) {
	header := c.Get(fiber.HeaderAuthorization)
	if header == "" {
		return "", apperror.New(apperror.CodeUnauthorized, apperror.Details{
			"reason": "missing_token",
		})
	}

	if len(header) <= len(bearerPrefix) || !strings.EqualFold(header[:len(bearerPrefix)], bearerPrefix) {
		return "", apperror.New(apperror.CodeUnauthorized, apperror.Details{
			"reason": "invalid_scheme",
		})
	}

	return header[len(bearerPrefix):], nil
}
