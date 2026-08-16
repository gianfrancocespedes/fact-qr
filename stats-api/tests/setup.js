/**
 * Configuración común a todas las pruebas.
 *
 * Instala el secreto en el entorno antes de que se importe cualquier módulo del servicio. El
 * código de producción ya no tiene valor por defecto —sin JWT_SECRET no arranca—, así que las
 * pruebas deben proveerlo en lugar de depender del .env del desarrollador.
 */
import { TEST_JWT_SECRET } from './constants.js';

process.env.JWT_SECRET = TEST_JWT_SECRET;
