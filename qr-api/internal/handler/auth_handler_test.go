package handler_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"

	"github.com/gianfrancocespedes/fact-qr/qr-api/internal/apperror"
	"github.com/gianfrancocespedes/fact-qr/qr-api/internal/auth"
)

const qrEndpoint = "/api/v1/qr"
const validMatrix = `{"matrix":[[1,2],[3,4]]}`

func TestPostTokenIsPublic(t *testing.T) {
	application := newTestApp(&stubStats{})

	// Sin cabecera de autorización: el endpoint emisor es la puerta de entrada a la API.
	status, body := post(t, application, "/api/v1/auth/token", `{"subject":"frontend"}`, nil)

	if status != http.StatusOK {
		t.Fatalf("estado: esperado 200, obtenido %d (%v)", status, body)
	}

	token, ok := body["token"].(string)
	if !ok || token == "" {
		t.Fatal("la respuesta debe incluir un token no vacío")
	}
	if body["tokenType"] != "Bearer" {
		t.Errorf("tokenType: esperado Bearer, obtenido %v", body["tokenType"])
	}
	if _, present := body["expiresAt"]; !present {
		t.Error("la respuesta debe informar cuándo expira el token")
	}

	// El token emitido debe ser utilizable de inmediato.
	if _, err := auth.Verify(token); err != nil {
		t.Errorf("el token emitido no es válido: %v", err)
	}
}

// TestPostTokenAcceptsEmptyBody comprueba que el sujeto sea opcional: pedir un token no
// debería exigir conocer el formato del cuerpo.
func TestPostTokenAcceptsEmptyBody(t *testing.T) {
	application := newTestApp(&stubStats{})

	status, body := post(t, application, "/api/v1/auth/token", ``, nil)

	if status != http.StatusOK {
		t.Fatalf("estado: esperado 200, obtenido %d (%v)", status, body)
	}
	if token, ok := body["token"].(string); !ok || token == "" {
		t.Error("debe emitirse un token aunque el cuerpo esté vacío")
	}
}

func TestQREndpointRequiresToken(t *testing.T) {
	application := newTestApp(&stubStats{})

	status, body := post(t, application, qrEndpoint, validMatrix, nil)

	if status != http.StatusUnauthorized {
		t.Fatalf("estado: esperado 401, obtenido %d", status)
	}
	if got := errorCode(t, body); got != string(apperror.CodeUnauthorized) {
		t.Errorf("código: esperado %s, obtenido %s", apperror.CodeUnauthorized, got)
	}
}

func TestQREndpointRejectsInvalidTokens(t *testing.T) {
	cases := map[string]string{
		"token inventado":    "Bearer no-es-un-token",
		"esquema incorrecto": "Basic dXNlcjpwYXNz",
		"sin esquema":        "solo-el-token",
		"esquema sin valor":  "Bearer ",
		"cabecera vacía":     "",
	}

	application := newTestApp(&stubStats{})

	for name, header := range cases {
		t.Run(name, func(t *testing.T) {
			status, body := post(t, application, qrEndpoint, validMatrix,
				map[string]string{fiber.HeaderAuthorization: header})

			if status != http.StatusUnauthorized {
				t.Fatalf("estado: esperado 401, obtenido %d", status)
			}
			if got := errorCode(t, body); got != string(apperror.CodeUnauthorized) {
				t.Errorf("código: esperado %s, obtenido %s", apperror.CodeUnauthorized, got)
			}
		})
	}
}

// TestQREndpointReportsExpiredToken comprueba que la expiración se distinga: el frontend
// pide un token nuevo en ese caso, en lugar de mostrar un error de credenciales.
func TestQREndpointReportsExpiredToken(t *testing.T) {
	expired := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   "frontend",
		Issuer:    "qr-api",
		IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
	})

	signed, err := expired.SignedString(auth.Secret())
	if err != nil {
		t.Fatalf("no se pudo firmar el token de prueba: %v", err)
	}

	application := newTestApp(&stubStats{})
	status, body := post(t, application, qrEndpoint, validMatrix,
		map[string]string{fiber.HeaderAuthorization: "Bearer " + signed})

	if status != http.StatusUnauthorized {
		t.Fatalf("estado: esperado 401, obtenido %d", status)
	}
	if got := errorCode(t, body); got != string(apperror.CodeTokenExpired) {
		t.Errorf("código: esperado %s, obtenido %s", apperror.CodeTokenExpired, got)
	}
}

// TestHealthStaysPublicWithAuth confirma que el chequeo de salud no exige token: las
// plataformas de despliegue lo consultan sin credenciales.
func TestHealthStaysPublicWithAuth(t *testing.T) {
	application := newTestApp(&stubStats{})

	response, err := application.Test(
		mustRequest(t, http.MethodGet, "/health"), 10_000,
	)
	if err != nil {
		t.Fatalf("no se pudo ejecutar la petición: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Errorf("estado: esperado 200, obtenido %d", response.StatusCode)
	}
}

// TestTokenFlowEndToEnd recorre el camino que seguirá el frontend: pedir token, usarlo y
// obtener el resultado.
func TestTokenFlowEndToEnd(t *testing.T) {
	application := newTestApp(&stubStats{})

	_, tokenBody := post(t, application, "/api/v1/auth/token", `{"subject":"frontend"}`, nil)
	token := tokenBody["token"].(string)

	status, body := post(t, application, qrEndpoint, validMatrix,
		map[string]string{fiber.HeaderAuthorization: "Bearer " + token})

	if status != http.StatusOK {
		t.Fatalf("estado: esperado 200, obtenido %d (%v)", status, body)
	}
	if _, present := body["factorization"]; !present {
		t.Error("la respuesta debe incluir la factorización")
	}
}

// TestAuthenticatedTokenReachesStatsService verifica la propagación: la llamada interna
// viaja con la misma identidad que autenticó al cliente original.
func TestAuthenticatedTokenReachesStatsService(t *testing.T) {
	stats := &stubStats{}
	application := newTestApp(stats)

	_, tokenBody := post(t, application, "/api/v1/auth/token", `{"subject":"frontend"}`, nil)
	token := tokenBody["token"].(string)

	post(t, application, qrEndpoint, validMatrix,
		map[string]string{fiber.HeaderAuthorization: "Bearer " + token})

	if stats.receivedToken != token {
		t.Error("el token del cliente debe propagarse a la stats-api sin alterarse")
	}
}
