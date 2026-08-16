import { describe, expect, it } from 'vitest';

import { AppError, ErrorCode } from '../src/errors/codes.js';
import { calculateStatistics } from '../src/services/statistics.service.js';

/** Comprueba que la función lance un AppError con el código esperado. */
function expectErrorCode(fn, code) {
    try {
        fn();
    } catch (error) {
        expect(error).toBeInstanceOf(AppError);
        expect(error.code).toBe(code);
        return error;
    }
    throw new Error(`se esperaba un error ${code}, pero no se lanzó ninguno`);
}

describe('calculateStatistics — agregados', () => {
    it('calcula máximo, mínimo, suma y promedio sobre una matriz', () => {
        const result = calculateStatistics([[[1, 2], [3, 4]]]);

        expect(result.max).toBe(4);
        expect(result.min).toBe(1);
        expect(result.sum).toBe(10);
        expect(result.average).toBe(2.5);
        expect(result.count).toBe(4);
    });

    it('agrega los valores de todas las matrices como un único conjunto', () => {
        // Las estadísticas son sobre "los datos de las matrices" en conjunto, no una por
        // matriz.
        const result = calculateStatistics([
            [[1, 2]],
            [[10, 20]],
        ]);

        expect(result.max).toBe(20);
        expect(result.min).toBe(1);
        expect(result.sum).toBe(33);
        expect(result.average).toBe(8.25);
        expect(result.count).toBe(4);
    });

    it('maneja valores negativos', () => {
        const result = calculateStatistics([[[-5, -1], [-3, -2]]]);

        expect(result.max).toBe(-1);
        expect(result.min).toBe(-5);
        expect(result.sum).toBe(-11);
        expect(result.average).toBe(-2.75);
    });

    it('maneja una matriz de un solo elemento', () => {
        const result = calculateStatistics([[[7]]]);

        expect(result.max).toBe(7);
        expect(result.min).toBe(7);
        expect(result.sum).toBe(7);
        expect(result.average).toBe(7);
        expect(result.count).toBe(1);
    });

    it('maneja valores decimales sin redondear', () => {
        // El dominio nunca redondea: eso es tarea de la capa de presentación.
        const result = calculateStatistics([[[0.1, 0.2]]]);

        expect(result.sum).toBeCloseTo(0.3, 10);
        expect(result.average).toBeCloseTo(0.15, 10);
    });

    it('acepta matrices rectangulares', () => {
        const result = calculateStatistics([[[1, 2, 3], [4, 5, 6]]]);

        expect(result.count).toBe(6);
        expect(result.sum).toBe(21);
    });
});

