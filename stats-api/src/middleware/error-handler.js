import { AppError, ErrorCode } from '../errors/codes.js';

/**
 * Punto único de serialización de errores.
 *
 * Centralizarlo garantiza que toda respuesta de error tenga la misma forma
 * `{ error: { code, details } }`
 */
export function errorHandler(error, request, response, next) {
    if (error instanceof AppError) {
        response.status(error.status).json(error.toJSON());
        return;
    }

    // Un JSON malformado lo rechaza el parser de Express antes de llegar al controlador.
    if (error instanceof SyntaxError && 'body' in error) {
        response.status(400).json({ error: { code: ErrorCode.INVALID_PAYLOAD, details: { reason: 'malformed_json' } } });
        return;
    }

    // Cualquier otro error es un fallo no previsto.
    console.error('[stats-api] error no controlado:', error);
    response.status(500).json({ error: { code: ErrorCode.INTERNAL } });
}

/** Responde 404 con el mismo formato de error que el resto de la API. */
export function notFoundHandler(request, response) {
    response.status(404).json({
        error: { code: ErrorCode.INVALID_PAYLOAD, details: { path: request.path } },
    });
}
