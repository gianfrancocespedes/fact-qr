import { useId } from 'react';

import type { Matrix } from '@/types/api';

/** Alineado con el límite que validan ambos backends. */
export const MAX_DIMENSION = 100;

interface MatrixInputProps {
    /** Celdas como texto: se conserva lo que el usuario escribió, sin convertir en cada tecla. */
    cells: string[][];
    onChange: (cells: string[][]) => void;
    disabled?: boolean;
}

/** Crea una grilla de celdas vacías. */
export function emptyCells(rows: number, columns: number): string[][] {
    return Array.from({ length: rows }, () => Array.from({ length: columns }, () => ''));
}

/**
 * Redimensiona la grilla conservando los valores que caben.
 *
 * Perder lo escrito al ajustar el tamaño sería frustrante: quien añade una columna a una
 * matriz de 3×3 espera conservar los nueve valores anteriores.
 */
export function resizeCells(cells: string[][], rows: number, columns: number): string[][] {
    return Array.from({ length: rows }, (_, row) =>
        Array.from({ length: columns }, (_, column) => cells[row]?.[column] ?? ''),
    );
}

/**
 * Convierte la grilla de texto a números.
 *
 * Devuelve la posición del primer valor inválido en lugar de lanzar: el llamador necesita
 * señalar *qué* celda falla, no solo que algo falló. Las coordenadas van en base 1 porque
 * el destinatario del mensaje lee una grilla, no un array.
 */
export function parseCells(
    cells: string[][],
): { matrix: Matrix } | { error: { row: number; column: number } } {
    const matrix: Matrix = [];

    for (let row = 0; row < cells.length; row += 1) {
        const parsedRow: number[] = [];

        for (let column = 0; column < cells[row].length; column += 1) {
            const raw = cells[row][column].trim();

            // Una celda vacía cuenta como cero: obligar a escribir 0 en matrices dispersas
            // sería tedioso y no aporta claridad.
            if (raw === '') {
                parsedRow.push(0);
                continue;
            }

            // Number('') devuelve 0 y Number('12abc') devuelve NaN; se comprueba lo segundo.
            const value = Number(raw.replace(',', '.'));
            if (!Number.isFinite(value)) {
                return { error: { row: row + 1, column: column + 1 } };
            }
            parsedRow.push(value);
        }
        matrix.push(parsedRow);
    }

    return { matrix };
}

/** Grilla editable de la matriz de entrada. */
export function MatrixInput({ cells, onChange, disabled }: MatrixInputProps) {
    const gridId = useId();
    const columns = cells[0]?.length ?? 0;

    const updateCell = (row: number, column: number, value: string) => {
        const next = cells.map((currentRow) => [...currentRow]);
        next[row][column] = value;
        onChange(next);
    };

    return (
        <div className="inline-flex items-stretch gap-[7px]">
            <div className="w-[9px] rounded-l-[3px] border-2 border-r-0 border-ink" />

            <div
                className="grid gap-[6px] px-[2px] py-[5px]"
                style={{ gridTemplateColumns: `repeat(${columns}, auto)` }}
            >
                {cells.map((row, rowIndex) =>
                    row.map((value, columnIndex) => (
                        <input
                            key={`${gridId}-${rowIndex}-${columnIndex}`}
                            type="text"
                            inputMode="decimal"
                            value={value}
                            disabled={disabled}
                            onChange={(event) => updateCell(rowIndex, columnIndex, event.target.value)}
                            // El label describe la posición porque un lector de pantalla anunciaría
                            // "cuadro de texto" sin contexto en una grilla de celdas idénticas.
                            aria-label={`Fila ${rowIndex + 1}, columna ${columnIndex + 1}`}
                            placeholder="0"
                            className="w-[5.5em] rounded-[4px] border border-line bg-surface px-2 py-1 text-right font-mono text-sm tabular-nums text-ink transition-colors placeholder:text-zero focus:border-vector disabled:opacity-60"
                        />
                    )),
                )}
            </div>

            <div className="w-[9px] rounded-r-[3px] border-2 border-l-0 border-ink" />
        </div>
    );
}
