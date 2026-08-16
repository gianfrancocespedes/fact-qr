import type { Matrix } from '@/types/api';

/** Decimales mostrados. El redondeo es solo de presentación: el backend nunca redondea. */
const DECIMALS = 4;

/**
 * Umbral bajo el cual un valor se muestra como cero.
 *
 * La factorización produce residuos de redondeo del orden de 1e-17 donde matemáticamente
 * hay ceros; mostrarlos como "0.0000" confundiría, y mostrarlos en notación científica
 * ensuciaría la lectura de la matriz.
 */
const DISPLAY_ZERO = 1e-9;

export type CellRole = 'plain' | 'zero' | 'diagonal' | 'orthogonal';

interface MatrixViewProps {
    matrix: Matrix;
    /** Etiqueta corta que precede a los corchetes (A, Q, R...). */
    label?: string;
    /** Asigna un papel visual a cada celda; permite colorear ceros y diagonal. */
    classify?: (row: number, column: number, value: number) => CellRole;
    /** Descripción para lectores de pantalla. */
    description?: string;
}

/** Formatea un valor para mostrarlo, colapsando los residuos de redondeo a cero. */
function formatValue(value: number): string {
    if (Math.abs(value) < DISPLAY_ZERO) {
        return '0';
    }

    const rounded = Number(value.toFixed(DECIMALS));
    return Number.isInteger(rounded) ? String(rounded) : rounded.toFixed(DECIMALS);
}

const ROLE_STYLES: Record<CellRole, string> = {
    plain: 'text-ink',
    zero: 'text-zero',
    diagonal: 'text-upper font-bold',
    orthogonal: 'text-ortho',
};

/**
 * Dibuja una matriz con corchetes reales.
 *
 * Los corchetes son dos bordes verticales, no caracteres ni una tabla: es lo que hace que
 * se lea como notación matemática y no como una hoja de cálculo.
 */
export function MatrixView({ matrix, label, classify, description }: MatrixViewProps) {
    const rows = matrix.length;
    const columns = matrix[0]?.length ?? 0;

    return (
        <div className="inline-flex items-center gap-2 align-middle">
            {label && (
                <span className="self-center font-mono text-[13px] font-bold text-ink">{label}</span>
            )}

            <div
                className="inline-flex items-stretch gap-[7px]"
                role="img"
                aria-label={description ?? `Matriz ${label ?? ''} de ${rows} por ${columns}`}
            >
                <div className="w-[9px] rounded-l-[3px] border-2 border-r-0 border-ink" />

                <div
                    className="grid gap-y-[2px] gap-x-[10px] px-[2px] py-[5px]"
                    style={{ gridTemplateColumns: `repeat(${columns}, auto)` }}
                >
                    {matrix.map((row, rowIndex) =>
                        row.map((value, columnIndex) => (
                            <div
                                key={`${rowIndex}-${columnIndex}`}
                                className={`min-w-[3.6em] px-[2px] py-px text-right font-mono text-sm tabular-nums ${
                                    ROLE_STYLES[classify?.(rowIndex, columnIndex, value) ?? 'plain']
                                }`}
                            >
                                {formatValue(value)}
                            </div>
                        )),
                    )}
                </div>

                <div className="w-[9px] rounded-r-[3px] border-2 border-l-0 border-ink" />
            </div>
        </div>
    );
}

/** Marca los ceros bajo la diagonal y resalta la diagonal: la firma visual de R. */
export const classifyTriangular = (row: number, column: number): CellRole => {
    if (row > column) return 'zero';
    if (row === column) return 'diagonal';
    return 'plain';
};

/** Colorea toda la matriz como ortogonal: la firma visual de Q. */
export const classifyOrthogonal = (): CellRole => 'orthogonal';
