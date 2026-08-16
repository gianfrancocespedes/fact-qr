import jwt from 'jsonwebtoken';

import { TEST_ISSUER as ISSUER, TEST_JWT_SECRET as SECRET } from '../constants.js';

/**
 * Emite tokens equivalentes a los de la qr-api para las pruebas.
 *
 * Replica deliberadamente el emisor y el algoritmo del servicio real: si esta API acepta
 * un token con otro `issuer` o firmado con otro algoritmo, la prueba debe fallar.
 *
 * Firma con la misma constante que setup.js instala en el entorno, en vez de leer
 * process.env: así la dependencia entre ambos archivos es explícita y no se rompe en
 * silencio si el orden de carga cambia.
 */

/** Token válido, listo para usar en la cabecera Authorization. */
export function validToken({ subject = 'test', expiresIn = '1h' } = {}) {
    return jwt.sign({}, SECRET, { subject, issuer: ISSUER, expiresIn, algorithm: 'HS256' });
}

/** Token ya vencido, para comprobar que se distingue de uno inválido. */
export function expiredToken() {
    return jwt.sign(
        { iat: Math.floor(Date.now() / 1000) - 7200 },
        SECRET,
        { subject: 'test', issuer: ISSUER, expiresIn: '-1h', algorithm: 'HS256' },
    );
}

/** Token firmado con otra clave: simula una falsificación. */
export function tokenWithWrongSecret() {
    return jwt.sign({}, 'secreto-equivocado', {
        subject: 'atacante',
        issuer: ISSUER,
        expiresIn: '1h',
        algorithm: 'HS256',
    });
}

/** Token emitido por otro servicio con el secreto correcto. */
export function tokenFromForeignIssuer() {
    return jwt.sign({}, SECRET, {
        subject: 'test',
        issuer: 'otro-servicio',
        expiresIn: '1h',
        algorithm: 'HS256',
    });
}

/** Cabecera de autorización lista para Supertest. */
export function authHeader(token = validToken()) {
    return ['Authorization', `Bearer ${token}`];
}
