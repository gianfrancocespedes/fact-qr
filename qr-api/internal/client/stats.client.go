package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gianfrancocespedes/fact-qr/qr-api/internal/apperror"
	"github.com/gianfrancocespedes/fact-qr/qr-api/internal/matrix"
)

const DefaultTimeout = 60 * time.Second

const statisticsPath = "/api/v1/statistics"

// Diagonal reporta el resultado de la verificación por matriz y de forma agregada.
type Diagonal struct {
	ByMatrix    []bool `json:"byMatrix"`
	AnyDiagonal bool   `json:"anyDiagonal"`
}

// Statistics es la respuesta de la stats-api.
type Statistics struct {
	Max      float64  `json:"max"`
	Min      float64  `json:"min"`
	Average  float64  `json:"average"`
	Sum      float64  `json:"sum"`
	Count    int      `json:"count"`
	Diagonal Diagonal `json:"diagonal"`
}

// statisticsRequest es el cuerpo que espera la stats-api.
type statisticsRequest struct {
	Matrices []matrix.Matrix `json:"matrices"`
}

// errorResponse captura el formato de error del contrato compartido para poder propagar
// el código original en lugar de enmascararlo tras un error genérico.
type errorResponse struct {
	Error struct {
		Code    string         `json:"code"`
		Details map[string]any `json:"details"`
	} `json:"error"`
}

// StatsClient consulta las estadísticas en la API de Node.
type StatsClient struct {
	baseURL string
	http    *http.Client
}

// New construye un cliente apuntando a baseURL.
func New(baseURL string, timeout time.Duration) *StatsClient {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	return &StatsClient{
		baseURL: baseURL,
		http:    &http.Client{Timeout: timeout},
	}
}

// Calculate envía las matrices a la stats-api y devuelve las estadísticas calculadas.
func (c *StatsClient) Calculate(ctx context.Context, matrices []matrix.Matrix, token string) (*Statistics, error) {
	payload, err := json.Marshal(statisticsRequest{Matrices: matrices})
	if err != nil {
		return nil, apperror.Wrap(apperror.CodeInternal, err, nil)
	}

	request, err := http.NewRequestWithContext(
		ctx, http.MethodPost, c.baseURL+statisticsPath, bytes.NewReader(payload),
	)
	if err != nil {
		return nil, apperror.Wrap(apperror.CodeInternal, err, nil)
	}

	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}

	response, err := c.http.Do(request)
	if err != nil {
		// Servicio caído, DNS fallido o timeout agotado: desde la perspectiva del cliente
		// todos significan lo mismo, la dependencia no está disponible.
		return nil, apperror.Wrap(apperror.CodeStatsServiceUnavailable, err, nil)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, c.translateError(response)
	}

	var statistics Statistics
	if err := json.NewDecoder(response.Body).Decode(&statistics); err != nil {
		return nil, apperror.Wrap(apperror.CodeStatsServiceUnavailable, err, nil)
	}

	return &statistics, nil
}

// translateError convierte una respuesta de error de la stats-api en un error de dominio.
func (c *StatsClient) translateError(response *http.Response) error {
	var body errorResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil || body.Error.Code == "" {
		return apperror.New(apperror.CodeStatsServiceUnavailable, apperror.Details{
			"status": response.StatusCode,
		})
	}

	return apperror.New(apperror.Code(body.Error.Code), body.Error.Details)
}

// HealthCheck verifica que la stats-api responda.
func (c *StatsClient) HealthCheck(ctx context.Context) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
	if err != nil {
		return fmt.Errorf("construyendo la petición de salud: %w", err)
	}

	response, err := c.http.Do(request)
	if err != nil {
		return fmt.Errorf("consultando la salud de stats-api: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("stats-api respondió %d en /health", response.StatusCode)
	}
	return nil
}
