package matrix

import "math"

// nearZero es el umbral bajo el cual un vector se considera nulo. Evita dividir entre
// cantidades despreciables al normalizar el reflector, lo que amplificaría el error.
const nearZero = 1e-300

// Factorization es el resultado de descomponer A en Q·R.
type Factorization struct {
	// Q es ortogonal de m×m: sus columnas son perpendiculares entre sí y de norma 1.
	Q Matrix
	// R es triangular superior de m×n: todo lo que está bajo la diagonal es cero.
	R Matrix
}

// Decompose calcula la factorización QR completa mediante reflexiones de Householder.
//
// Cada reflexión anula de golpe todos los elementos bajo la diagonal de una columna.
// Como reflejar conserva la norma, la transformación es ortogonal por construcción y el
// redondeo no puede degradar esa propiedad
//
// No mutará la matriz receptora: R parte de una copia.
func (m Matrix) Decompose() Factorization {
	rows, columns := m.Rows(), m.Columns()

	r := m.Clone()
	q := identity(rows)

	// La última fila no necesita reflexión: no queda nada debajo que anular. Por eso el
	// límite es rows-1 y no rows, y con una matriz de una sola fila no se ejecuta ninguna.
	steps := min(columns, rows-1)

	for k := 0; k < steps; k++ {
		v, ok := buildReflector(r, k)
		if !ok {
			// La columna ya tiene ceros bajo la diagonal: reflejar sería multiplicar por
			// la identidad. Saltar el paso evita una división entre cero.
			continue
		}

		applyLeft(r, v, k)
		applyRight(q, v, k)
	}

	forceZeroBelowDiagonal(r)
	return Factorization{Q: q, R: r}
}

// buildReflector construye el vector normal al espejo que lleva la subcolumna k sobre el
// primer eje, dejando ceros debajo. Devuelve false si la columna ya está alineada.
//
// El vector se devuelve normalizado (‖v‖ = 1), lo que simplifica las aplicaciones
// posteriores a x − 2v(v·x) sin arrastrar el divisor vᵀv en cada operación.
func buildReflector(m Matrix, k int) ([]float64, bool) {
	rows := m.Rows()

	// hypot en dos pasadas evita desbordamiento con valores muy grandes o muy pequeños.
	var norm float64
	for i := k; i < rows; i++ {
		norm = math.Hypot(norm, m[i][k])
	}
	if norm < nearZero {
		return nil, false
	}

	// El signo se elige opuesto al del pivote para que las magnitudes se sumen en lugar
	// de restarse. Con el signo contrario, una columna ya casi alineada produciría
	// v[0] ≈ 0 por cancelación catastrófica, y normalizar ese residuo amplificaría el
	// error de redondeo hasta arruinar la ortogonalidad de Q.
	alpha := -math.Copysign(norm, m[k][k])

	v := make([]float64, rows-k)
	for i := range v {
		v[i] = m[k+i][k]
	}
	v[0] -= alpha

	var vNorm float64
	for _, value := range v {
		vNorm = math.Hypot(vNorm, value)
	}
	if vNorm < nearZero {
		return nil, false
	}

	for i := range v {
		v[i] /= vNorm
	}
	return v, true
}

// applyLeft aplica la reflexión por la izquierda sobre las filas k..m de la matriz
// (R ← H·R), que es lo que va creando los ceros bajo la diagonal.
//
// Nunca se construye H explícitamente: formarla y multiplicar costaría O(m²n) por paso,
// mientras que aplicarla como x − 2v(v·x) cuesta O(mn).
func applyLeft(m Matrix, v []float64, k int) {
	columns := m.Columns()

	for j := k; j < columns; j++ {
		var dot float64
		for i := range v {
			dot += v[i] * m[k+i][j]
		}

		dot *= 2
		for i := range v {
			m[k+i][j] -= dot * v[i]
		}
	}
}

// applyRight aplica la reflexión por la derecha sobre las columnas k..m (Q ← Q·H).
//
// El lado importa: R acumula por la izquierda y Q por la derecha. Invertirlo produce la
// transpuesta de Q, que sigue pareciendo ortogonal en las pruebas superficiales pero hace
// que Q·R deje de reconstruir A.
func applyRight(m Matrix, v []float64, k int) {
	rows := m.Rows()

	for i := 0; i < rows; i++ {
		var dot float64
		for j := range v {
			dot += m[i][k+j] * v[j]
		}

		dot *= 2
		for j := range v {
			m[i][k+j] -= dot * v[j]
		}
	}
}

// forceZeroBelowDiagonal limpia los residuos de redondeo bajo la diagonal de R.
//
// Matemáticamente esas posiciones son cero exacto por construcción; en coma flotante
// quedan valores del orden de 1e-17. Se anulan explícitamente para que R sea triangular
// superior de forma literal y no aproximada, tanto en la respuesta JSON como al verificar
// si R es diagonal.
func forceZeroBelowDiagonal(r Matrix) {
	for i := 1; i < r.Rows(); i++ {
		for j := 0; j < min(i, r.Columns()); j++ {
			r[i][j] = 0
		}
	}
}
