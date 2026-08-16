import { AppError, ErrorCode } from '../errors/codes.js';
import { calculateStatistics, DEFAULT_EPSILON } from '../services/statistics.service.js';

// Lee la tolerancia desde el entorno.
function resolveEpsilon() {
    const configured = Number.parseFloat(process.env.DIAGONAL_EPSILON ?? '');
    return Number.isFinite(configured) && configured > 0 ? configured : DEFAULT_EPSILON;
}

/**
 * POST /api/v1/statistics
 *
 * Acepta `{ matrices: [[[...]]] }`. Los errores del dominio se delegan al middleware de
 * manejo de errores mediante next().
 */
export function postStatistics(request, response, next) {
    try {
        const { matrices } = request.body ?? {};

        if (matrices === undefined) {
            throw new AppError(ErrorCode.INVALID_PAYLOAD, { field: 'matrices' });
        }

        const statistics = calculateStatistics(matrices, resolveEpsilon());
        response.json(statistics);
    } catch (error) {
        next(error);
    }
}
