/**
 * Códigos de error de la API.
 */
export const ErrorCode = {
    INVALID_NUMBER: 'ERROR_INVALID_NUMBER',
    EMPTY_MATRIX: 'ERROR_EMPTY_MATRIX',
    INCONSISTENT_ROW_LENGTH: 'ERROR_INCONSISTENT_ROW_LENGTH',
    MATRIX_TOO_LARGE: 'ERROR_MATRIX_TOO_LARGE',
    INVALID_PAYLOAD: 'ERROR_INVALID_PAYLOAD',
    UNAUTHORIZED: 'ERROR_UNAUTHORIZED',
    TOKEN_EXPIRED: 'ERROR_TOKEN_EXPIRED',
    INTERNAL: 'ERROR_INTERNAL',
};

/**
 * Error de dominio con código estable.
 *
 * Extiende Error para que se propague con las herramientas habituales del lenguaje
 * (throw, stack trace) mientras transporta el código y los detalles del contrato.
 */
export class AppError extends Error {
    /**
     * @param {string} code Código estable del catálogo.
     * @param {object} [details] Contexto para que el frontend redacte un mensaje específico.
     * @param {number} [status] Código HTTP asociado.
     */
    constructor(code, details = undefined, status = 400) {
        super(code);
        this.name = 'AppError';
        this.code = code;
        this.details = details;
        this.status = status;
    }

    /** Representación que viaja en el cuerpo de la respuesta. */
    toJSON() {
        return this.details === undefined
            ? { error: { code: this.code } }
            : { error: { code: this.code, details: this.details } };
    }
}

/**
 * Señala una celda que no es un número finito o no es un número en absoluto. 
 * Se usa para validar la entrada antes de calcular.
 */
export function invalidNumber({ matrix, row, column }) {
    return new AppError(ErrorCode.INVALID_NUMBER, {
        matrix: matrix + 1,
        row: row + 1,
        column: column + 1,
    });
}

/** Señala una fila cuya longitud no coincide con la primera de su matriz. */
export function inconsistentRowLength({ matrix, row, expected, actual }) {
    return new AppError(ErrorCode.INCONSISTENT_ROW_LENGTH, {
        matrix: matrix + 1,
        row: row + 1,
        expected,
        actual,
    });
}
