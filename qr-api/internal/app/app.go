package app

import (
	"github.com/gofiber/fiber/v2"

	"github.com/gianfrancocespedes/fact-qr/qr-api/internal/handler"
	"github.com/gianfrancocespedes/fact-qr/qr-api/internal/middleware"
	"github.com/gianfrancocespedes/fact-qr/qr-api/internal/service"
)

// BodyLimit acota el cuerpo de la petición, holgado para la matriz máxima aceptada.
const BodyLimit = 1024 * 1024

// Config agrupa las dependencias que la aplicación necesita para construirse.
type Config struct {
	Stats service.StatsCalculator
}

// New construye la aplicación sin ponerla a escuchar.
//
// Separar la construcción del arranque permite que las pruebas de integración levanten la
// app en memoria con app.Test(), sin ocupar un puerto ni depender de tiempos de red.
func New(config Config) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: middleware.ErrorHandler,
		BodyLimit:    BodyLimit,
		// La familia del socket la fija este campo, no la dirección: con el defecto "tcp4",
		// pasar "[::]:8080" a Listen abre igualmente un socket IPv4. Se fuerza "tcp" para
		// obtener dual-stack, necesario porque la red privada de Railway es solo IPv6.
		Network:               fiber.NetworkTCP,
		DisableStartupMessage: true,
	})

	app.Use(middleware.NewCORS())

	// API de comprobación de salud (health check).
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "service": "qr-api"})
	})

	// API pública para obtener un token de autenticación. No requiere token previo.
	app.Post("/api/v1/auth/token", handler.NewAuthHandler().PostToken)

	// Todo lo que cuelga de este grupo exige un token válido.
	protected := app.Group("/api/v1", middleware.RequireAuth())

	qrHandler := handler.NewQRHandler(service.NewQRService(config.Stats))
	protected.Post("/qr", qrHandler.Post)

	return app
}
