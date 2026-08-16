package matrix

import "github.com/gianfrancocespedes/fact-qr/qr-api/internal/apperror"

// Direction indica el sentido de giro de la rotación.
type Direction string

const (
	Clockwise        Direction = "clockwise"
	CounterClockwise Direction = "counterclockwise"
)

// MaxRotations es el mayor número de rotaciones aceptado.
const MaxRotations = 3

// ParseDirection valida el sentido recibido. La cadena vacía adopta el valor por defecto (horario)
func ParseDirection(value string) (Direction, error) {
	switch Direction(value) {
	case "":
		return Clockwise, nil
	case Clockwise:
		return Clockwise, nil
	case CounterClockwise:
		return CounterClockwise, nil
	default:
		return "", apperror.New(apperror.CodeInvalidDirection, apperror.Details{
			"received": value,
			"allowed":  []string{string(Clockwise), string(CounterClockwise)},
		})
	}
}

// ValidateRotations comprueba que el número de rotaciones esté en el rango permitido.
func ValidateRotations(times int) error {
	if times < 0 || times > MaxRotations {
		return apperror.New(apperror.CodeInvalidRotations, apperror.Details{
			"received": times,
			"minimum":  0,
			"maximum":  MaxRotations,
		})
	}
	return nil
}

// Rotate gira la matriz 90 grados el número de veces indicado.
func (m Matrix) Rotate(times int, direction Direction) Matrix {
	if err := ValidateRotations(times); err != nil {
		return m.Clone()
	}

	result := m.Clone()
	for i := 0; i < times; i++ {
		if direction == CounterClockwise {
			result = result.rotateCounterClockwise()
			continue
		}
		result = result.rotateClockwise()
	}
	return result
}

// rotateClockwise gira 90° en sentido horario
func (m Matrix) rotateClockwise() Matrix {
	rows, columns := m.Rows(), m.Columns()
	result := zeros(columns, rows)

	for i := 0; i < rows; i++ {
		for j := 0; j < columns; j++ {
			result[j][rows-1-i] = m[i][j]
		}
	}
	return result
}

// rotateCounterClockwise gira 90° en sentido antihorario
func (m Matrix) rotateCounterClockwise() Matrix {
	rows, columns := m.Rows(), m.Columns()
	result := zeros(columns, rows)

	for i := 0; i < rows; i++ {
		for j := 0; j < columns; j++ {
			result[columns-1-j][i] = m[i][j]
		}
	}
	return result
}
