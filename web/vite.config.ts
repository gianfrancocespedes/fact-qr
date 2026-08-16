import { fileURLToPath, URL } from 'node:url';

import tailwindcss from '@tailwindcss/vite';
import react from '@vitejs/plugin-react';
import { defineConfig } from 'vite';

export default defineConfig({
    plugins: [react(), tailwindcss()],
    resolve: {
        alias: {
            // El alias se declara aquí y en tsconfig.app.json: Vite resuelve en build,
            // TypeScript en el editor. Si solo se configura uno, el otro no lo reconoce.
            '@': fileURLToPath(new URL('./src', import.meta.url)),
        },
    },
    server: {
        port: 5173,
        // Necesario para que el contenedor sea accesible desde el anfitrión.
        host: true,
    },
});