describe('calculateStatistics — verificación de matriz diagonal', () => {
    it('detecta una matriz diagonal', () => {
        const result = calculateStatistics([[[5, 0, 0], [0, -2, 0], [0, 0, 9]]]);

        expect(result.diagonal.byMatrix).toEqual([true]);
        expect(result.diagonal.anyDiagonal).toBe(true);
    });

    it('reconoce la identidad como diagonal', () => {
        const result = calculateStatistics([[[1, 0], [0, 1]]]);

        expect(result.diagonal.anyDiagonal).toBe(true);
    });

    it('rechaza una matriz con valores fuera de la diagonal', () => {
        const result = calculateStatistics([[[1, 2], [0, 1]]]);

        expect(result.diagonal.byMatrix).toEqual([false]);
        expect(result.diagonal.anyDiagonal).toBe(false);
    });

    it('acepta residuos de coma flotante dentro de la tolerancia', () => {
        // Este es el caso real: la R de una factorización QR trae valores como 1e-17 donde
        // matemáticamente hay ceros. Con === 0 esta matriz daría "no diagonal".
        const result = calculateStatistics([[[3, 1e-17], [-2.5e-18, 4]]]);

        expect(result.diagonal.anyDiagonal).toBe(true);
    });

    it('escala la tolerancia con la magnitud de la matriz', () => {
        // 1e-4 es despreciable frente a valores de 1e9, así que sigue siendo diagonal.
        const result = calculateStatistics([[[1e9, 1e-4], [0, 2e9]]]);

        expect(result.diagonal.anyDiagonal).toBe(true);
    });

    it('respeta una tolerancia personalizada', () => {
        const matrices = [[[1, 0.001], [0, 1]]];

        expect(calculateStatistics(matrices, 1e-9).diagonal.anyDiagonal).toBe(false);
        expect(calculateStatistics(matrices, 1e-2).diagonal.anyDiagonal).toBe(true);
    });

    it('no considera diagonal a una matriz rectangular', () => {
        // La definición de matriz diagonal exige que sea cuadrada.
        const result = calculateStatistics([[[1, 0, 0], [0, 2, 0]]]);

        expect(result.diagonal.anyDiagonal).toBe(false);
    });

    it('reporta el resultado por matriz y el agregado', () => {
        const result = calculateStatistics([
            [[1, 2], [3, 4]],
            [[5, 0], [0, 6]],
        ]);

        expect(result.diagonal.byMatrix).toEqual([false, true]);
        expect(result.diagonal.anyDiagonal).toBe(true);
    });

    it('considera diagonal a la matriz nula', () => {
        // Todos sus elementos fuera de la diagonal son cero, así que cumple la definición.
        const result = calculateStatistics([[[0, 0], [0, 0]]]);

        expect(result.diagonal.anyDiagonal).toBe(true);
    });
});

describe('calculateStatistics — validación', () => {
    it('rechaza un arreglo de matrices vacío', () => {
        expectErrorCode(() => calculateStatistics([]), ErrorCode.INVALID_PAYLOAD);
    });

    it('rechaza algo que no sea un arreglo', () => {
        expectErrorCode(() => calculateStatistics('no soy una matriz'), ErrorCode.INVALID_PAYLOAD);
    });

    it('rechaza una matriz vacía', () => {
        expectErrorCode(() => calculateStatistics([[]]), ErrorCode.EMPTY_MATRIX);
    });

    it('rechaza una matriz con filas vacías', () => {
        expectErrorCode(() => calculateStatistics([[[]]]), ErrorCode.EMPTY_MATRIX);
    });

    it('rechaza filas de longitud desigual e informa cuál', () => {
        const error = expectErrorCode(
            () => calculateStatistics([[[1, 2, 3], [4, 5]]]),
            ErrorCode.INCONSISTENT_ROW_LENGTH,
        );

        // Coordenadas en base 1: el destinatario lee una grilla, no un array.
        expect(error.details).toMatchObject({ matrix: 1, row: 2, expected: 3, actual: 2 });
    });

    it.each([
        ['NaN', Number.NaN],
        ['Infinity', Number.POSITIVE_INFINITY],
        ['-Infinity', Number.NEGATIVE_INFINITY],
    ])('rechaza %s como valor de celda', (_label, value) => {
        const error = expectErrorCode(
            () => calculateStatistics([[[1, 2], [3, value]]]),
            ErrorCode.INVALID_NUMBER,
        );

        expect(error.details).toMatchObject({ matrix: 1, row: 2, column: 2 });
    });

    it('rechaza celdas que no son números', () => {
        expectErrorCode(() => calculateStatistics([[['5']]]), ErrorCode.INVALID_NUMBER);
        expectErrorCode(() => calculateStatistics([[[null]]]), ErrorCode.INVALID_NUMBER);
    });

    it('identifica en qué matriz está el error', () => {
        const error = expectErrorCode(
            () => calculateStatistics([[[1]], [[Number.NaN]]]),
            ErrorCode.INVALID_NUMBER,
        );

        expect(error.details.matrix).toBe(2);
    });

    it('rechaza matrices que exceden el límite de dimensiones', () => {
        const oversized = [Array.from({ length: 101 }, (_, index) => index)];

        expectErrorCode(() => calculateStatistics([oversized]), ErrorCode.MATRIX_TOO_LARGE);
    });
});
