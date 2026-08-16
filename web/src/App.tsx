import { useState } from 'react';

import { ApiError, calculateStatistics, factorize } from './api/client';
import {
    MatrixInput,
    emptyCells,
    parseCells,
    resizeCells,
} from './components/MatrixInput';
import {
    MatrixView,
    classifyOrthogonal,
    classifyTriangular,
} from './components/MatrixView';
import { RotationField } from './components/RotationField';
import { StatsPanel } from './components/StatsPanel';
import type { Dimensions, Direction, QRResult, Statistics } from './types/api';

const MIN_DIMENSION = 1;

/**
 * Tope de la grilla editable. Es más bajo que el límite de los backends (100) a propósito:
 * una matriz de 100×100 son diez mil campos de texto, inmanejable en pantalla. El backend
 * sigue aceptando más para clientes que no sean esta interfaz.
 */
const MAX_DIMENSION = 12;

/** Ejemplo precargado: la matriz 3×2 clásica de Householder. Da algo que factorizar. */
const SAMPLE: string[][] = [
    ['12', '-51'],
    ['6', '167'],
    ['-4', '24'],
];

/** Qué operación está en curso, para deshabilitar la interfaz sin ambigüedad. */
type Pending = 'none' | 'qr' | 'statistics';

/**
 * Estadísticas de la matriz ingresada, junto a las dimensiones que tenía al calcularlas.
 *
 * Las dimensiones se guardan aquí y no se leen de la grilla al renderizar: si el usuario
 * cambia el tamaño después de calcular, el panel seguiría mostrando resultados viejos con
 * dimensiones nuevas, y la explicación de por qué una matriz no puede ser diagonal dejaría
 * de corresponder con los datos.
 */
interface StandaloneStats {
    statistics: Statistics;
    dimensions: Dimensions;
}

