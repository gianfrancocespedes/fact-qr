# =============================================================================
# Etapa 1 — Compilación
# =============================================================================
FROM golang:1.23-alpine AS builder

WORKDIR /build

# Las dependencias se copian antes que el código fuente para aprovechar la caché de
# capas: mientras go.mod y go.sum no cambien, Docker reutiliza la descarga de módulos
# aunque se modifique el código, que es lo que ocurre en la mayoría de los builds.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO_ENABLED=0 produce un binario estático, sin dependencias de bibliotecas del sistema.
# Es lo que permite ejecutarlo sobre una imagen mínima en la siguiente etapa.
# -ldflags="-s -w" descarta la tabla de símbolos y la información de depuración: unos 3 MB
# menos en una imagen que no se va a depurar dentro del contenedor.
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /build/server \
    ./cmd/server

# =============================================================================
# Etapa 2 — Ejecución
# =============================================================================
# Se parte de Alpine desnudo en lugar de arrastrar el compilador de Go: la imagen final
# baja de ~350 MB a ~15 MB, lo que acelera el despliegue y el arranque del contenedor.
FROM alpine:3.20

# Certificados raíz: sin ellos, cualquier llamada HTTPS saliente (la stats-api desplegada
# tras un dominio TLS) fallaría con un error de verificación de certificado.
RUN apk add --no-cache ca-certificates

# Ejecutar como root dentro del contenedor es innecesario para servir HTTP y amplía el
# impacto de una eventual ejecución de código arbitrario.
RUN adduser -D -u 10001 appuser

COPY --from=builder /build/server /usr/local/bin/server

USER appuser

EXPOSE 8080

# Forma exec (sin shell): el proceso recibe directamente las señales de parada, de modo que
# la plataforma puede detenerlo de forma ordenada en lugar de matarlo tras el tiempo de espera.
ENTRYPOINT ["/usr/local/bin/server"]
