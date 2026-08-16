import cors from 'cors';

export function buildCors() {
    const configured = (process.env.CORS_ALLOWED_ORIGINS ?? '')
        .split(',')
        .map((origin) => origin.trim())
        .filter(Boolean);

    if (configured.length === 0) {
        return cors();
    }

    return cors({
        origin: configured,
        methods: ['GET', 'POST', 'OPTIONS'],
        allowedHeaders: ['Content-Type', 'Authorization'],
    });
}
