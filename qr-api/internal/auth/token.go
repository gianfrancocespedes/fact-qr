package auth

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/gianfrancocespedes/fact-qr/qr-api/internal/apperror"
)

const (
	defaultExpiration = time.Hour

	// issuer identifica al emisor del token. Se comprueba en la verificación para
	// evitar que un token emitido por otro servicio sea aceptado.
	issuer = "qr-api"
)

type Claims struct {
	jwt.RegisteredClaims
}

// Secret devuelve la clave de firma configurada.
func Secret() []byte {
	return []byte(os.Getenv("JWT_SECRET"))
}

// RequireSecret verifica que la clave de firma esté configurada.
func RequireSecret() error {
	if os.Getenv("JWT_SECRET") == "" {
		return errors.New("JWT_SECRET no está definida: copia .env.example a .env o configúrala en el entorno")
	}
	return nil
}

// expiration devuelve la vigencia configurada, o el valor por defecto
func expiration() time.Duration {
	raw := os.Getenv("JWT_EXPIRATION")
	if raw == "" {
		return defaultExpiration
	}

	parsed, err := time.ParseDuration(raw)
	if err != nil || parsed <= 0 {
		return defaultExpiration
	}
	return parsed
}

// Issue emite un token firmado para el sujeto indicado.
func Issue(subject string) (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(expiration())

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subject,
			Issuer:    issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	})

	signed, err := token.SignedString(Secret())
	if err != nil {
		return "", time.Time{}, apperror.Wrap(apperror.CodeInternal, err, nil)
	}
	return signed, expiresAt, nil
}

// Verify valida la firma y la vigencia de un token.
func Verify(raw string) (*Claims, error) {
	claims := &Claims{}

	_, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
		// Sin esta comprobación, un atacante podría firmar con "alg": "none" o forzar un
		// algoritmo asimétrico para que la clave pública se interprete como secreto HMAC.
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("algoritmo de firma inesperado")
		}
		return Secret(), nil
	}, jwt.WithIssuer(issuer), jwt.WithExpirationRequired())

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, apperror.Wrap(apperror.CodeTokenExpired, err, nil)
		}
		return nil, apperror.Wrap(apperror.CodeUnauthorized, err, nil)
	}

	return claims, nil
}
