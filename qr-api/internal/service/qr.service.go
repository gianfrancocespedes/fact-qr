package service

import (
	"context"

	"github.com/gianfrancocespedes/fact-qr/qr-api/internal/client"
	"github.com/gianfrancocespedes/fact-qr/qr-api/internal/matrix"
)

// Dimensions describe el tamaño efectivo de la matriz procesada. Viaja en la respuesta
// porque rotar una matriz rectangular intercambia sus dimensiones y el frontend necesita
// saber cuántas filas y columnas dibujar.
type Dimensions struct {
	Rows    int `json:"rows"`
	Columns int `json:"columns"`
}

// Input refleja lo que se recibió y la transformación aplicada. Existe para trazabilidad:
// permite verificar la rotación de un vistazo sin recalcularla.
type Input struct {
	Original   matrix.Matrix    `json:"original"`
	Rotated    matrix.Matrix    `json:"rotated"`
	Rotations  int              `json:"rotations"`
	Direction  matrix.Direction `json:"direction"`
	Dimensions Dimensions       `json:"dimensions"`
}

// Factorization transporta las matrices resultantes de la descomposición.
type Factorization struct {
	Q matrix.Matrix `json:"q"`
	R matrix.Matrix `json:"r"`
}

// Result es la respuesta compuesta que devuelve la API.
type Result struct {
	Input         Input              `json:"input"`
	Factorization Factorization      `json:"factorization"`
	Statistics    *client.Statistics `json:"statistics"`
}

// Request son los parámetros ya validados del caso de uso.
type Request struct {
	Matrix    matrix.Matrix
	Rotations int
	Direction matrix.Direction
	Token     string
}

// StatsCalculator abstrae la dependencia con la stats-api.
//
// Es una interfaz y no el tipo concreto para poder sustituirla en las pruebas por un
// doble, sin levantar el servicio de Node.
type StatsCalculator interface {
	Calculate(ctx context.Context, matrices []matrix.Matrix, token string) (*client.Statistics, error)
}

// QRService implementa el flujo principal: esta API recibe la matriz, la procesa y envía
// los datos resultantes a la segunda API.
type QRService struct {
	stats StatsCalculator
}

// NewQRService construye el servicio con su dependencia.
func NewQRService(stats StatsCalculator) *QRService {
	return &QRService{stats: stats}
}

// Execute rota la matriz, la factoriza y delega el cálculo de estadísticas.
func (s *QRService) Execute(ctx context.Context, request Request) (*Result, error) {
	rotated := request.Matrix.Rotate(request.Rotations, request.Direction)
	factorization := rotated.Decompose()

	// Las estadísticas se calculan solo sobre Q y R, que son las matrices que produce la
	// factorización: la matriz de entrada viaja en la respuesta para trazabilidad, no como
	// dato de cálculo.
	statistics, err := s.stats.Calculate(
		ctx,
		[]matrix.Matrix{factorization.Q, factorization.R},
		request.Token,
	)
	if err != nil {
		return nil, err
	}

	return &Result{
		Input: Input{
			Original:  request.Matrix,
			Rotated:   rotated,
			Rotations: request.Rotations,
			Direction: request.Direction,
			Dimensions: Dimensions{
				Rows:    rotated.Rows(),
				Columns: rotated.Columns(),
			},
		},
		Factorization: Factorization{
			Q: factorization.Q,
			R: factorization.R,
		},
		Statistics: statistics,
	}, nil
}
