/**
 * Constantes compartidas por las pruebas.
 *
 * Son deliberadamente independientes del `.env`: las pruebas deben dar el mismo resultado en
 * cualquier máquina y en CI, donde no hay archivo de entorno. Este no es un secreto real —no
 * protege nada— sino un valor fijo para que firma y verificación usen la misma clave.
 */

/** Clave HS256 que setup.js instala en el entorno antes de cargar el servicio. */
export const TEST_JWT_SECRET = 'test-secret';

/** Emisor que la stats-api exige en los tokens. Debe coincidir con el de la qr-api. */
export const TEST_ISSUER = 'qr-api';
