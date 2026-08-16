# =============================================================================
# Etapa 1 — Build de la aplicación
# =============================================================================
FROM node:22-alpine AS builder

WORKDIR /app

COPY package*.json ./
RUN npm ci

COPY . .

# Vite sustituye las variables VITE_* en el código **durante el build**, no al arrancar el
# contenedor: quedan incrustadas en el bundle. Por eso se reciben como argumentos de
# construcción y no como variables de entorno del contenedor, que llegarían demasiado tarde.
#
# Consecuencia operativa: cambiar la URL de una API exige reconstruir la imagen, no solo
# reiniciar el servicio. Está documentado en el README.
ARG VITE_QR_API_URL
ARG VITE_STATS_API_URL
ENV VITE_QR_API_URL=$VITE_QR_API_URL
ENV VITE_STATS_API_URL=$VITE_STATS_API_URL

RUN npm run build

# =============================================================================
# Etapa 2 — Servidor estático
# =============================================================================
# El resultado del build son archivos estáticos: no hace falta Node en producción. Nginx
# sobre Alpine deja la imagen final en ~50 MB en lugar de arrastrar el runtime completo.
FROM nginx:alpine

COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/dist /usr/share/nginx/html

# El puerto 8080 permite que Nginx corra sin privilegios de root: los puertos por debajo de
# 1024 están reservados al superusuario en Linux.
EXPOSE 8080

# La imagen oficial de Nginx trae el usuario `nginx` sin privilegios, pero los directorios de
# trabajo pertenecen a root. Se ajustan antes de cambiar de usuario.
RUN touch /var/run/nginx.pid \
    && chown -R nginx:nginx /var/run/nginx.pid /var/cache/nginx /usr/share/nginx/html

USER nginx

CMD ["nginx", "-g", "daemon off;"]
