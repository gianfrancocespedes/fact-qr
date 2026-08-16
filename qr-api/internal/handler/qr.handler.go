// Package handler traduce entre HTTP y el dominio. No contiene lógica de negocio: lee la
// petición, delega en el servicio y serializa el resultado.
package handler

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/gianfrancocespedes/fact-qr/qr-api/internal/apperror"
	"github.com/gianfrancocespedes/fact-qr/qr-api/internal/matrix"
	"github.com/gianfrancocespedes/fact-qr/qr-api/internal/service"
)

// qrRequest es el cuerpo aceptado por POST /api/v1/qr.
//
// Rotations y Direction son opcionales: omitirlos equivale a factorizar la matriz tal
// como llegó, que es el caso principal.
type qrRequest struct {
	Matrix    [][]float64 `json:"matrix"`
	Rotations int         `json:"rotations"`
	Direction string      `json:"direction"`
}

// QRHandler atiende las peticiones de factorización.
type QRHandler struct {
	service *service.QRService
}

// NewQRHandler construye el handler con su servicio.
func NewQRHandler(qrService *service.QRService) *QRHandler {
	return &QRHandler{service: qrService}
}

// Post resuelve POST /api/v1/qr.
func (h *QRHandler) Post(c *fiber.Ctx) error {
	var request qrRequest
	if err := c.BodyParser(&request); err != nil {
		return apperror.New(apperror.CodeInvalidPayload, apperror.Details{
			"reason": "malformed_json",
		})
	}

	// La validación se repite aquí aunque el frontend use un slider de 0 a 3: la API es
	// pública y no puede confiar en su cliente.
	if err := matrix.ValidateRotations(request.Rotations); err != nil {
		return err
	}

	direction, err := matrix.ParseDirection(request.Direction)
	if err != nil {
		return err
	}

	input, err := matrix.New(request.Matrix)
	if err != nil {
		return err
	}

	result, err := h.service.Execute(c.Context(), service.Request{
		Matrix:    input,
		Rotations: request.Rotations,
		Direction: direction,
		Token:     extractBearerToken(c),
	})
	if err != nil {
		return err
	}

	return c.JSON(result)
}

// extractBearerToken obtiene el token del encabezado Authorization para propagarlo a la
// stats-api. Devuelve cadena vacía si no hay token: la autenticación se aplica en su
// propio middleware, no aquí.
func extractBearerToken(c *fiber.Ctx) string {
	const prefix = "Bearer "

	header := c.Get(fiber.HeaderAuthorization)
	if len(header) > len(prefix) && strings.EqualFold(header[:len(prefix)], prefix) {
		return header[len(prefix):]
	}
	return ""
}
