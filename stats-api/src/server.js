import 'dotenv/config';

import { createApp } from './app.js';
import { requireSecret } from './middleware/auth.js';

const DEFAULT_PORT = 3000;

try {
    // Se valida que exista el JWT_SECRET antes de inicar el servicio.
    requireSecret();
} catch (error) {
    console.error(`[stats-api] configuración inválida: ${error.message}`);
    process.exit(1);
}

const port = Number.parseInt(process.env.STATS_API_PORT ?? '', 10) || DEFAULT_PORT;

// `::` activa dual-stack, así el servicio es accesible por IPv4 o IPv6.
const HOST = '::';

createApp().listen(port, HOST, () => {
    console.log(`[stats-api] escuchando en el puerto ${port}`);
});
