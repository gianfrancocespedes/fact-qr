import jwt from 'jsonwebtoken';

import { AppError, ErrorCode } from '../errors/codes.js';

/**
 * Emisor esperado. Se valida para que un token firmado por otro servicio con el mismo
 * secreto no sea aceptado aquí.
 */
const EXPECTED_ISSUER = 'qr-api';

const BEARER_PREFIX = 'Bearer ';

/**
 * Lee el secreto en cada verificación para que las pruebas puedan cambiarlo sin reiniciar.
 *
 * No hay valor por defecto: debe coincidir con el de la qr-api, porque esta API verifica
 * tokens que aquella emite con la misma clave HS256.
 */
function secret() {
    return process.env.JWT_SECRET;
}

/**
 * Verifica que la clave de firma esté configurada.
 */
export function requireSecret() {
    if (!process.env.JWT_SECRET) {
        throw new Error(
            'JWT_SECRET no está definida: copia .env.example a .env o configúrala en el entorno',
        );
    }
}

/** Extrae el token de la cabecera Authorization. */
function extractToken(request) {
    const header = request.headers.authorization;

    if (!header) {
        throw new AppError(ErrorCode.UNAUTHORIZED, { reason: 'missing_token' }, 401);
    }
    if (!header.toLowerCase().startsWith(BEARER_PREFIX.toLowerCase())) {
        throw new AppError(ErrorCode.UNAUTHORIZED, { reason: 'invalid_scheme' }, 401);
    }

    return header.slice(BEARER_PREFIX.length);
}

/**
 * Rechaza las peticiones sin un token válido.
 */
export function requireAuth(request, response, next) {
    try {
        const token = extractToken(request);

        // Restringir el algoritmo y evitar un posible "alg": "none" de un atacante, la 
        // librería lo aceptaría sin verificar firma.
        const claims = jwt.verify(token, secret(), {
            algorithms: ['HS256'],
            issuer: EXPECTED_ISSUER,
        });

        request.subject = claims.sub;
        next();
    } catch (error) {
        if (error instanceof AppError) {
            next(error);
            return;
        }

        // Se distingue la expiración del token inválido porque el frontend actúa distinto:
        // ante expiración pide uno nuevo, ante firma inválida muestra un error de autenticación.
        if (error instanceof jwt.TokenExpiredError) {
            next(new AppError(ErrorCode.TOKEN_EXPIRED, { expiredAt: error.expiredAt }, 401));
            return;
        }

        next(new AppError(ErrorCode.UNAUTHORIZED, { reason: 'invalid_token' }, 401));
    }
}
