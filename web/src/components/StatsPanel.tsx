import type { Dimensions, Statistics } from '@/types/api';

/** Decimales de presentación. El backend entrega float64 sin redondear. */
const DECIMALS = 4;

/** Origen de los datos analizados. Determina toda la redacción del panel. */
export type StatsSource = 'factorization' | 'input';

interface StatsPanelProps {
    statistics: Statistics;
    source: StatsSource;
    /** Dimensiones de la matriz analizada. Solo se usa cuando source es 'input'. */
    dimensions?: Dimensions;
}

function formatNumber(value: number): string {
    const rounded = Number(value.toFixed(DECIMALS));
    return Number.isInteger(rounded) ? String(rounded) : rounded.toFixed(DECIMALS);
}

/** Celda de una métrica: rótulo arriba, valor destacado abajo. */
function Metric({ label, value }: { label: string; value: string }) {
    return (
        <div className="rounded-[6px] border border-line bg-surface px-4 py-3">
            <div className="font-mono text-[11px] uppercase tracking-[0.12em] text-ink-soft">
                {label}
            </div>
            <div className="mt-1 font-mono text-lg font-bold tabular-nums text-ink">{value}</div>
        </div>
    );
}

/** Etiquetas de las matrices que devuelve la factorización, en el orden en que se envían. */
const FACTORIZATION_LABELS = ['Q', 'R'];

const SCOPE_TEXT: Record<StatsSource, string> = {
    factorization:
        'Calculadas por la API de Node sobre las matrices Q y R devueltas por la factorización.',
    input:
        'Consultadas directamente a la API de estadísticas sobre la matriz tal como fue ingresada.',
};

/**
 * Bloque de la quinta métrica: la verificación de matriz diagonal.
 *
 * Se separa del resto porque es la única que no es un número, y la única cuya redacción
 * depende de qué se analizó: sobre Q y R hay dos veredictos que detallar, sobre una sola
 * matriz basta una frase.
 */
function DiagonalReport({ statistics, source, dimensions }: StatsPanelProps) {
    const { diagonal } = statistics;

    // Una matriz no cuadrada no puede ser diagonal: la definición exige que todo elemento
    // fuera de la diagonal principal sea cero, lo que solo tiene sentido con m = n.
    // Se comprueba antes de leer el veredicto porque en ese caso el `false` del backend
    // significa "no aplica", no "se evaluó y no lo es"; mostrarlo como un simple "no"
    // sugeriría que la matriz falló una prueba que en realidad nunca se le pudo aplicar.
    const notApplicable =
        source === 'input' && dimensions !== undefined && dimensions.rows !== dimensions.columns;

    return (
        <div className="mt-4 rounded-[6px] border border-line border-l-[3px] border-l-upper bg-surface px-5 py-4">
            <div className="font-mono text-[11px] uppercase tracking-[0.14em] text-upper">
                Matriz diagonal
            </div>

            {notApplicable ? (
                <p className="mt-2 text-[15px] text-ink">
                    No aplica: solo las matrices cuadradas pueden ser diagonales, y la ingresada es de{' '}
                    <span className="font-mono">
                        {dimensions.rows} × {dimensions.columns}
                    </span>
                    .
                </p>
            ) : source === 'input' ? (
                <p className="mt-2 text-[15px] text-ink">
                    La matriz ingresada{' '}
                    <span className={diagonal.anyDiagonal ? 'font-bold text-ortho' : 'font-bold'}>
                        {diagonal.anyDiagonal ? 'es diagonal' : 'no es diagonal'}
                    </span>
                    .
                </p>
            ) : (
                <>
                    <p className="mt-2 text-[15px] text-ink">
                        {diagonal.anyDiagonal
                            ? 'Al menos una de las matrices analizadas es diagonal.'
                            : 'Ninguna de las matrices analizadas es diagonal.'}
                    </p>

                    <ul className="mt-3 flex flex-wrap gap-x-6 gap-y-1">
                        {diagonal.byMatrix.map((isDiagonal, index) => (
                            <li key={FACTORIZATION_LABELS[index] ?? index} className="font-mono text-[13px]">
                                <span className="text-ink-soft">
                                    {FACTORIZATION_LABELS[index] ?? `Matriz ${index + 1}`}:{' '}
                                </span>
                                <span className={isDiagonal ? 'font-bold text-ortho' : 'text-ink-soft'}>
                                    {isDiagonal ? 'sí' : 'no'}
                                </span>
                            </li>
                        ))}
                    </ul>

                    {/* Esta aclaración solo tiene sentido aquí: los valores de Q y R salen de decenas
                            de operaciones en coma flotante, así que donde debería haber un cero exacto
                            aparece un residuo de ~1e-17. Sobre una matriz escrita a mano no hay nada
                            que justificar. */}
                    <p className="mt-3 font-mono text-[11px] leading-relaxed text-ink-soft">
                        La verificación usa una tolerancia relativa a la magnitud de cada matriz, porque los
                        ceros de una factorización no son exactos en coma flotante.
                    </p>
                </>
            )}
        </div>
    );
}

/** Muestra las cinco métricas requeridas. */
export function StatsPanel({ statistics, source, dimensions }: StatsPanelProps) {
    const { max, min, average, sum, count } = statistics;

    return (
        <div>
            <p className="mb-4 text-[15px] text-ink-soft">{SCOPE_TEXT[source]}</p>

            <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
                <Metric label="Valor máximo" value={formatNumber(max)} />
                <Metric label="Valor mínimo" value={formatNumber(min)} />
                <Metric label="Promedio" value={formatNumber(average)} />
                <Metric label="Suma total" value={formatNumber(sum)} />
            </div>

            <DiagonalReport statistics={statistics} source={source} dimensions={dimensions} />

            <p className="mt-3 font-mono text-[11px] text-ink-soft">
                Calculado sobre {count} {count === 1 ? 'valor' : 'valores'}.
            </p>
        </div>
    );
}
