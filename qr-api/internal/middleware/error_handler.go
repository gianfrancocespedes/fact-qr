package middleware

import (
	"errors"
	"log"
	"net/http"

	"github.com/gofiber/fiber/v2"

	"github.com/gianfrancocespedes/fact-qr/qr-api/internal/apperror"
)

// errorBody es la forma que toda respuesta de error devuelve al cliente.
type errorBody struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    apperror.Code    `json:"code"`
	Details apperror.Details `json:"details,omitempty"`
}

// statusByCode asigna el código HTTP a cada error de dominio.
var statusByCode = map[apperror.Code]int{
	apperror.CodeInvalidNumber:           http.StatusBadRequest,
	apperror.CodeEmptyMatrix:             http.StatusBadRequest,
	apperror.CodeInconsistentRowLength:   http.StatusBadRequest,
	apperror.CodeMatrixTooLarge:          http.StatusRequestEntityTooLarge,
	apperror.CodeInvalidRotations:        http.StatusBadRequest,
	apperror.CodeInvalidDirection:        http.StatusBadRequest,
	apperror.CodeInvalidPayload:          http.StatusBadRequest,
	apperror.CodeUnauthorized:            http.StatusUnauthorized,
	apperror.CodeTokenExpired:            http.StatusUnauthorized,
	apperror.CodeStatsServiceUnavailable: http.StatusServiceUnavailable,
	apperror.CodeInternal:                http.StatusInternalServerError,
}

// ErrorHandler es el punto único de serialización de errores de la API.
func ErrorHandler(c *fiber.Ctx, err error) error {
	var domainError *apperror.Error
	if errors.As(err, &domainError) {
		status, known := statusByCode[domainError.Code]
		if !known {
			status = http.StatusInternalServerError
		}
		return c.Status(status).JSON(errorBody{
			Error: errorDetail{Code: domainError.Code, Details: domainError.Details},
		})
	}

	// Fiber genera este error para rutas inexistentes y otros fallos de enrutamiento.
	var fiberError *fiber.Error
	if errors.As(err, &fiberError) {
		return c.Status(fiberError.Code).JSON(errorBody{
			Error: errorDetail{
				Code:    apperror.CodeInvalidPayload,
				Details: apperror.Details{"path": c.Path()},
			},
		})
	}

	// Cualquier otra cosa es un fallo no previsto.
	log.Printf("[qr-api] error no controlado: %v", err)
	return c.Status(http.StatusInternalServerError).JSON(errorBody{
		Error: errorDetail{Code: apperror.CodeInternal},
	})
}
