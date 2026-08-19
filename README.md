# Coding Challenge — Interseguro

Dos APIs que se comunican por HTTP para calcular la **factorización QR** de una matriz rectangular
y obtener estadísticas sobre las matrices resultantes, con una interfaz web que consume ambas.

El sistema está contenerizado, protegido con JWT y cubierto por pruebas unitarias y de integración
en las dos APIs.

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

La **qr-api orquesta** el flujo: recibe la matriz, la rota si se solicita, calcula la factorización
QR por reflexiones de Householder, envía **Q y R** a la stats-api y compone la respuesta final.

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
cd <carpeta-del-repositorio>

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

## Documentación

| Documento | Contenido |
|---|---|
| [docs/api.md](docs/api.md) | Contratos completos, ejemplos `curl` y catálogo de errores |
| [docs/decisions.md](docs/decisions.md) | Las decisiones técnicas y su sustento |

---

## Decisiones destacadas

Están desarrolladas en [docs/decisions.md](docs/decisions.md); estas son las tres que más
condicionan la solución.

**Householder en lugar de Gram-Schmidt.** Es el método que usan LAPACK, NumPy y MATLAB. Cada
reflexión es ortogonal por construcción, así que el redondeo no degrada la ortogonalidad de Q;
Gram-Schmidt clásico la pierde de forma proporcional a κ² por cancelación catastrófica.

**La rotación es un parámetro, no un paso obligatorio.** El pipeline encadena `rotación → QR`, con
`rotations` por defecto en 0: sin ese parámetro, el comportamiento es la factorización QR de la
matriz tal como llegó. El rango 0–3 cubre el espacio completo de rotaciones distintas, porque girar
cuatro veces devuelve la matriz original.

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
