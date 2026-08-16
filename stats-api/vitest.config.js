import { defineConfig } from 'vitest/config';

export default defineConfig({
    test: {
        // setupFiles corre antes de cada archivo de pruebas, y antes de que estos importen
        // el código del servicio: es el único punto donde JWT_SECRET llega a tiempo.
        setupFiles: ['./tests/setup.js'],
    },
});
