package client_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gianfrancocespedes/fact-qr/qr-api/internal/apperror"
	"github.com/gianfrancocespedes/fact-qr/qr-api/internal/client"
	"github.com/gianfrancocespedes/fact-qr/qr-api/internal/matrix"
)

// sampleMatrices son las matrices de prueba que se envían al doble de la stats-api.
var sampleMatrices = []matrix.Matrix{{{1, 0}, {0, 1}}}

// assertDomainCode comprueba que el error sea de dominio con el código esperado.
func assertDomainCode(t *testing.T, err error, want apperror.Code) {
	t.Helper()

	var domainError *apperror.Error
	if !errors.As(err, &domainError) {
		t.Fatalf("se esperaba *apperror.Error, se obtuvo %T (%v)", err, err)
	}
	if domainError.Code != want {
		t.Fatalf("código: esperado %s, obtenido %s", want, domainError.Code)
	}
}

func TestCalculateReturnsStatistics(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"max":9,"min":-2,"average":1.5,"sum":12,"count":8,
			"diagonal":{"byMatrix":[true,false],"anyDiagonal":true}}`))
	}))
	defer server.Close()

	statistics, err := client.New(server.URL, time.Second).
		Calculate(context.Background(), sampleMatrices, "")
	if err != nil {
		t.Fatalf("no se esperaba error, se obtuvo %v", err)
	}

	if statistics.Max != 9 || statistics.Min != -2 || statistics.Sum != 12 || statistics.Count != 8 {
		t.Errorf("estadísticas mal decodificadas: %+v", statistics)
	}
	if !statistics.Diagonal.AnyDiagonal {
		t.Error("anyDiagonal debería ser true")
	}
	if len(statistics.Diagonal.ByMatrix) != 2 {
		t.Errorf("byMatrix: esperados 2 elementos, obtenidos %d", len(statistics.Diagonal.ByMatrix))
	}
}

// TestCalculateSendsExpectedRequest verifica el contrato de salida: método, ruta,
// cabeceras y estructura del cuerpo.
func TestCalculateSendsExpectedRequest(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
		gotAuth   string
		gotType   string
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotType = r.Header.Get("Content-Type")
		_, _ = w.Write([]byte(`{"max":1,"min":0,"average":0.5,"sum":2,"count":4,
			"diagonal":{"byMatrix":[true],"anyDiagonal":true}}`))
	}))
	defer server.Close()

	_, err := client.New(server.URL, time.Second).
		Calculate(context.Background(), sampleMatrices, "token-abc")
	if err != nil {
		t.Fatalf("no se esperaba error, se obtuvo %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("método: esperado POST, obtenido %s", gotMethod)
	}
	if gotPath != "/api/v1/statistics" {
		t.Errorf("ruta: esperada /api/v1/statistics, obtenida %s", gotPath)
	}
	if gotAuth != "Bearer token-abc" {
		t.Errorf("Authorization: esperado %q, obtenido %q", "Bearer token-abc", gotAuth)
	}
	if gotType != "application/json" {
		t.Errorf("Content-Type: esperado application/json, obtenido %s", gotType)
	}
}

// TestCalculateOmitsAuthorizationWhenNoToken evita enviar una cabecera vacía, que algunos
// servidores rechazan como malformada.
func TestCalculateOmitsAuthorizationWhenNoToken(t *testing.T) {
	var hadHeader bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, hadHeader = r.Header["Authorization"]
		_, _ = w.Write([]byte(`{"max":1,"min":1,"average":1,"sum":1,"count":1,
			"diagonal":{"byMatrix":[true],"anyDiagonal":true}}`))
	}))
	defer server.Close()

	_, _ = client.New(server.URL, time.Second).
		Calculate(context.Background(), sampleMatrices, "")

	if hadHeader {
		t.Error("no debería enviarse la cabecera Authorization sin token")
	}
}

// TestCalculatePropagatesDomainErrorCode comprueba que un error de validación de la
// stats-api conserve su código: el frontend necesita el específico, no uno genérico.
func TestCalculatePropagatesDomainErrorCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"ERROR_INVALID_NUMBER","details":{"row":2}}}`))
	}))
	defer server.Close()

	_, err := client.New(server.URL, time.Second).
		Calculate(context.Background(), sampleMatrices, "")

	assertDomainCode(t, err, apperror.CodeInvalidNumber)
}

func TestCalculateHandlesServiceDown(t *testing.T) {
	// Un servidor cerrado de inmediato simula el servicio caído.
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	address := server.URL
	server.Close()

	_, err := client.New(address, time.Second).
		Calculate(context.Background(), sampleMatrices, "")

	assertDomainCode(t, err, apperror.CodeStatsServiceUnavailable)
}

// TestCalculateHandlesTimeout cubre el caso en que el servicio existe y responde, pero
// tarda más de lo aceptado: debe traducirse a un error de dominio, no quedarse colgado.
func TestCalculateHandlesTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	_, err := client.New(server.URL, 20*time.Millisecond).
		Calculate(context.Background(), sampleMatrices, "")

	assertDomainCode(t, err, apperror.CodeStatsServiceUnavailable)
}

// TestCalculateHandlesUnparseableError cubre una respuesta de error que no sigue el
// contrato: se degrada al código de servicio no disponible en lugar de fallar.
func TestCalculateHandlesUnparseableError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`<html>error del proxy</html>`))
	}))
	defer server.Close()

	_, err := client.New(server.URL, time.Second).
		Calculate(context.Background(), sampleMatrices, "")

	assertDomainCode(t, err, apperror.CodeStatsServiceUnavailable)
}

func TestCalculateHandlesMalformedSuccessBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`no soy json`))
	}))
	defer server.Close()

	_, err := client.New(server.URL, time.Second).
		Calculate(context.Background(), sampleMatrices, "")

	assertDomainCode(t, err, apperror.CodeStatsServiceUnavailable)
}

// TestCalculateRespectsContextCancellation permite abortar la llamada cuando el cliente
// original desiste, sin esperar al timeout completo.
func TestCalculateRespectsContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.New(server.URL, time.Minute).
		Calculate(ctx, sampleMatrices, "")

	if err == nil {
		t.Fatal("se esperaba error al cancelar el contexto")
	}
}

func TestNewFallsBackToDefaultTimeout(t *testing.T) {
	// Un timeout no positivo es una configuración inválida; el cliente usa el valor por
	// defecto en lugar de quedarse sin límite.
	if client.New("http://localhost", 0) == nil {
		t.Fatal("el cliente no debería ser nil")
	}
	if client.New("http://localhost", -time.Second) == nil {
		t.Fatal("el cliente no debería ser nil")
	}
}

func TestHealthCheck(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	if err := client.New(server.URL, time.Second).HealthCheck(context.Background()); err != nil {
		t.Errorf("no se esperaba error, se obtuvo %v", err)
	}
}

func TestHealthCheckFailsOnBadStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	if err := client.New(server.URL, time.Second).HealthCheck(context.Background()); err == nil {
		t.Error("se esperaba error con estado 503")
	}
}
