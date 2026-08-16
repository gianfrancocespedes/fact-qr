import {
    AppError,
    ErrorCode,
    inconsistentRowLength,
    invalidNumber,
} from '../errors/codes.js';

/**
 * Tolerancia por defecto para comparar contra cero.
 *
 * La factorización QR produce valores irracionales: donde matemáticamente hay un cero,
 * en coma flotante aparece algo como 1e-17. Comparar con === 0 daría falsos negativos al
 * verificar si una matriz es diagonal.
 */
export const DEFAULT_EPSILON = 1e-9;

/** Límite de dimensiones */
const MAX_DIMENSION = 100;

/**
 * Valida una matriz recibida y la devuelve tal cual.
 *
 * @param {unknown} matrix Candidata a matriz.
 * @param {number} index Posición en el arreglo recibido, para los detalles del error.
 */
function validateMatrix(matrix, index) {
    if (!Array.isArray(matrix) || matrix.length === 0 || !Array.isArray(matrix[0]) || matrix[0].length === 0) {
        throw new AppError(ErrorCode.EMPTY_MATRIX, { matrix: index + 1 });
    }

    const columns = matrix[0].length;
    if (matrix.length > MAX_DIMENSION || columns > MAX_DIMENSION) {
        throw new AppError(ErrorCode.MATRIX_TOO_LARGE, {
            matrix: index + 1,
            rows: matrix.length,
            columns,
            maximum: MAX_DIMENSION,
        });
    }

    matrix.forEach((row, rowIndex) => {
        if (!Array.isArray(row) || row.length !== columns) {
            throw inconsistentRowLength({
                matrix: index,
                row: rowIndex,
                expected: columns,
                actual: Array.isArray(row) ? row.length : 0,
            });
        }

        row.forEach((value, columnIndex) => {
            // NaN e Infinity sobreviven a JSON.parse en algunos clientes y envenenarían el
            // cálculo en silencio: una sola celda no finita vuelve NaN la suma entera.
            if (typeof value !== 'number' || !Number.isFinite(value)) {
                throw invalidNumber({ matrix: index, row: rowIndex, column: columnIndex });
            }
        });
    });

    return matrix;
}

/**
 * Determina si una matriz es diagonal: cuadrada y con todos sus valores fuera de la
 * diagonal principal iguales a cero.
 *
 * La comparación usa una tolerancia escalada con la magnitud de la matriz, porque un
 * residuo de 1e-9 es despreciable entre valores del orden de 1e6 y enorme entre valores
 * de 1e-6.
 */
function isDiagonal(matrix, epsilon) {
    const rows = matrix.length;
    const columns = matrix[0].length;

    // Una matriz no cuadrada no puede ser diagonal: la definición exige que todo elemento
    // fuera de la diagonal principal sea cero, lo que solo está definido para m = n.
    if (rows !== columns) {
        return false;
    }

    let largest = 0;
    for (const row of matrix) {
        for (const value of row) {
            largest = Math.max(largest, Math.abs(value));
        }
    }

    const tolerance = epsilon * Math.max(1, largest);

    for (let i = 0; i < rows; i += 1) {
        for (let j = 0; j < columns; j += 1) {
            if (i !== j && Math.abs(matrix[i][j]) > tolerance) {
                return false;
            }
        }
    }

    return true;
}

/**
 * Calcula las estadísticas agregadas sobre un conjunto de matrices.
 *
 * Máximo, mínimo, promedio y suma se calculan sobre *todos* los valores de *todas* las
 * matrices tratados como un único conjunto.
 *
 * @param {number[][][]} matrices Arreglo de matrices.
 * @param {number} [epsilon] Tolerancia para la verificación de diagonalidad.
 */
export function calculateStatistics(matrices, epsilon = DEFAULT_EPSILON) {
    if (!Array.isArray(matrices) || matrices.length === 0) {
        throw new AppError(ErrorCode.INVALID_PAYLOAD, { field: 'matrices' });
    }

    const validated = matrices.map(validateMatrix);

    let max = -Infinity;
    let min = Infinity;
    let sum = 0;
    let count = 0;

    for (const matrix of validated) {
        for (const row of matrix) {
            for (const value of row) {
                if (value > max) max = value;
                if (value < min) min = value;
                sum += value;
                count += 1;
            }
        }
    }

    const diagonalByMatrix = validated.map((matrix) => isDiagonal(matrix, epsilon));

    return {
        max,
        min,
        average: sum / count,
        sum,
        count,
        diagonal: {
            byMatrix: diagonalByMatrix,
            anyDiagonal: diagonalByMatrix.some(Boolean),
        },
    };
}
