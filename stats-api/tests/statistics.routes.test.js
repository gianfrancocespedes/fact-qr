import request from 'supertest';
import { beforeEach, describe, expect, it } from 'vitest';

import { createApp } from '../src/app.js';
import { ErrorCode } from '../src/errors/codes.js';
import { authHeader } from './helpers/token.js';

const app = createApp();
const ENDPOINT = '/api/v1/statistics';

/** Petición autenticada al endpoint de estadísticas. */
function postStatistics(body) {
    return request(app).post(ENDPOINT).set(...authHeader()).send(body);
}

describe('GET /health', () => {
    it('responde sin autenticación', async () => {
        // El chequeo de salud queda público: exigir token lo volvería inútil para las
        // plataformas de despliegue, que lo consultan sin credenciales.
        const response = await request(app).get('/health');

        expect(response.status).toBe(200);
        expect(response.body).toMatchObject({ status: 'ok', service: 'stats-api' });
    });
});

describe(`POST ${ENDPOINT}`, () => {
    it('devuelve las estadísticas de las matrices recibidas', async () => {
        const response = await postStatistics({ matrices: [[[1, 2], [3, 4]]] });

        expect(response.status).toBe(200);
        expect(response.body).toMatchObject({
            max: 4,
            min: 1,
            sum: 10,
            average: 2.5,
            count: 4,
        });
        expect(response.body.diagonal).toMatchObject({
            byMatrix: [false],
            anyDiagonal: false,
        });
    });

    it('procesa el caso real: Q y R de una factorización', async () => {
        // Valores tomados de la salida de la qr-api para la matriz [[12,-51],[6,167],[-4,24]].
        const q = [
            [-0.857142857142857, 0.394285714285714, 0.331428571428571],
            [-0.428571428571428, -0.902857142857143, -0.034285714285714],
            [0.285714285714286, -0.171428571428571, 0.942857142857143],
        ];
        const r = [[-14, -21], [0, -175], [0, 0]];

        const response = await postStatistics({ matrices: [q, r] });

        expect(response.status).toBe(200);
        expect(response.body.max).toBeCloseTo(0.942857142857143, 10);
        expect(response.body.min).toBe(-175);
        expect(response.body.count).toBe(15);
        expect(response.body.diagonal.anyDiagonal).toBe(false);
    });

    it('detecta una matriz diagonal entre varias', async () => {
        const response = await postStatistics({
            matrices: [[[1, 2], [3, 4]], [[5, 0], [0, 6]]],
        });

        expect(response.status).toBe(200);
        expect(response.body.diagonal).toMatchObject({
            byMatrix: [false, true],
            anyDiagonal: true,
        });
    });

    it('rechaza un cuerpo sin el campo matrices', async () => {
        const response = await postStatistics({});

        expect(response.status).toBe(400);
        expect(response.body.error).toMatchObject({
            code: ErrorCode.INVALID_PAYLOAD,
            details: { field: 'matrices' },
        });
    });

    it('rechaza celdas no finitas indicando la posición', async () => {
        // JSON no admite NaN literal, así que se envía como cadena cruda para simular un
        // cliente que serializa mal.
        const response = await request(app)
            .post(ENDPOINT)
            .set(...authHeader())
            .set('Content-Type', 'application/json')
            .send('{"matrices":[[[1,2],[3,"x"]]]}');

        expect(response.status).toBe(400);
        expect(response.body.error).toMatchObject({
            code: ErrorCode.INVALID_NUMBER,
            details: { matrix: 1, row: 2, column: 2 },
        });
    });

    it('rechaza filas de longitud desigual', async () => {
        const response = await postStatistics({ matrices: [[[1, 2, 3], [4, 5]]] });

        expect(response.status).toBe(400);
        expect(response.body.error.code).toBe(ErrorCode.INCONSISTENT_ROW_LENGTH);
    });

    it('rechaza JSON malformado con el código de error del contrato', async () => {
        const response = await request(app)
            .post(ENDPOINT)
            .set(...authHeader())
            .set('Content-Type', 'application/json')
            .send('{"matrices": [[[1,2]');

        expect(response.status).toBe(400);
        expect(response.body.error.code).toBe(ErrorCode.INVALID_PAYLOAD);
    });

    it('nunca devuelve texto legible en el error, solo el código', async () => {
        // El contrato es explícito: la traducción al español vive en el frontend.
        const response = await postStatistics({ matrices: [] });

        expect(response.body.error.code).toMatch(/^ERROR_[A-Z_]+$/);
        expect(response.body.error).not.toHaveProperty('message');
    });
});

describe('rutas desconocidas', () => {
    it('responde 404 con el formato de error del contrato', async () => {
        const response = await request(app).get('/api/v1/no-existe');

        expect(response.status).toBe(404);
        expect(response.body.error).toHaveProperty('code');
    });
});

describe('CORS', () => {
    it('expone las cabeceras que el navegador necesita', async () => {
        // El frontend llama a esta API directamente, así que sin CORS el botón de
        // estadísticas fallaría en producción.
        const response = await request(app)
            .post(ENDPOINT)
            .set(...authHeader())
            .set('Origin', 'http://localhost:5173')
            .send({ matrices: [[[1]]] });

        expect(response.headers['access-control-allow-origin']).toBeDefined();
    });
});

describe('tolerancia configurable por entorno', () => {
    beforeEach(() => {
        delete process.env.DIAGONAL_EPSILON;
    });

    it('usa DIAGONAL_EPSILON cuando está definida', async () => {
        process.env.DIAGONAL_EPSILON = '1e-2';

        const response = await postStatistics({ matrices: [[[1, 0.001], [0, 1]]] });

        expect(response.body.diagonal.anyDiagonal).toBe(true);
    });

    it('vuelve al valor por defecto si la variable no es válida', async () => {
        process.env.DIAGONAL_EPSILON = 'no-es-un-numero';

        const response = await postStatistics({ matrices: [[[1, 0.001], [0, 1]]] });

        expect(response.body.diagonal.anyDiagonal).toBe(false);
    });
});
