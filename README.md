# Coding Challenge — Interseguro

Solución al reto técnico de la División TI: dos APIs que se comunican por HTTP para calcular la
**factorización QR** de una matriz rectangular y obtener estadísticas sobre las matrices
resultantes.

Se implementan los tres requisitos obligatorios y los **tres opcionales** (frontend, JWT y
pruebas).

---

## Arquitectura

```
                    ┌──────────────────────────────────────┐
                    │           web (React)                │
                    │        Nginx · puerto 5173           │
                    └───────┬──────────────────────┬───────┘
                            │                      │
              Calcular QR   │                      │  Calcular estadísticas
                            ▼                      │  (matriz sin rotar)
                    ┌───────────────┐              │
                    │    qr-api     │              │
                    │   Go + Fiber  │              │
                    │  puerto 8080  │              │
                    └───────┬───────┘              │
                            │  HTTP (Q y R)        │
                            ▼                      ▼
                    ┌──────────────────────────────────────┐
                    │             stats-api                │
                    │        Node.js + Express             │
                    │            puerto 3000               │
                    └──────────────────────────────────────┘
```

La **qr-api orquesta**, tal como indican los requisitos: recibe la matriz, la rota si se solicita,
calcula la factorización QR por reflexiones de Householder, envía **Q y R** a la stats-api y
compone la respuesta final.

El frontend consume **ambas APIs**: la qr-api para el flujo completo y la stats-api directamente
para calcular estadísticas de la matriz ingresada sin factorizarla.

---

## Stack

| Componente | Tecnología |
|---|---|
| qr-api | Go 1.23 · Fiber v2.52 |
| stats-api | Node.js 22 · Express 5.2 |
| web | React 19 · Vite · TypeScript · Tailwind CSS v4 |
| Contenedores | Docker · Docker Compose |
| Autenticación | JWT (HS256) |
| Despliegue | Railway |

---

## Ejecución con Docker (recomendado)

Único requisito: Docker.

```bash
git clone <url-del-repositorio>
cd coding-challenge-interseguro

cp .env.example .env          # copy .env.example .env  en Windows
docker compose up --build
```

| Servicio | URL |
|---|---|
| Frontend | http://localhost:5173 |
| qr-api | http://localhost:8080 |
| stats-api | http://localhost:3000 |

Para detenerlo: `docker compose down`.

---

## Ejecución sin Docker

Requiere Go ≥ 1.23 y Node ≥ 22. Tres terminales:

```bash
# 1 — stats-api
cd stats-api && npm install && npm start

# 2 — qr-api
cd qr-api && go run ./cmd/server

# 3 — web
cd web && npm install && npm run dev
```

Los valores por defecto ya apuntan a `localhost`; no hace falta configurar nada para desarrollo.

---

## Uso de la interfaz

1. **Ingresa una matriz.** Viene precargada con `[[12,-51],[6,167],[-4,24]]`, el ejemplo clásico
   de Householder. Puedes cambiar filas y columnas, y las celdas vacías cuentan como cero.
2. **Rotación (opcional).** El slider de 0 a 3 indica cuántos giros de 90° aplicar antes de
   factorizar; la palanca de sentido se habilita al elegir al menos una rotación.
3. **Calcular QR** → llama a la qr-api, que orquesta todo el flujo.
4. **Calcular estadísticas de la matriz ingresada** → llama directamente a la stats-api con la
   matriz **sin rotar**.

> **Por qué el segundo botón no rota la matriz.** Máximo, mínimo, promedio y suma son invariantes
> ante rotación: rotar permuta las celdas, no altera el conjunto de valores. Hacerlo sería trabajo
> sin efecto observable. La quinta métrica —si la matriz es diagonal— sí puede cambiar, y por eso
> la interfaz enumera cuáles son invariantes en lugar de generalizar.

---

## Pruebas

```bash
cd qr-api    && go test ./... -v     # 71 pruebas
cd stats-api && npm test             # 51 pruebas
```

Se verifican **propiedades matemáticas** en lugar de valores esperados, porque la factorización QR
no es única: invertir el signo de una columna de Q y de la fila correspondiente de R produce otra
factorización igualmente válida.

Las tres propiedades comprobadas —sobre 14 formas de matriz escritas a mano y 200 aleatorias— son:

- `Q·R ≈ A` (reconstrucción)
- `QᵀQ ≈ I` (ortogonalidad)
- `R` triangular superior

La implementación se validó además contra **NumPy (LAPACK)**, con coincidencia hasta precisión de
máquina incluso en matrices mal condicionadas.

---

## Fundamento matemático y referencias

