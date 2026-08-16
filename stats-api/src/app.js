import express from 'express';

import { buildCors } from './middleware/cors.js';
import { errorHandler, notFoundHandler } from './middleware/error-handler.js';
import statisticsRoutes from './routes/statistics.routes.js';

// Límite del cuerpo de la petición
const BODY_LIMIT = '1mb';

/**
 * Se separa la construcción del arranque para que las pruebas de integración levanten
 * la app en memoria con Supertest, sin ocupar un puerto ni depender de tiempos de red.
 */
export function createApp() {
    const app = express();

    app.use(buildCors());
    app.use(express.json({ limit: BODY_LIMIT }));

    // API para monitoreo de la salud del servicio
    app.get('/health', (request, response) => {
        response.json({ status: 'ok', service: 'stats-api' });
    });

    // API para estadísticas
    app.use('/api/v1', statisticsRoutes);

    // Manejo de rutas no encontradas
    app.use(notFoundHandler);

    // Manejo de errores
    app.use(errorHandler);

    return app;
}
