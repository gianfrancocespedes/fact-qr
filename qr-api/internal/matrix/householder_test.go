package matrix

import (
	"math"
	"math/rand"
	"testing"
)

// tolerance escala con la magnitud de la matriz: un residuo de 1e-9 es despreciable en
// valores del orden de 1e6 y enorme en valores de 1e-6.
func tolerance(m Matrix) float64 {
	return 1e-9 * math.Max(1, m.MaxAbs())
}

// assertReconstructs verifica Q·R ≈ A, la propiedad que define la factorización.
func assertReconstructs(t *testing.T, original Matrix, f Factorization) {
	t.Helper()

	product := f.Q.Multiply(f.R)
	if product == nil {
		t.Fatalf("dimensiones incompatibles: Q es %dx%d y R es %dx%d",
			f.Q.Rows(), f.Q.Columns(), f.R.Rows(), f.R.Columns())
	}

	limit := tolerance(original)
	for i := range original {
		for j := range original[i] {
			if diff := math.Abs(original[i][j] - product[i][j]); diff > limit {
				t.Errorf("Q·R ≠ A en (%d,%d): esperado %g, obtenido %g (diferencia %g)",
					i, j, original[i][j], product[i][j], diff)
			}
		}
	}
}

// assertOrthogonal verifica QᵀQ ≈ I. Es la propiedad que Gram-Schmidt pierde y que
// justifica haber elegido Householder.
func assertOrthogonal(t *testing.T, q Matrix) {
	t.Helper()

	product := q.Transpose().Multiply(q)
	limit := tolerance(q)

	for i := range product {
		for j := range product[i] {
			expected := 0.0
			if i == j {
				expected = 1.0
			}
			if diff := math.Abs(product[i][j] - expected); diff > limit {
				t.Errorf("QᵀQ ≠ I en (%d,%d): esperado %g, obtenido %g", i, j, expected, product[i][j])
			}
		}
	}
}

// assertUpperTriangular verifica que R no tenga nada distinto de cero bajo la diagonal.
func assertUpperTriangular(t *testing.T, r Matrix) {
	t.Helper()

	for i := range r {
		for j := range r[i] {
			if i > j && r[i][j] != 0 {
				t.Errorf("R no es triangular superior: R[%d][%d] = %g", i, j, r[i][j])
			}
		}
	}
}

// assertValidFactorization comprueba las tres propiedades de una vez.
//
// Se verifican propiedades y no valores concretos porque la QR no es única: invertir el
// signo de una columna de Q y de la fila correspondiente de R da otra factorización
// igualmente válida. Un test contra valores fijos fallaría ante implementaciones correctas.
func assertValidFactorization(t *testing.T, original Matrix) Factorization {
	t.Helper()

	f := original.Decompose()
	assertReconstructs(t, original, f)
	assertOrthogonal(t, f.Q)
	assertUpperTriangular(t, f.R)
	return f
}

func TestDecomposeShapes(t *testing.T) {
	cases := map[string]Matrix{
		"cuadrada 3x3":       {{1, 2, 3}, {4, 5, 6}, {7, 8, 10}},
		"alta 3x2 (m>n)":     {{12, -51}, {6, 167}, {-4, 24}},
		"ancha 2x3 (m<n)":    {{1, 2, 3}, {4, 5, 6}},
		"ancha 3x4":          {{2, -1, 3, 4}, {1, 5, -2, 0}, {3, 1, 1, -1}},
		"una celda 1x1":      {{5}},
		"columna 3x1":        {{3}, {4}, {0}},
		"fila 1x3":           {{1, 2, 3}},
		"con negativos":      {{-4, -7}, {2, -3}, {5, 1}},
		"con ceros":          {{0, 2}, {0, 0}, {0, 5}},
		"identidad 3x3":      identity(3),
		"ya triangular":      {{2, 3, 4}, {0, 5, 6}, {0, 0, 7}},
		"columnas repetidas": {{1, 1}, {2, 2}, {3, 3}},
		"valores grandes":    {{1e8, 2e8}, {3e8, 4e8}},
		"valores diminutos":  {{1e-8, 2e-8}, {3e-8, 4e-8}},
	}

	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			f := assertValidFactorization(t, input)

			if f.Q.Rows() != input.Rows() || f.Q.Columns() != input.Rows() {
				t.Errorf("Q debe ser %dx%d, es %dx%d",
					input.Rows(), input.Rows(), f.Q.Rows(), f.Q.Columns())
			}
			if f.R.Rows() != input.Rows() || f.R.Columns() != input.Columns() {
				t.Errorf("R debe ser %dx%d, es %dx%d",
					input.Rows(), input.Columns(), f.R.Rows(), f.R.Columns())
			}
		})
	}
}

// TestDecomposeIllConditioned usa el caso donde Gram-Schmidt clásico falla: columnas casi
// linealmente dependientes. Householder debe mantener la ortogonalidad de Q.
func TestDecomposeIllConditioned(t *testing.T) {
	input := Matrix{
		{1, 1, 1},
		{1, 1.0001, 1},
		{1, 1, 1.0001},
	}
	assertValidFactorization(t, input)
}

// TestDecomposeDoesNotMutateInput protege una garantía del contrato: la respuesta de la
// API incluye la matriz de entrada, así que factorizar no puede alterarla.
func TestDecomposeDoesNotMutateInput(t *testing.T) {
	input := Matrix{{12, -51}, {6, 167}, {-4, 24}}
	snapshot := input.Clone()

	input.Decompose()

	for i := range input {
		for j := range input[i] {
			if input[i][j] != snapshot[i][j] {
				t.Fatalf("Decompose mutó la entrada en (%d,%d): %g → %g",
					i, j, snapshot[i][j], input[i][j])
			}
		}
	}
}

// TestDecomposeRandomized cubre tamaños y valores que los casos escritos a mano no
// alcanzan. Las tres propiedades valen para cualquier matriz, así que sirven de oráculo.
func TestDecomposeRandomized(t *testing.T) {
	source := rand.New(rand.NewSource(20240615))

	for iteration := 0; iteration < 200; iteration++ {
		rows := 1 + source.Intn(8)
		columns := 1 + source.Intn(8)

		input := zeros(rows, columns)
		for i := range input {
			for j := range input[i] {
				input[i][j] = (source.Float64() - 0.5) * 200
			}
		}

		assertValidFactorization(t, input)
	}
}

// TestDecomposeAlreadyTriangularSkipsReflection documenta el caso degenerado: si una
// columna ya tiene ceros bajo la diagonal, no hay reflexión que aplicar. Sin ese salto,
// el algoritmo dividiría entre cero.
func TestDecomposeAlreadyTriangularSkipsReflection(t *testing.T) {
	input := Matrix{{0, 0}, {0, 0}, {0, 0}}
	f := assertValidFactorization(t, input)

	// Sin reflexiones, Q queda como la identidad.
	for i := range f.Q {
		for j := range f.Q[i] {
			expected := 0.0
			if i == j {
				expected = 1.0
			}
			if f.Q[i][j] != expected {
				t.Errorf("Q debería ser la identidad; Q[%d][%d] = %g", i, j, f.Q[i][j])
			}
		}
	}
}
