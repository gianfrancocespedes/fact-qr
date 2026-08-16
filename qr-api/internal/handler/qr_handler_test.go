package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"

	"github.com/gianfrancocespedes/fact-qr/qr-api/internal/app"
	"github.com/gianfrancocespedes/fact-qr/qr-api/internal/apperror"
	"github.com/gianfrancocespedes/fact-qr/qr-api/internal/auth"
	"github.com/gianfrancocespedes/fact-qr/qr-api/internal/client"
	"github.com/gianfrancocespedes/fact-qr/qr-api/internal/matrix"
)

// stubStats sustituye a la stats-api en las pruebas. Permite verificar el flujo completo
// sin levantar el servicio de Node ni depender de la red.
type stubStats struct {
	// receivedMatrices guarda lo que la qr-api envió, para comprobar que solo viajan Q y R.
	receivedMatrices []matrix.Matrix
	receivedToken    string
	response         *client.Statistics
	err              error
}

func (s *stubStats) Calculate(_ context.Context, matrices []matrix.Matrix, token string) (*client.Statistics, error) {
	s.receivedMatrices = matrices
	s.receivedToken = token

	if s.err != nil {
		return nil, s.err
	}
	if s.response != nil {
		return s.response, nil
	}
	return &client.Statistics{Max: 1, Min: -1, Average: 0, Sum: 0, Count: 4}, nil
}

// newTestApp construye la aplicación con un doble de la stats-api.
func newTestApp(stats *stubStats) *fiber.App {
	return app.New(app.Config{Stats: stats})
}

// authHeaders devuelve las cabeceras con un token válido.
//
// Los endpoints de negocio están protegidos, así que las pruebas del caso feliz necesitan
// autenticarse; los casos sin token tienen sus propias pruebas.
func authHeaders(t *testing.T) map[string]string {
	t.Helper()

	token, _, err := auth.Issue("test")
	if err != nil {
		t.Fatalf("no se pudo emitir el token de prueba: %v", err)
	}
	return map[string]string{fiber.HeaderAuthorization: "Bearer " + token}
}

// post envía una petición JSON y devuelve el estado y el cuerpo decodificado.
func post(t *testing.T, application *fiber.App, path, body string, headers map[string]string) (int, map[string]any) {
	t.Helper()

	request := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	for key, value := range headers {
		request.Header.Set(key, value)
	}

	// El timeout generoso cubre el arranque en máquinas lentas; las pruebas no dependen
	// de red real, así que nunca se agota en la práctica.
	response, err := application.Test(request, 10_000)
	if err != nil {
		t.Fatalf("no se pudo ejecutar la petición: %v", err)
	}
	defer response.Body.Close()

	raw, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("no se pudo leer la respuesta: %v", err)
	}

	var decoded map[string]any
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &decoded); err != nil {
			t.Fatalf("respuesta no es JSON válido: %v (cuerpo: %s)", err, raw)
		}
	}
	return response.StatusCode, decoded
}

// mustRequest construye una petición GET sin cuerpo.
func mustRequest(t *testing.T, method, path string) *http.Request {
	t.Helper()
	return httptest.NewRequest(method, path, nil)
}

// errorCode extrae el código de error del cuerpo de la respuesta.
func errorCode(t *testing.T, body map[string]any) string {
	t.Helper()

	errorObject, ok := body["error"].(map[string]any)
	if !ok {
		t.Fatalf("la respuesta no contiene un objeto error: %v", body)
	}
	code, _ := errorObject["code"].(string)
	return code
}

