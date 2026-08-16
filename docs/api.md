# Contratos de las APIs

Especificación de ambos servicios. Todos los ejemplos están tomados de respuestas reales.

- **qr-api** (Go + Fiber) — factorización QR y orquestación. Puerto `8080`.
- **stats-api** (Node + Express) — estadísticas sobre matrices. Puerto `3000`.

**Convenciones.** Todos los cuerpos son `application/json`. Los endpoints de negocio exigen
`Authorization: Bearer <token>`; `/health` y la emisión de token son públicos. Los valores
numéricos son `float64` sin redondear: el redondeo pertenece a la capa de presentación.

---

## qr-api

### `POST /api/v1/auth/token`

Emite un JWT firmado con HS256. **Público**: no requiere credenciales (ver
[ADR-09](decisions.md#adr-09-jwt-con-emisor-público) para el sustento de esa decisión).

**Petición** — el cuerpo es opcional:

```json
{ "subject": "web" }
```

**Respuesta `200`:**

```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJxci1hcGki...",
  "expiresAt": "2026-08-16T02:09:00Z",
  "tokenType": "Bearer"
}
```

| Campo | Tipo | Descripción |
|---|---|---|
| `token` | string | JWT a enviar en la cabecera `Authorization` |
| `expiresAt` | string | Vencimiento en UTC (ISO 8601) |
| `tokenType` | string | Siempre `Bearer` |

El token vale para **ambas APIs**: comparten el secreto de firma.

```bash
curl -X POST http://localhost:8080/api/v1/auth/token \
  -H "Content-Type: application/json" \
  -d '{"subject":"frontend"}'
```

---

### `POST /api/v1/qr` 🔒

Rota la matriz (opcional), calcula su factorización QR por reflexiones de Householder, delega el
cálculo de estadísticas en la stats-api y compone la respuesta.

**Petición:**

```json
{
  "matrix": [[12, -51], [6, 167], [-4, 24]],
  "rotations": 0,
  "direction": "clockwise"
}
```

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `matrix` | number[][] | Sí | Matriz rectangular m×n. Máximo 100×100 |
| `rotations` | integer | No (defecto `0`) | Giros de 90° previos a factorizar, de 0 a 3 |
| `direction` | string | No (defecto `clockwise`) | `clockwise` \| `counterclockwise` |

Omitir `rotations` y `direction` equivale a factorizar la matriz tal como llegó.

**Respuesta `200`:**

```json
{
  "input": {
    "original":   [[12, -51], [6, 167], [-4, 24]],
    "rotated":    [[12, -51], [6, 167], [-4, 24]],
    "rotations":  0,
    "direction":  "clockwise",
    "dimensions": { "rows": 3, "columns": 2 }
  },
  "factorization": {
    "q": [
      [-0.8571428571428565,  0.3942857142857141,  0.33142857142857135],
      [-0.4285714285714284, -0.9028571428571427, -0.03428571428571431],
      [ 0.2857142857142856, -0.1714285714285713,  0.9428571428571430]
    ],
    "r": [[-13.999999999999993, -21.00000000000001], [0, -175.00000000000003], [0, 0]]
  },
  "statistics": {
    "max": 0.942857142857143,
    "min": -175.00000000000003,
    "average": -14.029333333333335,
    "sum": -210.44000000000003,
    "count": 15,
    "diagonal": { "byMatrix": [false, false], "anyDiagonal": false }
  }
}
```

**Notas sobre la respuesta.**

- `input` viaja solo para **trazabilidad**: permite verificar la rotación sin recalcularla. No
  entra en el cálculo de estadísticas.
- Q es de m×m y R de m×n (factorización **completa**).
- `dimensions` corresponde a la matriz **efectivamente procesada**. Rotar una matriz rectangular
  intercambia sus dimensiones: una 2×3 con `rotations: 1` produce una 3×2.
- `count` = 15 confirma el alcance: 9 celdas de Q + 6 de R. La matriz de entrada no participa.
- Las estadísticas las calcula la **stats-api**, no la qr-api.

```bash
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/token \
  -H "Content-Type: application/json" -d '{}' | jq -r .token)

curl -X POST http://localhost:8080/api/v1/qr \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"matrix":[[12,-51],[6,167],[-4,24]]}'
```

---

### `GET /health`

Estado del servicio. **Público**: las plataformas de despliegue lo consultan sin credenciales.

```json
{ "status": "ok", "service": "qr-api" }
```

---

## stats-api

### `POST /api/v1/statistics` 🔒

Calcula las cinco métricas sobre un conjunto de matrices. Lo consumen tanto la qr-api
(servidor-a-servidor, enviando Q y R) como el frontend (directamente desde el navegador, con la
matriz ingresada).

**Petición:**

```json
{ "matrices": [[[3, 0], [0, 7]]] }
```

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---|---|
| `matrices` | number[][][] | Sí | Una o más matrices. Cada una hasta 100×100 |

**Respuesta `200`:**

```json
{
  "max": 7,
  "min": 0,
  "average": 2.5,
  "sum": 10,
  "count": 4,
  "diagonal": { "byMatrix": [true], "anyDiagonal": true }
}
```

| Campo | Descripción |
|---|---|
| `max` / `min` | Extremos sobre **todos** los valores de **todas** las matrices |
| `average` / `sum` | Promedio y suma del conjunto completo |
| `count` | Número total de valores considerados |
| `diagonal.byMatrix` | Un booleano por matriz, en el orden recibido |
| `diagonal.anyDiagonal` | `true` si alguna es diagonal |

**Sobre la verificación de diagonalidad.** Usa una tolerancia **relativa a la magnitud** de cada
matriz (`epsilon × max(1, |mayor valor|)`), no una comparación exacta contra cero: los ceros de una
factorización son residuos del orden de `1e-16` y `=== 0` produciría falsos negativos. Una matriz
**no cuadrada nunca es diagonal**, por definición. Detalle completo en
[ADR-06](decisions.md#adr-06-verificación-de-matriz-diagonal-con-tolerancia-relativa).

```bash
curl -X POST http://localhost:3000/api/v1/statistics \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"matrices":[[[3,0],[0,7]]]}'
```

---

### `GET /health`

```json
{ "status": "ok", "service": "stats-api" }
```

---

## Errores

Formato uniforme en ambas APIs. El backend **nunca** devuelve texto para mostrar al usuario: emite
un código estable más los datos con los que el frontend redacta el mensaje en español
([ADR-07](decisions.md#adr-07-errores-por-código-estable-traducción-en-el-frontend)).

```json
{
  "error": {
    "code": "ERROR_INVALID_NUMBER",
    "details": { "row": 2, "column": 3 }
  }
}
```

### Catálogo

La columna **Emitido por** indica qué servicio puede devolver cada código. Importa si se integra
contra una sola de las dos APIs: un cliente de la stats-api nunca verá los tres códigos exclusivos
de la qr-api.

| Código | HTTP | Emitido por | Situación | `details` |
|---|---|---|---|---|
| `ERROR_INVALID_NUMBER` | 400 | ambas | Una celda no es un número finito | `row`, `column`, `matrix` |
| `ERROR_EMPTY_MATRIX` | 400 | ambas | Matriz vacía o sin columnas | `matrix` |
| `ERROR_INCONSISTENT_ROW_LENGTH` | 400 | ambas | Las filas no tienen la misma longitud | `row`, `expected`, `actual` |
| `ERROR_MATRIX_TOO_LARGE` | 413 | ambas | Excede 100×100 | `rows`, `columns`, `maximum` |
| `ERROR_INVALID_PAYLOAD` | 400 | ambas | JSON malformado o campo ausente | `field` o `reason` |
| `ERROR_UNAUTHORIZED` | 401 | ambas | Token ausente, mal formado o inválido | `reason` |
| `ERROR_TOKEN_EXPIRED` | 401 | ambas | Token vencido | `expiredAt` |
| `ERROR_INTERNAL` | 500 | ambas | Fallo no previsto | — |
| `ERROR_INVALID_ROTATIONS` | 400 | solo qr-api | `rotations` fuera del rango 0–3 | `received`, `minimum`, `maximum` |
| `ERROR_INVALID_DIRECTION` | 400 | solo qr-api | `direction` no reconocido | `received`, `allowed` |
| `ERROR_STATS_SERVICE_UNAVAILABLE` | 503 | solo qr-api | La stats-api no respondió | — |

`rotations` y `direction` son parámetros exclusivos de `/api/v1/qr`, y
`ERROR_STATS_SERVICE_UNAVAILABLE` lo emite la qr-api cuando su llamada interna falla: la stats-api
no puede reportar su propia indisponibilidad.

Las coordenadas de `details` van en **base 1**: el destinatario del mensaje lee una grilla, no un
índice de array.

### Ejemplos

```bash
# Sin token
curl -X POST http://localhost:8080/api/v1/qr \
  -H "Content-Type: application/json" -d '{"matrix":[[1,2],[3,4]]}'
# 401 → {"error":{"code":"ERROR_UNAUTHORIZED","details":{"reason":"missing_token"}}}

# Rotaciones fuera de rango
# 400 → {"error":{"code":"ERROR_INVALID_ROTATIONS",
#                 "details":{"received":9,"minimum":0,"maximum":3}}}

# Filas de longitud desigual
# 400 → {"error":{"code":"ERROR_INCONSISTENT_ROW_LENGTH",
#                 "details":{"row":2,"expected":3,"actual":2}}}
```

**Propagación entre servicios.** Si la stats-api rechaza los datos, la qr-api **propaga el código
original** en lugar de enmascararlo tras un genérico: el frontend necesita el código específico
para redactar un mensaje útil. Solo cuando la stats-api no responde —caída, timeout o red— se
devuelve `ERROR_STATS_SERVICE_UNAVAILABLE`.

---

## Autenticación

Ambas APIs verifican JWT con **HS256** y un secreto compartido (`JWT_SECRET`). El token debe
enviarse como:

```
Authorization: Bearer <token>
```

**Endpoints públicos:** `POST /api/v1/auth/token` y `GET /health` en ambos servicios.

**Propagación.** Cuando la qr-api llama a la stats-api, **reenvía el mismo token** del cliente
original: la llamada interna viaja con la misma identidad, sin necesidad de credenciales de
servicio separadas.

**Validaciones aplicadas.** Firma, vencimiento (obligatorio), emisor (`qr-api`) y algoritmo
restringido a HS256 — sin esa restricción, un token con `"alg": "none"` sería aceptado sin
verificar la firma.
