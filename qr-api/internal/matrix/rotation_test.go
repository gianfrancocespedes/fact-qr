package matrix

import (
	"errors"
	"testing"

	"github.com/gianfrancocespedes/fact-qr/qr-api/internal/apperror"
)

func assertEqual(t *testing.T, got, want Matrix) {
	t.Helper()

	if got.Rows() != want.Rows() || got.Columns() != want.Columns() {
		t.Fatalf("dimensiones: esperado %dx%d, obtenido %dx%d",
			want.Rows(), want.Columns(), got.Rows(), got.Columns())
	}
	for i := range want {
		for j := range want[i] {
			if got[i][j] != want[i][j] {
				t.Errorf("en (%d,%d): esperado %g, obtenido %g", i, j, want[i][j], got[i][j])
			}
		}
	}
}

func TestRotateClockwise(t *testing.T) {
	// Una 2×3 rotada queda 3×2: la rotación intercambia las dimensiones.
	input := Matrix{{1, 2, 3}, {4, 5, 6}}
	want := Matrix{{4, 1}, {5, 2}, {6, 3}}

	assertEqual(t, input.Rotate(1, Clockwise), want)
}

func TestRotateCounterClockwise(t *testing.T) {
	input := Matrix{{1, 2, 3}, {4, 5, 6}}
	want := Matrix{{3, 6}, {2, 5}, {1, 4}}

	assertEqual(t, input.Rotate(1, CounterClockwise), want)
}

// TestRotateZeroIsIdentity cubre el valor por defecto del contrato: omitir el parámetro
// equivale a factorizar la matriz tal como llegó.
func TestRotateZeroIsIdentity(t *testing.T) {
	input := Matrix{{1, 2, 3}, {4, 5, 6}}

	for _, direction := range []Direction{Clockwise, CounterClockwise} {
		assertEqual(t, input.Rotate(0, direction), input)
	}
}

// TestRotateFourTimesIsIdentity es el sustento de que el rango 0–3 no limite nada: la
// cuarta rotación devuelve la matriz original, así que 0–3 cubre todas las distintas.
func TestRotateFourTimesIsIdentity(t *testing.T) {
	input := Matrix{{1, 2, 3}, {4, 5, 6}}

	for _, direction := range []Direction{Clockwise, CounterClockwise} {
		result := input.Clone()
		for i := 0; i < 4; i++ {
			result = result.Rotate(1, direction)
		}
		assertEqual(t, result, input)
	}
}

// TestRotateOppositeDirectionsCancel verifica que ambos sentidos sean realmente inversos.
func TestRotateOppositeDirectionsCancel(t *testing.T) {
	input := Matrix{{1, 2, 3}, {4, 5, 6}}

	result := input.Rotate(1, Clockwise).Rotate(1, CounterClockwise)
	assertEqual(t, result, input)
}

// TestRotateThreeEqualsOneOpposite comprueba la equivalencia 3 horarias ≡ 1 antihoraria.
func TestRotateThreeEqualsOneOpposite(t *testing.T) {
	input := Matrix{{1, 2, 3}, {4, 5, 6}}

	assertEqual(t, input.Rotate(3, Clockwise), input.Rotate(1, CounterClockwise))
	assertEqual(t, input.Rotate(3, CounterClockwise), input.Rotate(1, Clockwise))
}

func TestRotateTwiceReversesBothAxes(t *testing.T) {
	input := Matrix{{1, 2, 3}, {4, 5, 6}}
	want := Matrix{{6, 5, 4}, {3, 2, 1}}

	// Con dos rotaciones el sentido es irrelevante: ambos llegan al mismo resultado.
	assertEqual(t, input.Rotate(2, Clockwise), want)
	assertEqual(t, input.Rotate(2, CounterClockwise), want)
}

// TestRotateDoesNotMutateInput protege la trazabilidad: la respuesta muestra la matriz
// original junto a la rotada, así que la original no puede haber cambiado.
func TestRotateDoesNotMutateInput(t *testing.T) {
	input := Matrix{{1, 2, 3}, {4, 5, 6}}
	snapshot := input.Clone()

	input.Rotate(1, Clockwise)

	assertEqual(t, input, snapshot)
}

func TestValidateRotations(t *testing.T) {
	cases := map[string]struct {
		times     int
		wantError bool
	}{
		"cero es válido":      {0, false},
		"tres es válido":      {3, false},
		"negativo se rechaza": {-1, true},
		"cuatro se rechaza":   {4, true},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			err := ValidateRotations(tc.times)

			if !tc.wantError {
				if err != nil {
					t.Fatalf("no se esperaba error, se obtuvo %v", err)
				}
				return
			}

			var domainError *apperror.Error
			if !errors.As(err, &domainError) {
				t.Fatalf("se esperaba *apperror.Error, se obtuvo %T", err)
			}
			if domainError.Code != apperror.CodeInvalidRotations {
				t.Errorf("código: esperado %s, obtenido %s",
					apperror.CodeInvalidRotations, domainError.Code)
			}
		})
	}
}

func TestParseDirection(t *testing.T) {
	cases := map[string]struct {
		input     string
		want      Direction
		wantError bool
	}{
		"vacío usa el defecto": {"", Clockwise, false},
		"horario":              {"clockwise", Clockwise, false},
		"antihorario":          {"counterclockwise", CounterClockwise, false},
		"valor desconocido":    {"diagonal", "", true},
		"distinto case":        {"Clockwise", "", true},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := ParseDirection(tc.input)

			if tc.wantError {
				var domainError *apperror.Error
				if !errors.As(err, &domainError) {
					t.Fatalf("se esperaba *apperror.Error, se obtuvo %T", err)
				}
				if domainError.Code != apperror.CodeInvalidDirection {
					t.Errorf("código: esperado %s, obtenido %s",
						apperror.CodeInvalidDirection, domainError.Code)
				}
				return
			}

			if err != nil {
				t.Fatalf("no se esperaba error, se obtuvo %v", err)
			}
			if got != tc.want {
				t.Errorf("esperado %q, obtenido %q", tc.want, got)
			}
		})
	}
}