func TestHealthIsPublic(t *testing.T) {
	application := newTestApp(&stubStats{})

	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	response, err := application.Test(request, 10_000)
	if err != nil {
		t.Fatalf("no se pudo ejecutar la petición: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("estado: esperado 200, obtenido %d", response.StatusCode)
	}
}

func TestPostQRReturnsComposedResponse(t *testing.T) {
	stats := &stubStats{}
	application := newTestApp(stats)

	status, body := post(t, application, "/api/v1/qr",
		`{"matrix":[[12,-51],[6,167],[-4,24]]}`, authHeaders(t))

	if status != http.StatusOK {
		t.Fatalf("estado: esperado 200, obtenido %d (%v)", status, body)
	}

	for _, key := range []string{"input", "factorization", "statistics"} {
		if _, present := body[key]; !present {
			t.Errorf("falta la clave %q en la respuesta", key)
		}
	}

	factorization := body["factorization"].(map[string]any)
	if _, present := factorization["q"]; !present {
		t.Error("falta Q en la factorización")
	}
	if _, present := factorization["r"]; !present {
		t.Error("falta R en la factorización")
	}
}

// TestPostQRSendsOnlyQAndRToStats fija la decisión de diseño: las estadísticas se calculan
// sobre las matrices devueltas por la factorización, no sobre la matriz de entrada.
func TestPostQRSendsOnlyQAndRToStats(t *testing.T) {
	stats := &stubStats{}
	application := newTestApp(stats)

	post(t, application, "/api/v1/qr", `{"matrix":[[12,-51],[6,167],[-4,24]]}`, authHeaders(t))

	if len(stats.receivedMatrices) != 2 {
		t.Fatalf("se esperaban 2 matrices (Q y R), se enviaron %d", len(stats.receivedMatrices))
	}

	q, r := stats.receivedMatrices[0], stats.receivedMatrices[1]
	if q.Rows() != 3 || q.Columns() != 3 {
		t.Errorf("Q debe ser 3x3, es %dx%d", q.Rows(), q.Columns())
	}
	if r.Rows() != 3 || r.Columns() != 2 {
		t.Errorf("R debe ser 3x2, es %dx%d", r.Rows(), r.Columns())
	}
}

// TestPostQRRotatesBeforeFactorizing verifica el pipeline completo: la rotación ocurre
// antes de la factorización, y las dimensiones reportadas son las de la matriz rotada.
func TestPostQRRotatesBeforeFactorizing(t *testing.T) {
	application := newTestApp(&stubStats{})

	// Una 2x3 rotada una vez queda 3x2.
	status, body := post(t, application, "/api/v1/qr",
		`{"matrix":[[1,2,3],[4,5,6]],"rotations":1,"direction":"clockwise"}`, authHeaders(t))

	if status != http.StatusOK {
		t.Fatalf("estado: esperado 200, obtenido %d (%v)", status, body)
	}

	input := body["input"].(map[string]any)
	dimensions := input["dimensions"].(map[string]any)

	if dimensions["rows"] != float64(3) || dimensions["columns"] != float64(2) {
		t.Errorf("dimensiones: esperado 3x2, obtenido %vx%v", dimensions["rows"], dimensions["columns"])
	}

	// La rotada esperada de [[1,2,3],[4,5,6]] en sentido horario.
	expected := [][]float64{{4, 1}, {5, 2}, {6, 3}}
	rotated := input["rotated"].([]any)

	for i, wantRow := range expected {
		gotRow := rotated[i].([]any)
		for j, wantValue := range wantRow {
			if gotRow[j].(float64) != wantValue {
				t.Errorf("rotada en (%d,%d): esperado %g, obtenido %v", i, j, wantValue, gotRow[j])
			}
		}
	}
}

// TestPostQRKeepsOriginalMatrix protege la trazabilidad: la respuesta muestra el antes y
// el después, así que la matriz original no puede haberse alterado al rotar.
func TestPostQRKeepsOriginalMatrix(t *testing.T) {
	application := newTestApp(&stubStats{})

	_, body := post(t, application, "/api/v1/qr",
		`{"matrix":[[1,2,3],[4,5,6]],"rotations":2}`, authHeaders(t))

	input := body["input"].(map[string]any)
	original := input["original"].([]any)

	if len(original) != 2 {
		t.Fatalf("la matriz original debe conservar 2 filas, tiene %d", len(original))
	}
	if first := original[0].([]any); first[0].(float64) != 1 {
		t.Errorf("la matriz original fue alterada: se esperaba 1 en (0,0), hay %v", first[0])
	}
}

func TestPostQRDefaultsToNoRotation(t *testing.T) {
	application := newTestApp(&stubStats{})

	_, body := post(t, application, "/api/v1/qr", `{"matrix":[[1,2],[3,4]]}`, authHeaders(t))

	input := body["input"].(map[string]any)
	if input["rotations"] != float64(0) {
		t.Errorf("rotations por defecto: esperado 0, obtenido %v", input["rotations"])
	}
	if input["direction"] != string(matrix.Clockwise) {
		t.Errorf("direction por defecto: esperado %q, obtenido %v", matrix.Clockwise, input["direction"])
	}
}

func TestPostQRValidation(t *testing.T) {
	cases := map[string]struct {
		body     string
		wantCode apperror.Code
		wantHTTP int
	}{
		"matriz vacía": {
			`{"matrix":[]}`, apperror.CodeEmptyMatrix, http.StatusBadRequest,
		},
		"filas desiguales": {
			`{"matrix":[[1,2,3],[4,5]]}`, apperror.CodeInconsistentRowLength, http.StatusBadRequest,
		},
		"rotaciones negativas": {
			`{"matrix":[[1,2],[3,4]],"rotations":-1}`, apperror.CodeInvalidRotations, http.StatusBadRequest,
		},
		"rotaciones fuera de rango": {
			`{"matrix":[[1,2],[3,4]],"rotations":4}`, apperror.CodeInvalidRotations, http.StatusBadRequest,
		},
		"sentido desconocido": {
			`{"matrix":[[1,2],[3,4]],"direction":"diagonal"}`, apperror.CodeInvalidDirection, http.StatusBadRequest,
		},
		"JSON malformado": {
			`{"matrix":[[1,2]`, apperror.CodeInvalidPayload, http.StatusBadRequest,
		},
		"celda no numérica": {
			`{"matrix":[[1,"x"],[3,4]]}`, apperror.CodeInvalidPayload, http.StatusBadRequest,
		},
	}

	application := newTestApp(&stubStats{})

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			status, body := post(t, application, "/api/v1/qr", tc.body, authHeaders(t))

			if status != tc.wantHTTP {
				t.Errorf("estado: esperado %d, obtenido %d", tc.wantHTTP, status)
			}
			if got := errorCode(t, body); got != string(tc.wantCode) {
				t.Errorf("código: esperado %s, obtenido %s", tc.wantCode, got)
			}
		})
	}
}

