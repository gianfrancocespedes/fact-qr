package matrix

import (
	"errors"
	"math"
	"testing"

	"github.com/gianfrancocespedes/fact-qr/qr-api/internal/apperror"
)

// assertErrorCode comprueba que el error sea de dominio y traiga el código esperado.
func assertErrorCode(t *testing.T, err error, want apperror.Code) *apperror.Error {
	t.Helper()

	var domainError *apperror.Error
	if !errors.As(err, &domainError) {
		t.Fatalf("se esperaba *apperror.Error, se obtuvo %T (%v)", err, err)
	}
	if domainError.Code != want {
		t.Fatalf("código: esperado %s, obtenido %s", want, domainError.Code)
	}
	return domainError
}

func TestNewAcceptsValidMatrix(t *testing.T) {
	got, err := New([][]float64{{1, 2, 3}, {4, 5, 6}})
	if err != nil {
		t.Fatalf("no se esperaba error, se obtuvo %v", err)
	}

	if got.Rows() != 2 || got.Columns() != 3 {
		t.Errorf("dimensiones: esperado 2x3, obtenido %dx%d", got.Rows(), got.Columns())
	}
}

// TestNewClonesInput evita el aliasing: si New devolviera la misma memoria recibida, una
// mutación posterior del dominio alteraría el JSON de entrada que se reporta al cliente.
func TestNewClonesInput(t *testing.T) {
	values := [][]float64{{1, 2}, {3, 4}}

	got, err := New(values)
	if err != nil {
		t.Fatalf("no se esperaba error, se obtuvo %v", err)
	}

	values[0][0] = 99
	if got[0][0] == 99 {
		t.Error("New devolvió una vista de la entrada en vez de una copia")
	}
}

func TestNewRejectsEmptyMatrix(t *testing.T) {
	cases := map[string][][]float64{
		"nil":                nil,
		"sin filas":          {},
		"primera fila vacía": {{}},
	}

	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := New(input)
			assertErrorCode(t, err, apperror.CodeEmptyMatrix)
		})
	}
}

func TestNewRejectsInconsistentRows(t *testing.T) {
	_, err := New([][]float64{{1, 2, 3}, {4, 5}})
	domainError := assertErrorCode(t, err, apperror.CodeInconsistentRowLength)

	// Las coordenadas se reportan en base 1: el destinatario lee una grilla, no un array.
	if got := domainError.Details["row"]; got != 2 {
		t.Errorf("details.row: esperado 2, obtenido %v", got)
	}
	if got := domainError.Details["expected"]; got != 3 {
		t.Errorf("details.expected: esperado 3, obtenido %v", got)
	}
	if got := domainError.Details["actual"]; got != 2 {
		t.Errorf("details.actual: esperado 2, obtenido %v", got)
	}
}

func TestNewRejectsNonFiniteValues(t *testing.T) {
	cases := map[string]float64{
		"NaN":               math.NaN(),
		"infinito positivo": math.Inf(1),
		"infinito negativo": math.Inf(-1),
	}

	for name, value := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := New([][]float64{{1, 2}, {3, value}})
			domainError := assertErrorCode(t, err, apperror.CodeInvalidNumber)

			if got := domainError.Details["row"]; got != 2 {
				t.Errorf("details.row: esperado 2, obtenido %v", got)
			}
			if got := domainError.Details["column"]; got != 2 {
				t.Errorf("details.column: esperado 2, obtenido %v", got)
			}
		})
	}
}

func TestNewRejectsOversizedMatrix(t *testing.T) {
	oversized := make([][]float64, MaxDimension+1)
	for i := range oversized {
		oversized[i] = make([]float64, 1)
	}

	_, err := New(oversized)
	assertErrorCode(t, err, apperror.CodeMatrixTooLarge)
}

func TestMultiply(t *testing.T) {
	a := Matrix{{1, 2, 3}, {4, 5, 6}}
	b := Matrix{{7, 6}, {9, 8}, {11, 12}}

	got := a.Multiply(b)
	want := Matrix{{58, 58}, {139, 136}}

	assertEqual(t, got, want)
}

func TestMultiplyRejectsMismatchedDimensions(t *testing.T) {
	a := Matrix{{1, 2, 3}}
	b := Matrix{{1, 2}}

	if got := a.Multiply(b); got != nil {
		t.Errorf("se esperaba nil con dimensiones incompatibles, se obtuvo %v", got)
	}
}

func TestTranspose(t *testing.T) {
	input := Matrix{{1, 2, 3}, {4, 5, 6}}
	want := Matrix{{1, 4}, {2, 5}, {3, 6}}

	assertEqual(t, input.Transpose(), want)
}

func TestMaxAbs(t *testing.T) {
	input := Matrix{{1, -7.5}, {3, 2}}

	if got := input.MaxAbs(); got != 7.5 {
		t.Errorf("esperado 7.5, obtenido %g", got)
	}
}
