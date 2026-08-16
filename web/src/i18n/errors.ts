/**
 * Traducción de los códigos de error al español.
 *
 * El backend emite códigos estables; el texto vive aquí, donde está el idioma del usuario.
 * Así, reescribir un mensaje no rompe ningún cliente porque el contrato es el código.
 *
 * Cada entrada es una función y no una cadena fija: los `details` que acompañan al código
 * son justamente lo que permite decir "la fila 2, columna 3" en lugar de un genérico
 * "hay un valor inválido".
 */

type Details = Record<string, unknown>;
type Translator = (details?: Details) => string;

/** Lee un número de los detalles, o devuelve null si no viene o no es válido. */
function readNumber(details: Details | undefined, key: string): number | null {
    const value = details?.[key];
    return typeof value === 'number' && Number.isFinite(value) ? value : null;
}

/** Menciona la matriz solo cuando hay más de una en juego. */
function matrixPrefix(details?: Details): string {
    const index = readNumber(details, 'matrix');
    return index === null ? '' : `En la matriz ${index}: `;
}

const TRANSLATIONS: Record<string, Translator> = {
    ERROR_INVALID_NUMBER: (details) => {
        const row = readNumber(details, 'row');
        const column = readNumber(details, 'column');

        if (row === null || column === null) {
            return 'Hay un valor que no es un número válido.';
        }
        return `${matrixPrefix(details)}el valor de la fila ${row}, columna ${column} no es un número válido.`;
    },

    ERROR_EMPTY_MATRIX: () =>
        'La matriz está vacía. Ingresa al menos un valor para poder factorizarla.',

    ERROR_INCONSISTENT_ROW_LENGTH: (details) => {
        const row = readNumber(details, 'row');
        const expected = readNumber(details, 'expected');
        const actual = readNumber(details, 'actual');

        if (row === null || expected === null || actual === null) {
            return 'Todas las filas deben tener la misma cantidad de columnas.';
        }
        return `${matrixPrefix(details)}la fila ${row} tiene ${actual} ${
            actual === 1 ? 'valor' : 'valores'
        }, pero se esperaban ${expected}.`;
    },

    ERROR_MATRIX_TOO_LARGE: (details) => {
        const maximum = readNumber(details, 'maximum');
        return maximum === null
            ? 'La matriz es demasiado grande.'
            : `La matriz es demasiado grande: el máximo permitido es ${maximum} filas y ${maximum} columnas.`;
    },

    ERROR_INVALID_ROTATIONS: (details) => {
        const maximum = readNumber(details, 'maximum');
        return `El número de rotaciones debe estar entre 0 y ${maximum ?? 3}.`;
    },

    ERROR_INVALID_DIRECTION: () =>
        'El sentido de rotación indicado no es válido.',

    ERROR_INVALID_PAYLOAD: () =>
        'Los datos enviados no tienen el formato esperado.',

    ERROR_UNAUTHORIZED: () =>
        'No se pudo autenticar la petición. Recarga la página para obtener un nuevo acceso.',

    ERROR_TOKEN_EXPIRED: () =>
        'La sesión expiró. Recarga la página para continuar.',

    // El mensaje contempla que la primera petición tarde —arranque en frío o latencia de
    // red— en lugar de sonar a avería definitiva.
    ERROR_STATS_SERVICE_UNAVAILABLE: () =>
        'El servicio de estadísticas no respondió a tiempo. Si es la primera petición, puede estar iniciándose: espera unos segundos y vuelve a intentarlo.',

    ERROR_INTERNAL: () =>
        'Ocurrió un error inesperado en el servidor. Inténtalo nuevamente.',

    /** No proviene del backend: lo emite el cliente cuando la red falla por completo. */
    ERROR_NETWORK: () =>
        'No se pudo conectar con el servidor. Revisa tu conexión e inténtalo nuevamente.',
};

/**
 * Traduce un código de error a un mensaje en español.
 *
 * Ante un código desconocido devuelve un mensaje genérico digno: mostrar la etiqueta cruda
 * `ERROR_ALGO` en pantalla sería filtrar un detalle de implementación al usuario.
 */
export function translateError(code: string, details?: Details): string {
    const translate = TRANSLATIONS[code];
    return translate
        ? translate(details)
        : 'Ocurrió un error al procesar la solicitud. Inténtalo nuevamente.';
}