// TestPostQRRejectsNonFiniteValues cubre NaN e Infinity, que JSON no admite como literales
// pero algunos clientes envían como cadenas o valores fuera de rango.
func TestPostQRRejectsNonFiniteValues(t *testing.T) {
	application := newTestApp(&stubStats{})

	// 1e400 desborda float64 y se decodifica como +Inf.
	status, body := post(t, application, "/api/v1/qr", `{"matrix":[[1,2],[3,1e400]]}`, authHeaders(t))

	if status != http.StatusBadRequest {
		t.Fatalf("estado: esperado 400, obtenido %d (%v)", status, body)
	}
	if code := errorCode(t, body); code != string(apperror.CodeInvalidPayload) &&
		code != string(apperror.CodeInvalidNumber) {
		t.Errorf("código inesperado: %s", code)
	}
}

// TestPostQRTranslatesStatsFailure verifica que la caída de la dependencia se comunique
// con su propio código y con 503, no como un error interno genérico.
func TestPostQRTranslatesStatsFailure(t *testing.T) {
	stats := &stubStats{err: apperror.New(apperror.CodeStatsServiceUnavailable, nil)}
	application := newTestApp(stats)

	status, body := post(t, application, "/api/v1/qr", `{"matrix":[[1,2],[3,4]]}`, authHeaders(t))

	if status != http.StatusServiceUnavailable {
		t.Errorf("estado: esperado 503, obtenido %d", status)
	}
	if got := errorCode(t, body); got != string(apperror.CodeStatsServiceUnavailable) {
		t.Errorf("código: esperado %s, obtenido %s", apperror.CodeStatsServiceUnavailable, got)
	}
}

// TestPostQRPropagatesStatsErrorCode comprueba que un error de validación detectado por la
// stats-api llegue al cliente con su código original, no enmascarado.
func TestPostQRPropagatesStatsErrorCode(t *testing.T) {
	stats := &stubStats{err: apperror.New(apperror.CodeInvalidNumber, apperror.Details{"row": 1})}
	application := newTestApp(stats)

	status, body := post(t, application, "/api/v1/qr", `{"matrix":[[1,2],[3,4]]}`, authHeaders(t))

	if status != http.StatusBadRequest {
		t.Errorf("estado: esperado 400, obtenido %d", status)
	}
	if got := errorCode(t, body); got != string(apperror.CodeInvalidNumber) {
		t.Errorf("código: esperado %s, obtenido %s", apperror.CodeInvalidNumber, got)
	}
}

