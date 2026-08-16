package matrix

import (
	"math"

	"github.com/gianfrancocespedes/fact-qr/qr-api/internal/apperror"
)

// MaxDimension acota el tamaño aceptado.
const MaxDimension = 100

// Matrix es una matriz rectangular de m×n
type Matrix [][]float64

// Rows devuelve el número de filas (m).
func (m Matrix) Rows() int { return len(m) }

// Columns devuelve el número de columnas (n).
func (m Matrix) Columns() int {
	if len(m) == 0 {
		return 0
	}
	return len(m[0])
}

// Clone devuelve una copia profunda de la matriz.
func (m Matrix) Clone() Matrix {
	clone := make(Matrix, len(m))
	for i, row := range m {
		clone[i] = make([]float64, len(row))
		copy(clone[i], row)
	}
	return clone
}

// New valida los datos recibidos y construye una Matrix.
//
// Se valida en el dominio y no solo en el handler porque estas reglas son invariantes
// del tipo: ninguna operación posterior tiene sentido sobre una matriz malformada.
func New(values [][]float64) (Matrix, error) {
	if len(values) == 0 || len(values[0]) == 0 {
		return nil, apperror.New(apperror.CodeEmptyMatrix, nil)
	}

	columns := len(values[0])
	if len(values) > MaxDimension || columns > MaxDimension {
		return nil, apperror.New(apperror.CodeMatrixTooLarge, apperror.Details{
			"rows":    len(values),
			"columns": columns,
			"maximum": MaxDimension,
		})
	}

	for i, row := range values {
		if len(row) != columns {
			return nil, apperror.InconsistentRowLength(i, columns, len(row))
		}
		for j, value := range row {
			// NaN e Inf sobreviven al parseo de JSON en algunos clientes y envenenarían
			// todo el cálculo en silencio: una sola celda infinita vuelve NaN la matriz entera.
			if math.IsNaN(value) || math.IsInf(value, 0) {
				return nil, apperror.InvalidNumber(i, j)
			}
		}
	}

	return Matrix(values).Clone(), nil
}

// identity construye la matriz identidad de tamaño size×size.
func identity(size int) Matrix {
	result := make(Matrix, size)
	for i := range result {
		result[i] = make([]float64, size)
		result[i][i] = 1
	}
	return result
}

// zeros construye una matriz de ceros de rows×columns.
func zeros(rows, columns int) Matrix {
	result := make(Matrix, rows)
	for i := range result {
		result[i] = make([]float64, columns)
	}
	return result
}

// Multiply calcula el producto m·other. Devuelve nil si las dimensiones no encajan
// (columnas de m distintas de filas de other).
//
// Se usa en los tests para verificar Q·R ≈ A y QᵀQ ≈ I
func (m Matrix) Multiply(other Matrix) Matrix {
	inner := m.Columns()
	if inner != other.Rows() {
		return nil
	}

	result := zeros(m.Rows(), other.Columns())
	for i := range m {
		for k := 0; k < inner; k++ {
			// Recorrer k en el bucle intermedio mantiene el acceso a other[k] contiguo
			// en memoria, que es notablemente más rápido que el orden i-j-k clásico.
			factor := m[i][k]
			if factor == 0 {
				continue
			}
			for j := range other[k] {
				result[i][j] += factor * other[k][j]
			}
		}
	}
	return result
}

// Transpose devuelve la transpuesta de la matriz.
func (m Matrix) Transpose() Matrix {
	result := zeros(m.Columns(), m.Rows())
	for i, row := range m {
		for j, value := range row {
			result[j][i] = value
		}
	}
	return result
}

// MaxAbs devuelve el mayor valor absoluto de la matriz. Sirve para escalar tolerancias:
// un residuo de 1e-9 es despreciable en una matriz de valores ~1e6 y enorme en una de ~1e-6.
func (m Matrix) MaxAbs() float64 {
	var largest float64
	for _, row := range m {
		for _, value := range row {
			if abs := math.Abs(value); abs > largest {
				largest = abs
			}
		}
	}
	return largest
}
