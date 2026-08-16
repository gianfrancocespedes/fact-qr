// Package apperror define los errores de dominio de la API.
//
// El principio: el backend nunca devuelve texto para mostrar al usuario, solo un código
// estable y los datos necesarios para redactar el mensaje. La traducción al español vive
// en el frontend, donde está el idioma del usuario. Así, cambiar la redacción no rompe
// ningún cliente porque el contrato es el código, no el texto.
package apperror

import "fmt"

// Code identifica un tipo de error de forma estable. Es parte del contrato público de
// la API: estos valores los consume el frontend para elegir qué mensaje mostrar.
type Code string

const (
	CodeInvalidNumber           Code = "ERROR_INVALID_NUMBER"
	CodeEmptyMatrix             Code = "ERROR_EMPTY_MATRIX"
	CodeInconsistentRowLength   Code = "ERROR_INCONSISTENT_ROW_LENGTH"
	CodeMatrixTooLarge          Code = "ERROR_MATRIX_TOO_LARGE"
	CodeInvalidRotations        Code = "ERROR_INVALID_ROTATIONS"
	CodeInvalidDirection        Code = "ERROR_INVALID_DIRECTION"
	CodeInvalidPayload          Code = "ERROR_INVALID_PAYLOAD"
	CodeUnauthorized            Code = "ERROR_UNAUTHORIZED"
	CodeTokenExpired            Code = "ERROR_TOKEN_EXPIRED"
	CodeStatsServiceUnavailable Code = "ERROR_STATS_SERVICE_UNAVAILABLE"
	CodeInternal                Code = "ERROR_INTERNAL"
)

// Details lleva el contexto que el frontend necesita para construir un mensaje específico
// ("El valor de la fila 2, columna 3 no es válido") en lugar de uno genérico.
type Details map[string]any

// Error es un error de dominio con código estable. Implementa error, así que se propaga
// con las herramientas habituales de Go (errors.Is, errors.As, %w).
type Error struct {
	Code    Code
	Details Details
	// cause conserva el error subyacente para diagnóstico interno; nunca se expone al
	// cliente, porque puede contener detalles de infraestructura.
	cause error
}

func (e *Error) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %v", e.Code, e.cause)
	}
	return string(e.Code)
}

// Unwrap permite que errors.Is y errors.As atraviesen esta capa hasta el error original.
func (e *Error) Unwrap() error { return e.cause }

// New construye un error de dominio sin causa subyacente.
func New(code Code, details Details) *Error {
	return &Error{Code: code, Details: details}
}

// Wrap envuelve un error existente conservando su cadena para diagnóstico.
func Wrap(code Code, cause error, details Details) *Error {
	return &Error{Code: code, Details: details, cause: cause}
}

// InvalidNumber señala una celda que no es un número finito. Las coordenadas se reportan
// en base 1 porque el destinatario es una persona leyendo una grilla, no un índice de array.
func InvalidNumber(row, column int) *Error {
	return New(CodeInvalidNumber, Details{
		"row":    row + 1,
		"column": column + 1,
	})
}

// InconsistentRowLength señala una fila cuya longitud no coincide con la primera.
func InconsistentRowLength(row, expected, actual int) *Error {
	return New(CodeInconsistentRowLength, Details{
		"row":      row + 1,
		"expected": expected,
		"actual":   actual,
	})
}
