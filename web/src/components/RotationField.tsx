import { useId } from 'react';

import type { Direction } from '@/types/api';

/** Rotar cuatro veces devuelve la matriz original, así que 0–3 cubre todas las distintas. */
export const MAX_ROTATIONS = 3;

interface RotationFieldProps {
    rotations: number;
    direction: Direction;
    onRotationsChange: (rotations: number) => void;
    onDirectionChange: (direction: Direction) => void;
    disabled?: boolean;
}

/**
 * Control de rotación: cuántas veces girar la matriz y en qué sentido.
 *
 * El rango cerrado de 0 a 3 hace que un valor inválido sea irrepresentable desde la
 * interfaz. Los backends validan igual, porque la API es pública y no puede confiar en su
 * cliente.
 */
export function RotationField({
    rotations,
    direction,
    onRotationsChange,
    onDirectionChange,
    disabled,
}: RotationFieldProps) {
    const sliderId = useId();

    // Con cero rotaciones el sentido no significa nada, así que la palanca se deshabilita.
    // El valor elegido se conserva en el estado del padre: al volver a subir el slider, el
    // usuario reencuentra lo que había seleccionado. Deshabilitar no es olvidar.
    const directionDisabled = disabled || rotations === 0;

    return (
        <div className="flex flex-wrap items-start gap-8">
            <div className="min-w-[200px]">
                <label
                    htmlFor={sliderId}
                    className="mb-2 block font-mono text-[11px] uppercase tracking-[0.14em] text-ink-soft"
                >
                    Rotaciones
                </label>

                <div className="flex items-center gap-3">
                    <input
                        id={sliderId}
                        type="range"
                        min={0}
                        max={MAX_ROTATIONS}
                        step={1}
                        value={rotations}
                        disabled={disabled}
                        onChange={(event) => onRotationsChange(Number(event.target.value))}
                        className="h-1 flex-1 cursor-pointer appearance-none rounded bg-line accent-vector disabled:cursor-not-allowed disabled:opacity-50"
                    />
                    <span className="w-[3ch] text-right font-mono text-lg font-bold tabular-nums text-vector">
                        {rotations}
                    </span>
                </div>

                <p className="mt-2 font-mono text-[11px] text-ink-soft">
                    {rotations === 0
                        ? 'Sin rotación: se factoriza la matriz tal como se ingresó.'
                        : `Giro de ${rotations * 90}° antes de factorizar.`}
                </p>
            </div>

            <fieldset className="min-w-[210px] border-0 p-0" disabled={directionDisabled}>
                <legend className="mb-2 font-mono text-[11px] uppercase tracking-[0.14em] text-ink-soft">
                    Sentido
                </legend>

                <div
                    className={`inline-flex overflow-hidden rounded-[5px] border border-line ${
                        directionDisabled ? 'opacity-45' : ''
                    }`}
                >
                    {(
                        [
                            ['clockwise', 'Horario'],
                            ['counterclockwise', 'Antihorario'],
                        ] as const
                    ).map(([value, label]) => (
                        <button
                            key={value}
                            type="button"
                            // aria-pressed comunica el estado a los lectores de pantalla; sin él, el
                            // grupo sonaría como dos botones sueltos sin indicar cuál está activo.
                            aria-pressed={direction === value}
                            disabled={directionDisabled}
                            onClick={() => onDirectionChange(value)}
                            className={`px-4 py-2 font-mono text-xs transition-colors ${
                                direction === value
                                    ? 'bg-ink text-white'
                                    : 'bg-transparent text-ink hover:bg-hover'
                            } disabled:cursor-not-allowed`}
                        >
                            {label}
                        </button>
                    ))}
                </div>

                {directionDisabled && !disabled && (
                    <p className="mt-2 font-mono text-[11px] text-ink-soft">
                        Se habilita al elegir al menos una rotación.
                    </p>
                )}
            </fieldset>
        </div>
    );
}
