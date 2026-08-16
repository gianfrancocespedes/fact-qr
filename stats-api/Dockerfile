FROM node:22-alpine

# dumb-init actúa como PID 1 y reenvía las señales al proceso de Node. Sin él, Node recibe
# SIGTERM como PID 1 y no lo maneja por defecto, así que la plataforma acaba matando el
# contenedor tras el tiempo de espera en lugar de detenerlo de forma ordenada.
RUN apk add --no-cache dumb-init

WORKDIR /app

# Los manifiestos se copian antes que el código para aprovechar la caché de capas: mientras
# las dependencias no cambien, Docker reutiliza la instalación aunque se modifique el código.
COPY package*.json ./

# `npm ci` instala exactamente lo fijado en package-lock.json, a diferencia de `npm install`,
# que puede resolver versiones distintas. --omit=dev descarta Vitest y Supertest, que no
# tienen ningún papel en producción.
RUN npm ci --omit=dev && npm cache clean --force

COPY src ./src

# La imagen de Node trae un usuario `node` sin privilegios. Ejecutar como root dentro del
# contenedor es innecesario para servir HTTP y amplía el impacto de una eventual ejecución
# de código arbitrario.
USER node

EXPOSE 3000

ENTRYPOINT ["dumb-init", "--"]
CMD ["node", "src/server.js"]
