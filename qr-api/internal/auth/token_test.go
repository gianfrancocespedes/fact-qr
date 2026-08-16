package auth_test

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/gianfrancocespedes/fact-qr/qr-api/internal/apperror"
	"github.com/gianfrancocespedes/fact-qr/qr-api/internal/auth"
)

// assertCode comprueba que el error sea de dominio con el código esperado.
func assertCode(t *testing.T, err error, want apperror.Code) {
	t.Helper()

	var domainError *apperror.Error
	if !errors.As(err, &domainError) {
		t.Fatalf("se esperaba *apperror.Error, se obtuvo %T (%v)", err, err)
	}
	if domainError.Code != want {
		t.Fatalf("código: esperado %s, obtenido %s", want, domainError.Code)
	}
}

func TestIssueAndVerify(t *testing.T) {
	token, expiresAt, err := auth.Issue("frontend")
	if err != nil {
		t.Fatalf("no se esperaba error al emitir: %v", err)
	}
	if token == "" {
		t.Fatal("el token no debería estar vacío")
	}
	if !expiresAt.After(time.Now()) {
		t.Error("la expiración debería estar en el futuro")
	}

	claims, err := auth.Verify(token)
	if err != nil {
		t.Fatalf("no se esperaba error al verificar: %v", err)
	}
	if claims.Subject != "frontend" {
		t.Errorf("subject: esperado %q, obtenido %q", "frontend", claims.Subject)
	}
}

func TestVerifyRejectsTamperedToken(t *testing.T) {
	token, _, err := auth.Issue("frontend")
	if err != nil {
		t.Fatalf("no se esperaba error al emitir: %v", err)
	}

	// Alterar un carácter invalida la firma.
	tampered := token[:len(token)-2] + "xy"

	_, err = auth.Verify(tampered)
	assertCode(t, err, apperror.CodeUnauthorized)
}

func TestVerifyRejectsGarbage(t *testing.T) {
	for _, raw := range []string{"", "no-es-un-token", "a.b.c"} {
		_, err := auth.Verify(raw)
		assertCode(t, err, apperror.CodeUnauthorized)
	}
}

// TestVerifyRejectsExpiredToken comprueba que la expiración se distinga del token inválido:
// el frontend pide uno nuevo en un caso y muestra error de autenticación en el otro.
func TestVerifyRejectsExpiredToken(t *testing.T) {
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

	_, err = auth.Verify(signed)
	assertCode(t, err, apperror.CodeTokenExpired)
}

// TestVerifyRejectsNoneAlgorithm cubre una vulnerabilidad clásica de JWT: un token que
// declara "alg": "none" para saltarse la verificación de firma.
func TestVerifyRejectsNoneAlgorithm(t *testing.T) {
	unsigned := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.RegisteredClaims{
		Subject:   "atacante",
		Issuer:    "qr-api",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	})

	signed, err := unsigned.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("no se pudo construir el token de prueba: %v", err)
	}

	_, err = auth.Verify(signed)
	assertCode(t, err, apperror.CodeUnauthorized)
}

// TestVerifyRejectsForeignIssuer evita que un token emitido por otro servicio con el mismo
// secreto sea aceptado aquí.
func TestVerifyRejectsForeignIssuer(t *testing.T) {
	foreign := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   "frontend",
		Issuer:    "otro-servicio",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	})

	signed, err := foreign.SignedString(auth.Secret())
	if err != nil {
		t.Fatalf("no se pudo firmar el token de prueba: %v", err)
	}

	_, err = auth.Verify(signed)
	assertCode(t, err, apperror.CodeUnauthorized)
}

// TestVerifyRequiresExpiration rechaza tokens sin vencimiento, que serían válidos para
// siempre si se filtraran.
func TestVerifyRequiresExpiration(t *testing.T) {
	eternal := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject: "frontend",
		Issuer:  "qr-api",
	})

	signed, err := eternal.SignedString(auth.Secret())
	if err != nil {
		t.Fatalf("no se pudo firmar el token de prueba: %v", err)
	}

	_, err = auth.Verify(signed)
	assertCode(t, err, apperror.CodeUnauthorized)
}

// TestVerifyRejectsDifferentSecret confirma que el secreto realmente protege: un token
// firmado con otra clave no pasa.
func TestVerifyRejectsDifferentSecret(t *testing.T) {
	other := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   "atacante",
		Issuer:    "qr-api",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	})

	signed, err := other.SignedString([]byte("secreto-equivocado"))
	if err != nil {
		t.Fatalf("no se pudo firmar el token de prueba: %v", err)
	}

	_, err = auth.Verify(signed)
	assertCode(t, err, apperror.CodeUnauthorized)
}

// TestSecretPrefersEnvironment comprueba que el secreto de producción tenga prioridad
// sobre el de desarrollo.
func TestSecretPrefersEnvironment(t *testing.T) {
	t.Setenv("JWT_SECRET", "secreto-de-produccion")

	if got := string(auth.Secret()); got != "secreto-de-produccion" {
		t.Errorf("secreto: esperado el del entorno, obtenido %q", got)
	}
}

// TestIssueRespectsConfiguredExpiration verifica que la vigencia sea configurable.
func TestIssueRespectsConfiguredExpiration(t *testing.T) {
	t.Setenv("JWT_EXPIRATION", "15m")

	_, expiresAt, err := auth.Issue("frontend")
	if err != nil {
		t.Fatalf("no se esperaba error: %v", err)
	}

	remaining := time.Until(expiresAt)
	if remaining > 16*time.Minute || remaining < 14*time.Minute {
		t.Errorf("vigencia: esperada ~15m, obtenida %v", remaining)
	}
}

// TestIssueFallsBackOnInvalidExpiration comprueba que una configuración mal escrita no
// impida emitir tokens.
func TestIssueFallsBackOnInvalidExpiration(t *testing.T) {
	t.Setenv("JWT_EXPIRATION", "no-es-una-duracion")

	_, expiresAt, err := auth.Issue("frontend")
	if err != nil {
		t.Fatalf("no se esperaba error: %v", err)
	}

	if remaining := time.Until(expiresAt); remaining < 50*time.Minute {
		t.Errorf("debería usarse el valor por defecto (1h), se obtuvo %v", remaining)
	}
}