El algoritmo no es original: la factorización QR por reflexiones de Householder es un método
estándar del álgebra lineal numérica, publicado en 1958 y usado desde entonces por todas las
bibliotecas serias de cálculo matricial. Lo que sigue documenta **de dónde proviene cada decisión**
y qué se verificó antes de implementarla.

### El método

Una reflexión de Householder es la matriz `H = I − 2vvᵀ`, con `‖v‖ = 1`. Geométricamente refleja
cualquier vector respecto al hiperplano perpendicular a `v`. Dos propiedades la hacen útil aquí:

1. **Es ortogonal y simétrica** (`H = Hᵀ = H⁻¹`), así que conserva las normas: es una isometría.
2. **Con el `v` adecuado, lleva un vector sobre el primer eje**, anulando de una sola vez todos sus
   componentes restantes.

El algoritmo aplica esas reflexiones columna por columna. Tras `min(n, m−1)` pasos, `A` ha quedado
triangular superior —esa es `R`— y el producto acumulado de las reflexiones es `Q`.

La clave de estabilidad está en el signo. Para la subcolumna `x` se elige:

```
α = −sign(x₁)·‖x‖        v = x − α·e₁
```

El signo **opuesto** al del pivote no es una convención arbitraria: con el mismo signo, `v₁` sería
`x₁ − ‖x‖`, la resta de dos números casi iguales cuando la columna ya está casi alineada. Eso es
**cancelación catastrófica** — los dígitos significativos se anulan y queda mayormente error de
redondeo, que al normalizar se amplifica. Con el signo opuesto las magnitudes se suman.
Implementado en [householder.go:71](qr-api/internal/matrix/householder.go#L71).

### Fuentes

Documentación de acceso libre consultada durante la implementación:

| Fuente | Qué se tomó de aquí |
|---|---|
| [LAPACK Users' Guide](https://www.netlib.org/lapack/lug/node40.html) — rutinas `xGEQRF` / `xORGQR` | La separación entre calcular los reflectores y materializar `Q`, y la confirmación de que es el método estándar de la industria |
| [Documentación de `numpy.linalg.qr`](https://numpy.org/doc/stable/reference/generated/numpy.linalg.qr.html) | La referencia contra la que se validó numéricamente; también el origen de la distinción `reduced` / `complete` |

El método de Householder es material estándar de álgebra lineal numérica —publicado en 1958 y
descrito en cualquier texto de la disciplina—, así que las propiedades que se citan abajo son
conocimiento establecido, no un resultado propio. La verificación que **sí** hizo este proyecto es
empírica: la salida se contrastó contra `numpy.linalg.qr`, y las pruebas comprueban las
propiedades que definen la factorización (`QᵀQ = I`, `QR = A`, `R` triangular superior).

### Por qué Householder y no las alternativas

La comparación se resolvió sobre un criterio: **estabilidad numérica**.

| | Gram-Schmidt clásico | GS modificado | Givens | **Householder** |
|---|---|---|---|---|
| Ortogonalidad de Q | Se pierde ∝ κ(A)² | Se pierde ∝ κ(A) | Preservada | **Preservada** |
| Costo (denso) | ≈ 2mn² | ≈ 2mn² | ≈ 50 % más | **2mn² − 2n³/3** |
| Mejor caso de uso | Didáctico | Métodos iterativos | Matrices dispersas | **Denso, propósito general** |

Gram-Schmidt resta a cada columna su proyección sobre las anteriores; cuando dos columnas apuntan
casi en la misma dirección esa resta cancela, y la ortogonalidad de `Q` se degrada de forma
proporcional al **cuadrado** del número de condición. Householder no puede sufrir
ese problema: cada reflexión es ortogonal por construcción, y el producto de matrices ortogonales
sigue siendo ortogonal, con independencia del redondeo.

Givens es igualmente estable, pero hace más trabajo sobre matrices densas. Gana cuando hay que
anular elementos aislados —matrices dispersas o casi triangulares—, que no es el caso de esta API.

### Verificación contra la referencia de la industria

Afirmar que el método es estable no basta: se contrastó la implementación en Go contra
`numpy.linalg.qr` (que llama a `dgeqrf`/`dorgqr` de LAPACK) midiendo tres residuos.

```
caso   ‖QR − A‖         ‖QᵀQ − I‖       |R| vs. LAPACK
2x3    3.11e-15         4.44e-16        1.78e-15
3x2    2.13e-14         1.22e-15        1.07e-14
3x3    8.88e-16         4.44e-16        4.00e-15
ill    3.33e-16         5.55e-16        2.71e-20
```

Todos los residuos están en el orden del épsilon de máquina (`2.22e-16`), incluida la matriz
deliberadamente mal condicionada (`ill`). El caso `3×2` reproduce el ejemplo canónico de la
literatura: para `A = [[12,−51],[6,167],[−4,24]]`, `R = [[−14,−21],[0,−175],[0,0]]`.

> La comparación es de **valores absolutos** por una razón matemática: la factorización QR no es
> única. Multiplicar una columna de `Q` y la fila correspondiente de `R` por `−1` produce otra
> factorización igual de válida, y LAPACK elige los signos según su propio criterio interno. Por
> eso las pruebas verifican **propiedades** (`QR ≈ A`, `QᵀQ ≈ I`, `R` triangular) y no valores
> literales.

---

## Documentación

| Documento | Contenido |
|---|---|
| [docs/api.md](docs/api.md) | Contratos completos, ejemplos `curl` y catálogo de errores |
| [docs/decisions.md](docs/decisions.md) | Las 11 decisiones técnicas y su sustento |
| [Fundamento matemático](#fundamento-matemático-y-referencias) | Bibliografía del algoritmo QR y verificación contra LAPACK |

---

## Decisiones destacadas

Están desarrolladas en [docs/decisions.md](docs/decisions.md); estas son las tres que más
condicionan la solución.

**La ambigüedad de la especificación se resuelve satisfaciendo ambas lecturas.** La descripción de
arquitectura habla de *rotación* y la funcionalidad requerida pide *factorización QR*. En lugar de
descartar una, el pipeline las
encadena: `rotación → QR`. Como el valor por defecto es 0 rotaciones, el comportamiento base es
exactamente la factorización QR de la matriz original.

**Householder en lugar de Gram-Schmidt.** Es el método que usan LAPACK, NumPy y MATLAB. Cada
reflexión es ortogonal por construcción, así que el redondeo no degrada la ortogonalidad de Q;
Gram-Schmidt clásico la pierde de forma proporcional a κ² por cancelación catastrófica. Las fuentes
consultadas y la verificación numérica están en
[Fundamento matemático](#fundamento-matemático-y-referencias).

**El backend devuelve códigos de error, no texto.** `ERROR_INVALID_NUMBER` con
`details: { row, column }` permite al frontend redactar *"El valor de la fila 2, columna 3 no es un
número válido"* en español. Los códigos son contrato de API; la traducción vive donde está el
idioma del usuario.

---

## Variables de entorno

Documentadas en [.env.example](.env.example). Las esenciales:

| Variable | Descripción |
|---|---|
| `JWT_SECRET` | Clave de firma HS256. **Obligatoria** y **idéntica en ambas APIs** |
| `STATS_API_URL` | URL de la stats-api vista desde la qr-api |
| `CORS_ALLOWED_ORIGINS` | Orígenes permitidos, separados por coma |
| `DIAGONAL_EPSILON` | Tolerancia para la verificación de matriz diagonal |
| `VITE_QR_API_URL` · `VITE_STATS_API_URL` | URLs que consume el frontend |

> **`JWT_SECRET` no tiene valor por defecto**: ninguna de las dos APIs arranca sin ella, y
> `docker compose up` aborta antes de construir. Es deliberado —un secreto con valor de respaldo
> se filtra a producción sin que nadie lo note—, así que el primer paso siempre es copiar
> `.env.example` a `.env`. En Railway se configura en el panel de cada servicio.

> **Las variables `VITE_*` se aplican en tiempo de build**, no de ejecución: Vite las sustituye en
> el código al compilar, porque el bundle se ejecuta en el navegador y no tiene acceso al entorno
> del contenedor. Cambiar una URL exige **reconstruir la imagen**, no basta con reiniciar.

---

## Notas de despliegue

**Binding IPv6.** Ambas APIs escuchan en `::` (dual-stack) y no en `0.0.0.0`, porque la red
privada de Railway es **solo IPv6** mientras que la pública es IPv4. Sin esto, la llamada interna
entre APIs fallaría únicamente en producción. En Fiber no basta con la dirección: hay que forzar
`Network: fiber.NetworkTCP`, ya que su valor por defecto es `tcp4`.

**Monorepo.** Un solo repositorio con tres servicios; en Railway cada uno apunta a su subcarpeta
mediante *Root Directory* (`/qr-api`, `/stats-api`, `/web`).

**Contenedores.** Los tres usan multi-stage builds y ejecutan como **usuario sin privilegios**. La
imagen de la qr-api pesa 24.5 MB frente a los ~350 MB de la imagen base de Go.
