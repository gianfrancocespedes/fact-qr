import jwt from 'jsonwebtoken';
import request from 'supertest';
import { describe, expect, it } from 'vitest';

import { createApp } from '../src/app.js';
import { ErrorCode } from '../src/errors/codes.js';
import { TEST_JWT_SECRET } from './constants.js';
import {
    authHeader,
    expiredToken,
    tokenFromForeignIssuer,
    tokenWithWrongSecret,
    validToken,
} from './helpers/token.js';

const app = createApp();
const ENDPOINT = '/api/v1/statistics';
const VALID_BODY = { matrices: [[[1, 2], [3, 4]]] };

describe('protección de los endpoints de negocio', () => {
    it('acepta una petición con token válido', async () => {
        const response = await request(app)
            .post(ENDPOINT)
            .set(...authHeader(validToken()))
            .send(VALID_BODY);

        expect(response.status).toBe(200);
    });

    it('rechaza una petición sin token', async () => {
        const response = await request(app).post(ENDPOINT).send(VALID_BODY);

        expect(response.status).toBe(401);
        expect(response.body.error).toMatchObject({
            code: ErrorCode.UNAUTHORIZED,
            details: { reason: 'missing_token' },
        });
    });

    it.each([
        ['esquema incorrecto', 'Basic dXNlcjpwYXNz'],
        ['sin esquema', 'solo-el-token'],
        ['cabecera vacía', ''],
    ])('rechaza %s', async (_label, header) => {
        const response = await request(app)
            .post(ENDPOINT)
            .set('Authorization', header)
            .send(VALID_BODY);

        expect(response.status).toBe(401);
        expect(response.body.error.code).toBe(ErrorCode.UNAUTHORIZED);
    });

    it('rechaza un token inventado', async () => {
        const response = await request(app)
            .post(ENDPOINT)
            .set('Authorization', 'Bearer no-es-un-token')
            .send(VALID_BODY);

        expect(response.status).toBe(401);
        expect(response.body.error).toMatchObject({
            code: ErrorCode.UNAUTHORIZED,
            details: { reason: 'invalid_token' },
        });
    });

    it('distingue el token expirado del inválido', async () => {
        // El frontend actúa distinto en cada caso: ante expiración pide uno nuevo, ante firma
        // inválida muestra un error de autenticación.
        const response = await request(app)
            .post(ENDPOINT)
            .set(...authHeader(expiredToken()))
            .send(VALID_BODY);

        expect(response.status).toBe(401);
        expect(response.body.error.code).toBe(ErrorCode.TOKEN_EXPIRED);
    });

    it('rechaza un token firmado con otro secreto', async () => {
        // Confirma que el secreto realmente protege: sin él no se pueden falsificar tokens.
        const response = await request(app)
            .post(ENDPOINT)
            .set(...authHeader(tokenWithWrongSecret()))
            .send(VALID_BODY);

        expect(response.status).toBe(401);
        expect(response.body.error.code).toBe(ErrorCode.UNAUTHORIZED);
    });

    it('rechaza un token de otro emisor', async () => {
        // Aunque la firma sea correcta, un token emitido por otro servicio no sirve aquí.
        const response = await request(app)
            .post(ENDPOINT)
            .set(...authHeader(tokenFromForeignIssuer()))
            .send(VALID_BODY);

        expect(response.status).toBe(401);
        expect(response.body.error.code).toBe(ErrorCode.UNAUTHORIZED);
    });

    it('rechaza un token con algoritmo "none"', async () => {
        // Vulnerabilidad clásica de JWT: sin restringir el algoritmo, un token sin firma
        // sería aceptado como válido.
        const unsigned = jwt.sign(
            { sub: 'atacante', iss: 'qr-api' },
            '',
            { algorithm: 'none' },
        );

        const response = await request(app)
            .post(ENDPOINT)
            .set(...authHeader(unsigned))
            .send(VALID_BODY);

        expect(response.status).toBe(401);
        expect(response.body.error.code).toBe(ErrorCode.UNAUTHORIZED);
    });

    it('valida el token antes que el cuerpo de la petición', async () => {
        // Un cuerpo inválido no debe revelar información a quien no está autenticado.
        const response = await request(app).post(ENDPOINT).send({ matrices: 'basura' });

        expect(response.status).toBe(401);
        expect(response.body.error.code).toBe(ErrorCode.UNAUTHORIZED);
    });
});

describe('secreto compartido entre servicios', () => {
    it('acepta el token emitido por la qr-api', async () => {
        // Este es el caso de la llamada interna: la qr-api propaga el token del cliente
        // original, y esta API lo valida con el mismo secreto HS256.
        const tokenDeLaQrApi = jwt.sign({}, TEST_JWT_SECRET, {
            subject: 'frontend',
            issuer: 'qr-api',
            expiresIn: '1h',
            algorithm: 'HS256',
        });

        const response = await request(app)
            .post(ENDPOINT)
            .set(...authHeader(tokenDeLaQrApi))
            .send(VALID_BODY);

        expect(response.status).toBe(200);
    });
});