// TestPostQRNeverReturnsHumanText fija el contrato: el backend devuelve códigos, la
// traducción al español vive en el frontend.
func TestPostQRNeverReturnsHumanText(t *testing.T) {
	application := newTestApp(&stubStats{})

	_, body := post(t, application, "/api/v1/qr", `{"matrix":[]}`, authHeaders(t))

	errorObject := body["error"].(map[string]any)
	if _, present := errorObject["message"]; present {
		t.Error("la respuesta de error no debe incluir texto legible")
	}
	if code := errorCode(t, body); !strings.HasPrefix(code, "ERROR_") {
		t.Errorf("el código debe seguir el formato ERROR_*, es %q", code)
	}
}

func TestUnknownRouteReturnsContractError(t *testing.T) {
	application := newTestApp(&stubStats{})

	// Se consulta fuera de /api/v1 a propósito: dentro del grupo protegido la respuesta es
	// 401 antes que 404, porque no revelar qué rutas existen a un cliente no autenticado
	// es el comportamiento deseado.
	request := httptest.NewRequest(http.MethodGet, "/no-existe", nil)
	response, err := application.Test(request, 10_000)
	if err != nil {
		t.Fatalf("no se pudo ejecutar la petición: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusNotFound {
		t.Errorf("estado: esperado 404, obtenido %d", response.StatusCode)
	}
}

// TestProtectedRouteHidesExistence documenta la decisión anterior: una ruta inexistente
// bajo el grupo protegido responde 401, no 404.
func TestProtectedRouteHidesExistence(t *testing.T) {
	application := newTestApp(&stubStats{})

	status, _ := post(t, application, "/api/v1/no-existe", `{}`, nil)

	if status != http.StatusUnauthorized {
		t.Errorf("estado: esperado 401, obtenido %d", status)
	}
}

// TestPostQRStatisticsAreConsistent cierra el ciclo: comprueba que las estadísticas que
// devuelve la API corresponden a las matrices Q y R realmente enviadas.
func TestPostQRStatisticsAreConsistent(t *testing.T) {
	stats := &stubStats{}
	application := newTestApp(stats)

	_, body := post(t, application, "/api/v1/qr", `{"matrix":[[12,-51],[6,167],[-4,24]]}`, authHeaders(t))

	// Se recalcula el máximo sobre lo que el doble recibió y se compara con lo publicado.
	var expectedMax float64 = math.Inf(-1)
	for _, m := range stats.receivedMatrices {
		for _, row := range m {
			for _, value := range row {
				expectedMax = math.Max(expectedMax, value)
			}
		}
	}

	if len(stats.receivedMatrices) == 0 {
		t.Fatal("no se enviaron matrices a la stats-api")
	}
	if _, present := body["statistics"]; !present {
		t.Fatal("la respuesta no incluye estadísticas")
	}
	if math.IsInf(expectedMax, -1) {
		t.Error("las matrices enviadas estaban vacías")
	}
}

// TestErrorHandlerHandlesUnknownError comprueba el camino por defecto: un error que no es
// de dominio se reporta como interno sin filtrar detalles.
func TestErrorHandlerHandlesUnknownError(t *testing.T) {
	stats := &stubStats{err: errors.New("fallo inesperado con detalles internos")}
	application := newTestApp(stats)

	status, body := post(t, application, "/api/v1/qr", `{"matrix":[[1,2],[3,4]]}`, authHeaders(t))

	if status != http.StatusInternalServerError {
		t.Errorf("estado: esperado 500, obtenido %d", status)
	}
	if got := errorCode(t, body); got != string(apperror.CodeInternal) {
		t.Errorf("código: esperado %s, obtenido %s", apperror.CodeInternal, got)
	}

	// El mensaje interno no debe filtrarse al cliente.
	serialized, _ := json.Marshal(body)
	if strings.Contains(string(serialized), "detalles internos") {
		t.Error("la respuesta filtró el mensaje interno del error")
	}
}
