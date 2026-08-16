/**
 * Tipos que reflejan los contratos de ambas APIs.
 *
 * Se escriben a mano en lugar de generarlos: son pocos y estables, y tenerlos explícitos
 * documenta el contrato en el propio frontend.
 */

/** Matriz en orden fila-mayor, tal como viaja en el JSON. */
export type Matrix = number[][];

/** Sentido de giro aceptado por la qr-api. */
export type Direction = 'clockwise' | 'counterclockwise';

/** Dimensiones efectivas de la matriz procesada. */
export interface Dimensions {
    rows: number;
    columns: number;
}

/** Lo recibido y la transformación aplicada, para trazabilidad. */
export interface QRInput {
    original: Matrix;
    rotated: Matrix;
    rotations: number;
    direction: Direction;
    dimensions: Dimensions;
}

/** Matrices resultantes de la descomposición. */
export interface Factorization {
    q: Matrix;
    r: Matrix;
}

/** Verificación de diagonalidad, por matriz y agregada. */
export interface DiagonalReport {
    byMatrix: boolean[];
    anyDiagonal: boolean;
}

/** Estadísticas calculadas por la stats-api. */
export interface Statistics {
    max: number;
    min: number;
    average: number;
    sum: number;
    count: number;
    diagonal: DiagonalReport;
}

/** Respuesta compuesta de POST /api/v1/qr. */
export interface QRResult {
    input: QRInput;
    factorization: Factorization;
    statistics: Statistics;
}

/** Petición a POST /api/v1/qr. Los dos últimos campos son opcionales. */
export interface QRRequest {
    matrix: Matrix;
    rotations?: number;
    direction?: Direction;
}

/**
 * Error del contrato compartido.
 *
 * El backend nunca devuelve texto para mostrar: solo un código estable y los datos con
 * los que el frontend redacta el mensaje en español.
 */
export interface ApiErrorBody {
    error: {
        code: string;
        details?: Record<string, unknown>;
    };
}
