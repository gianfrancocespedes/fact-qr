package handler

import (
	"github.com/gofiber/fiber/v2"

	"github.com/gianfrancocespedes/fact-qr/qr-api/internal/auth"
)

// defaultSubject identifica al portador cuando la petición no especifica uno.
const defaultSubject = "anonymous"

// tokenRequest permite nombrar al portador del token. Es opcional.
type tokenRequest struct {
	Subject string `json:"subject"`
}

// tokenResponse es lo que recibe el cliente al solicitar un token.
type tokenResponse struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expiresAt"`
	TokenType string `json:"tokenType"`
}

// AuthHandler emite los tokens que protegen la API.
type AuthHandler struct{}

// NewAuthHandler construye el handler de autenticación.
func NewAuthHandler() *AuthHandler {
	return &AuthHandler{}
}

// PostToken resuelve POST /api/v1/auth/token.
func (h *AuthHandler) PostToken(c *fiber.Ctx) error {
	request := tokenRequest{Subject: defaultSubject}

	// Un cuerpo ausente o malformado no es un error: todos los campos son opcionales.
	if len(c.Body()) > 0 {
		_ = c.BodyParser(&request)
	}
	if request.Subject == "" {
		request.Subject = defaultSubject
	}

	token, expiresAt, err := auth.Issue(request.Subject)
	if err != nil {
		return err
	}

	return c.JSON(tokenResponse{
		Token:     token,
		ExpiresAt: expiresAt.UTC().Format("2006-01-02T15:04:05Z"),
		TokenType: "Bearer",
	})
}