export default function App() {
    const [cells, setCells] = useState<string[][]>(SAMPLE);
    const [rotations, setRotations] = useState(0);
    // El sentido se conserva aunque el slider vuelva a 0: deshabilitar no es olvidar.
    const [direction, setDirection] = useState<Direction>('clockwise');

    const [pending, setPending] = useState<Pending>('none');
    const [error, setError] = useState<string | null>(null);
    const [result, setResult] = useState<QRResult | null>(null);
    const [standaloneStats, setStandaloneStats] = useState<StandaloneStats | null>(null);

    const rows = cells.length;
    const columns = cells[0]?.length ?? 0;
    const busy = pending !== 'none';

    const resize = (nextRows: number, nextColumns: number) => {
        setCells(resizeCells(cells, nextRows, nextColumns));
    };

    /** Lee la grilla y devuelve la matriz, o publica el error de formato y devuelve null. */
    const readMatrix = () => {
        const parsed = parseCells(cells);
        if ('error' in parsed) {
            setError(
                `El valor de la fila ${parsed.error.row}, columna ${parsed.error.column} no es un número válido.`,
            );
            return null;
        }
        return parsed.matrix;
    };

    /** Ejecuta una acción contra la API gestionando estado de carga y errores. */
    const run = async (kind: Exclude<Pending, 'none'>, action: () => Promise<void>) => {
        const matrix = readMatrix();
        if (!matrix) return;

        setPending(kind);
        setError(null);
        try {
            await action();
        } catch (caught) {
            // El cliente ya tradujo el código al español; aquí solo se muestra.
            setError(caught instanceof ApiError ? caught.message : 'Ocurrió un error inesperado.');
        } finally {
            setPending('none');
        }
    };

    const handleFactorize = () =>
        run('qr', async () => {
            const matrix = readMatrix();
            if (!matrix) return;

            setStandaloneStats(null);
            setResult(await factorize({ matrix, rotations, direction }));
        });

    const handleStatistics = () =>
        run('statistics', async () => {
            const matrix = readMatrix();
            if (!matrix) return;

            setResult(null);
            setStandaloneStats({
                statistics: await calculateStatistics(matrix),
                // Se capturan las dimensiones de lo que realmente se envió, no las de la grilla
                // en el momento de renderizar.
                dimensions: { rows: matrix.length, columns: matrix[0].length },
            });
        });

    return (
        <div className="mx-auto max-w-[900px] px-6 pb-32">
            <header className="border-b border-line py-14">
                <p className="mb-3 font-mono text-[11px] uppercase tracking-[0.18em] text-ink-soft">
                    Coding Challenge · Interseguro
                </p>
                <h1 className="mb-4 font-mono text-4xl font-semibold leading-tight tracking-tight text-ink sm:text-5xl">
                    Factorización QR
                </h1>
                <p className="max-w-[62ch] text-[19px] text-ink-soft">
                    Ingresa una matriz rectangular y obtén su descomposición en una matriz{' '}
                    <span className="font-mono text-ortho">Q</span> ortogonal y una{' '}
                    <span className="font-mono text-upper">R</span> triangular superior, junto con las
                    estadísticas de ambas.
                </p>
            </header>

            {/* ---------- Entrada ---------- */}
            <section className="border-b border-line py-12">
                <h2 className="mb-1 font-mono text-2xl font-semibold tracking-tight text-ink">
                    Matriz de entrada
                </h2>
                <p className="mb-6 text-ink-soft">
                    Las celdas vacías se interpretan como cero.
                </p>

                <div className="mb-6 flex flex-wrap items-end gap-6">
                    {(
                        [
                            ['Filas', rows, (value: number) => resize(value, columns)],
                            ['Columnas', columns, (value: number) => resize(rows, value)],
                        ] as const
                    ).map(([label, value, onChange]) => (
                        <label key={label} className="block">
                            <span className="mb-2 block font-mono text-[11px] uppercase tracking-[0.14em] text-ink-soft">
                                {label}
                            </span>
                            <input
                                type="number"
                                min={MIN_DIMENSION}
                                max={MAX_DIMENSION}
                                value={value}
                                disabled={busy}
                                onChange={(event) => {
                                    const next = Number(event.target.value);
                                    if (next >= MIN_DIMENSION && next <= MAX_DIMENSION) onChange(next);
                                }}
                                className="w-24 rounded-[5px] border border-line bg-surface px-3 py-2 font-mono text-sm text-ink disabled:opacity-60"
                            />
                        </label>
                    ))}

                    <button
                        type="button"
                        disabled={busy}
                        onClick={() => setCells(emptyCells(rows, columns))}
                        className="rounded-[5px] border border-line px-4 py-2 font-mono text-xs text-ink transition-colors hover:bg-hover disabled:opacity-50"
                    >
                        Limpiar
                    </button>
                </div>

                <div className="overflow-x-auto pb-2">
                    <MatrixInput cells={cells} onChange={setCells} disabled={busy} />
                </div>
            </section>

            {/* ---------- Rotación ---------- */}
            <section className="border-b border-line py-12">
                <h2 className="mb-1 font-mono text-2xl font-semibold tracking-tight text-ink">
                    Rotación previa
                </h2>
                <p className="mb-6 max-w-[62ch] text-ink-soft">
                    Opcional. La matriz se rota antes de factorizarla; rotar una matriz rectangular
                    intercambia sus dimensiones.
                </p>

                <RotationField
                    rotations={rotations}
                    direction={direction}
                    onRotationsChange={setRotations}
                    onDirectionChange={setDirection}
                    disabled={busy}
                />
            </section>

            {/* ---------- Acciones ---------- */}
            <section className="border-b border-line py-12">
                <div className="flex flex-wrap gap-3">
                    <button
                        type="button"
                        onClick={handleFactorize}
                        disabled={busy}
                        className="rounded-[5px] bg-ink px-6 py-3 font-mono text-[13px] tracking-wide text-white transition-colors hover:bg-vector disabled:opacity-50"
                    >
                        {pending === 'qr' ? 'Calculando…' : 'Calcular QR'}
                    </button>

                    <button
                        type="button"
                        onClick={handleStatistics}
                        disabled={busy}
                        className="rounded-[5px] border border-line px-6 py-3 font-mono text-[13px] tracking-wide text-ink transition-colors hover:bg-hover disabled:opacity-50"
                    >
                        {pending === 'statistics'
                            ? 'Calculando…'
                            : 'Calcular estadísticas de la matriz ingresada'}
                    </button>
                </div>

                {/* La nota enumera las métricas invariantes en vez de generalizar, porque la
                        verificación de matriz diagonal sí puede cambiar al rotar. */}
                <p className="mt-4 max-w-[68ch] font-mono text-[11px] leading-relaxed text-ink-soft">
                    El segundo botón consulta directamente la API de estadísticas con la matriz sin rotar:
                    máximo, mínimo, promedio y suma son los mismos valores en otro orden.
                </p>

                {busy && (
                    <p className="mt-4 font-mono text-[13px] text-vector">
                        Procesando… La primera petición puede tardar hasta un minuto si el servicio está
                        iniciándose.
                    </p>
                )}

                {error && (
                    <div
                        role="alert"
                        className="mt-5 rounded-r-[6px] border border-l-[3px] border-line border-l-mirror bg-mirror-soft px-5 py-4 text-[15px] text-ink"
                    >
                        {error}
                    </div>
                )}
            </section>

            {/* ---------- Resultado de la factorización ---------- */}
            {result && (
                <>
                    <section className="border-b border-line py-12">
                        <h2 className="mb-1 font-mono text-2xl font-semibold tracking-tight text-ink">
                            Factorización
                        </h2>
                        <p className="mb-6 text-ink-soft">
                            Matriz procesada de {result.input.dimensions.rows} ×{' '}
                            {result.input.dimensions.columns}.
                        </p>

                        {result.input.rotations > 0 && (
                            <div className="mb-8 overflow-x-auto rounded-[6px] border border-line bg-card px-6 py-5">
                                <div className="flex flex-wrap items-center gap-5">
                                    <MatrixView matrix={result.input.original} label="A" />
                                    <span className="font-mono text-lg text-ink-soft">
                                        → {result.input.rotations} ×{' '}
                                        {result.input.direction === 'clockwise' ? '90° ↻' : '90° ↺'} →
                                    </span>
                                    <MatrixView matrix={result.input.rotated} label="A′" />
                                </div>
                                <p className="mt-4 font-mono text-[12px] text-ink-soft">
                                    La factorización se aplica sobre la matriz rotada.
                                </p>
                            </div>
                        )}

                        <div className="overflow-x-auto rounded-[6px] border border-line bg-card px-6 py-5">
                            <div className="flex flex-wrap items-center gap-6">
                                <MatrixView
                                    matrix={result.factorization.q}
                                    label="Q"
                                    classify={classifyOrthogonal}
                                    description="Matriz Q ortogonal"
                                />
                                <MatrixView
                                    matrix={result.factorization.r}
                                    label="R"
                                    classify={classifyTriangular}
                                    description="Matriz R triangular superior"
                                />
                            </div>
                            <p className="mt-4 max-w-[68ch] font-mono text-[12px] leading-relaxed text-ink-soft">
                                Q tiene columnas ortonormales: perpendiculares entre sí y de longitud 1. R es
                                triangular superior, con ceros bajo la diagonal.
                            </p>
                        </div>
                    </section>

                    <section className="py-12">
                        <h2 className="mb-1 font-mono text-2xl font-semibold tracking-tight text-ink">
                            Estadísticas
                        </h2>
                        <StatsPanel statistics={result.statistics} source="factorization" />
                    </section>
                </>
            )}

            {/* ---------- Estadísticas de la matriz ingresada ---------- */}
            {standaloneStats && (
                <section className="py-12">
                    <h2 className="mb-1 font-mono text-2xl font-semibold tracking-tight text-ink">
                        Estadísticas de la matriz ingresada
                    </h2>
                    <StatsPanel
                        statistics={standaloneStats.statistics}
                        source="input"
                        dimensions={standaloneStats.dimensions}
                    />
                </section>
            )}
        </div>
    );
}
