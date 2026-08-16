/**
 * Cliente tipado hacia ambas APIs.
 *
 * Concentra tres responsabilidades que no deberían repartirse por los componentes:
 * obtener y renovar el token, traducir las respuestas de error del contrato, y aplicar el
 * reintento cuando el token caduca.
 */
import { translateError } from '@/i18n/errors';
import type { ApiErrorBody, Matrix, QRRequest, QRResult, Statistics } from '@/types/api';

const QR_API_URL = import.meta.env.VITE_QR_API_URL ?? 'http://localhost:8080';
const STATS_API_URL = import.meta.env.VITE_STATS_API_URL ?? 'http://localhost:3000';

/**
 * Error ya traducido al español, listo para mostrarse.
 *
 * Conserva el código original para que los componentes puedan reaccionar de forma distinta
 * a casos concretos sin depender del texto.
 */
export class ApiError extends Error {
    readonly code: string;
    readonly details?: Record<string, unknown>;

    constructor(code: string, message: string, details?: Record<string, unknown>) {
        super(message);
        this.name = 'ApiError';
        this.code = code;
        this.details = details;
    }
}

/**
 * El token vive en memoria y no en localStorage: dura una hora y se puede volver a pedir,
 * así que persistirlo no aporta nada y localStorage es accesible desde cualquier script
 * de la página.
 */
let cachedToken: string | null = null;

/** Solicita un token nuevo al emisor y lo guarda. */
async function fetchToken(): Promise<string> {
    const response = await fetch(`${QR_API_URL}/api/v1/auth/token`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ subject: 'web' }),
    });

    if (!response.ok) {
        throw new ApiError('ERROR_UNAUTHORIZED', translateError('ERROR_UNAUTHORIZED'));
    }

    const { token } = (await response.json()) as { token: string };
    cachedToken = token;
    return token;
}

/** Devuelve el token vigente, pidiéndolo si aún no hay uno. */
async function currentToken(): Promise<string> {
    return cachedToken ?? fetchToken();
}

/** Convierte una respuesta fallida en un ApiError con el mensaje ya en español. */
async function toApiError(response: Response): Promise<ApiError> {
    try {
        const body = (await response.json()) as ApiErrorBody;
        const { code, details } = body.error;
        return new ApiError(code, translateError(code, details), details);
    } catch {
        // La respuesta no siguió el contrato: un proxy intermedio, un HTML de error...
        return new ApiError('ERROR_INTERNAL', translateError('ERROR_INTERNAL'));
    }
}

/**
 * Envía una petición autenticada, renovando el token si caducó.
 *
 * El reintento se limita a uno: si el 401 no se debe a la expiración sino a un secreto mal
 * configurado entre servicios, insistir generaría cientos de peticiones sin resolver nada
 * y el usuario nunca vería el error real.
 */
async function authorizedPost<T>(url: string, payload: unknown): Promise<T> {
    const send = async (token: string): Promise<Response> =>
        fetch(url, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                Authorization: `Bearer ${token}`,
            },
            body: JSON.stringify(payload),
        });

    let response: Response;
    try {
        response = await send(await currentToken());

        if (response.status === 401) {
            cachedToken = null;
            response = await send(await fetchToken());
        }
    } catch (error) {
        // fetch solo rechaza ante fallos de red; los códigos de error HTTP llegan resueltos.
        if (error instanceof ApiError) {
            throw error;
        }
        throw new ApiError('ERROR_NETWORK', translateError('ERROR_NETWORK'));
    }

    if (!response.ok) {
        throw await toApiError(response);
    }

    return (await response.json()) as T;
}

/**
 * Factoriza una matriz. La qr-api rota, descompone y consulta las estadísticas a la
 * stats-api, devolviendo la respuesta compuesta.
 */
export function factorize(request: QRRequest): Promise<QRResult> {
    return authorizedPost<QRResult>(`${QR_API_URL}/api/v1/qr`, request);
}

/**
 * Calcula estadísticas de una matriz llamando directamente a la stats-api.
 *
 * Envía la matriz **tal como fue ingresada**, sin rotar: máximo, mínimo, promedio y suma
 * son invariantes ante rotación, así que rotarla antes sería trabajo sin efecto observable.
 */
export function calculateStatistics(matrix: Matrix): Promise<Statistics> {
    return authorizedPost<Statistics>(`${STATS_API_URL}/api/v1/statistics`, {
        matrices: [matrix],
    });
}

/** Descarta el token guardado. Se usa en las pruebas para partir de un estado limpio. */
export function resetToken(): void {
    cachedToken = null;
}
