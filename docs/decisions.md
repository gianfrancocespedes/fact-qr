# Decisiones técnicas (ADR)

Registro de las decisiones de diseño y su sustento. Los requisitos dejan varios puntos abiertos a
interpretación; este documento deja constancia de cómo se resolvió cada uno y por qué.

Cada decisión indica el problema, la opción elegida, las alternativas descartadas y por qué.

---

## ADR-01 · La ambigüedad "rotación" vs. "factorización QR"

**Problema.** La especificación se contradice entre dos secciones. La descripción de arquitectura
dice que la API en Go *"realizará la rotación de la matriz"*, mientras que la funcionalidad
requerida pide que *"devuelva la factorización QR de dicha matriz"*. Son operaciones distintas: la
rotación reordena celdas, la factorización descompone la matriz en dos.

**Decisión.** Implementar **ambas**, encadenadas: `rotación → factorización QR`. La API acepta
`rotations` (0–3, por defecto **0**) y `direction` (`clockwise` | `counterclockwise`), rota la
matriz de entrada y factoriza el resultado.

**Sustento.** Elegir una lectura y descartar la otra habría dejado la mitad de los requisitos sin
cumplir, apoyándose en suponer un error en la especificación. Encadenarlas satisface las dos, y
como el valor por defecto es 0 rotaciones, **el comportamiento base es exactamente la
factorización QR de la matriz original**: la funcionalidad adicional no distorsiona el caso
principal, es un superconjunto de lo pedido.

Nota complementaria: el método de Householder construye Q como composición de **reflexiones**, y
el de Givens mediante **rotaciones**. El término tiene, por tanto, una lectura coherente con QR
incluso sin el parámetro.

**Por qué el rango es 0–3 y no limita nada.** Rotar cuatro veces devuelve la matriz original, así
que 0–3 cubre el **espacio completo** de rotaciones distintas. No se recorta funcionalidad: se
expone el dominio real de la operación. Si alguien pidiera 5 rotaciones, 5 ≡ 1 (mód 4).

**Modelado del contrato.** Se eligió `rotations: 0..3` + `direction: enum` en lugar de un entero
con signo (`-3..3`). Razones: el JSON es autoexplicativo al leerlo, no hay aritmética de módulo
con negativos —en Go, `-1 % 4` da `-1`, no `3`, y es una fuente clásica de errores— y el enum
hace que la validación falle de forma evidente.

**Consecuencia asumida.** Rotar una matriz rectangular **intercambia sus dimensiones**: una 2×3
rotada 90° queda 3×2. Es correcto —QR soporta m≠n— y por eso la respuesta reporta las dimensiones
efectivas y el frontend redibuja la grilla.

---

## ADR-02 · Householder frente a Gram-Schmidt y Givens

**Problema.** Hay tres algoritmos clásicos para calcular la factorización QR, y los requisitos
exigen soportar matrices **rectangulares** e implementar la lógica *"de manera eficiente y
correcta"*.

**Decisión.** **Reflexiones de Householder**, devolviendo la factorización completa (Q de m×m,
R de m×n).

**Fuentes.** El método no se derivó desde cero: la factorización QR por reflexiones de Householder
es material estándar de álgebra lineal numérica. Las fuentes de acceso libre consultadas —la
[guía de LAPACK](https://www.netlib.org/lapack/lug/node40.html) y la
[documentación de NumPy](https://numpy.org/doc/stable/reference/generated/numpy.linalg.qr.html)—
están en el [README](../README.md#fundamento-matemático-y-referencias). La corrección de la
implementación se verificó contrastando la salida contra `numpy.linalg.qr`.

**Sustento.**

| Criterio | Gram-Schmidt clásico | GS modificado | **Householder** |
|---|---|---|---|
| Estrategia | Construye Q proyectando | Igual, en otro orden | Triangulariza A reflejando |
| Estabilidad | Mala: pierde ortogonalidad ∝ κ² | Aceptable (∝ κ) | **Backward stable** |
| Produce | QR reducida | QR reducida | QR completa |
| Costo | ≈ 2mn² | ≈ 2mn² | ≈ 2mn² − 2n³/3 |
| Uso real | Enseñar la intuición | Métodos iterativos | **LAPACK, NumPy, MATLAB** |

La razón de fondo es la **estabilidad numérica**. Gram-Schmidt resta a cada columna su proyección
sobre las anteriores; cuando dos columnas apuntan casi en la misma dirección, esa resta produce
**cancelación catastrófica**: se restan dos números casi iguales, los dígitos significativos se
anulan y lo que queda es mayormente error de redondeo. Al normalizar ese residuo minúsculo, el
error se amplifica, y la Q resultante deja de tener columnas ortogonales de forma medible.

Householder invierte la estrategia: aplica reflexiones que van anulando los elementos bajo la
diagonal. Cada reflexión **es exactamente ortogonal por construcción** —es una isometría, conserva
las normas—, así que el redondeo no puede degradar la ortogonalidad. La Q calculada satisface
`QᵀQ ≈ I` a nivel de épsilon de máquina, independientemente del condicionamiento de A.

Givens es igual de estable, pero realiza más operaciones sobre matrices densas; gana cuando la
matriz es dispersa o casi triangular, que no es el caso aquí.

**Verificación.** La implementación se comparó contra **NumPy (LAPACK)**, la referencia de la
industria, sobre cuatro matrices incluida una deliberadamente mal condicionada:

```
caso   reconstrucción   ortogonalidad   |R| vs. LAPACK
2x3    3.11e-15         4.44e-16        1.78e-15
3x2    2.13e-14         1.22e-15        1.07e-14
3x3    8.88e-16         4.44e-16        4.00e-15
ill    3.33e-16         5.55e-16        2.71e-20
```

Coincide hasta precisión de máquina. El caso `3x2` reproduce exactamente el ejemplo clásico:
`R = [[-14, -21], [0, -175], [0, 0]]`.

**Detalles de implementación que se derivan.**

1. **El signo del vector de Householder.** Se elige `α = −sign(x₁)·‖x‖`, opuesto al pivote. Ambos
   signos son válidos matemáticamente, pero con el mismo signo la primera componente queda
   `x₁ − ‖x‖`: dos números parecidos restándose, otra vez cancelación catastrófica. Con el signo
   opuesto las magnitudes se **suman** y el vector queda robusto.
2. **Nunca se forma la matriz H.** Construirla y multiplicar costaría `O(m²n)` por paso; aplicarla
   como `x − 2v(v·x)` cuesta `O(mn)`.
3. **La asimetría de acumulación.** R acumula por la izquierda (`R ← H·R`) y Q por la derecha
   (`Q ← Q·H`). Invertir un lado produce la transpuesta de Q, que sigue pareciendo ortogonal en
   una comprobación superficial pero hace que `Q·R` deje de reconstruir A.
4. **El caso degenerado.** Si una columna ya tiene ceros bajo la diagonal, no hay reflexión que
   aplicar y hay que saltarla; sin ese control, el algoritmo dividiría entre cero.

---

## ADR-03 · Quién orquesta la comunicación entre las APIs

**Problema.** Con dos APIs y un frontend, hay varias topologías posibles.

**Decisión.** La **qr-api (Go) orquesta**: recibe la matriz, rota, factoriza, llama por HTTP a la
stats-api y compone la respuesta final.

```
Frontend → qr-api (rota + QR) → stats-api (estadísticas) → qr-api compone → Frontend
```

**Sustento.** No es interpretación: la especificación lo indica literalmente.

> *"API en Go: [...] realizará la rotación de la matriz y luego **enviará los datos resultantes a
> la segunda API en Node.js**."*
>
> *"API en Node.js: Esta API **recibirá los datos de la matriz rotada de la API en Go**."*

Además, si el frontend orquestara llamando a cada API por separado, las dos APIs **no se
comunicarían entre sí** y quedaría incumplido el requisito explícito de *"implementar la
comunicación entre las dos API utilizando un mecanismo como HTTP"*.

Nota estructural: la stats-api no puede iniciar comunicación hacia el frontend —solo responde a
quien la llama—, por lo que el resultado completo vuelve necesariamente por la qr-api.

**Cómo se satisface "un frontend que consuma ambas APIs".** La interfaz tiene **dos botones**:

| Botón | Llama a | Envía | Devuelve |
|---|---|---|---|
| Calcular QR | qr-api, que orquesta | Matriz + `rotations` + `direction` | Rotada + Q + R + estadísticas |
| Calcular estadísticas de la matriz ingresada | stats-api, directo | Matriz **sin rotar** | Solo estadísticas |

El segundo botón no es un artificio para cumplir una casilla: es una acción con sentido propio,
ver las estadísticas de la matriz que se está escribiendo sin factorizarla.

**Consecuencia operativa.** Con ese botón, la stats-api recibe peticiones **directas del
navegador**, no solo llamadas servidor-a-servidor. Por eso necesita su propio middleware de CORS;
sin él, el botón falla en producción con un error de consola poco descriptivo.

---

## ADR-04 · El botón de estadísticas no aplica la rotación

**Problema.** Si el usuario elige 2 rotaciones y pulsa el botón de estadísticas, ¿debe enviarse la
matriz rotada o la original? Respetar el slider parecía lo coherente, pero exigiría implementar la
rotación también en el frontend o en la stats-api.

**Decisión.** Se envía la matriz **tal como fue ingresada**, sin rotar.

**Sustento — la razón es matemática, no de conveniencia.** **Máximo, mínimo, promedio y suma total
son invariantes ante rotación**: rotar permuta las celdas, no altera el multiconjunto de valores.
Aplicar la rotación antes de calcular sería trabajo sin ningún efecto observable en cuatro de las
cinco métricas.

Como beneficio secundario, el sistema conserva **una sola implementación de rotación**, en el
dominio de la qr-api.

**Matiz que la interfaz refleja.** La quinta métrica, la verificación de matriz diagonal, **sí**
puede cambiar: una matriz diagonal rotada 90° se vuelve antidiagonal. Por eso la nota de la
interfaz **enumera** las cuatro métricas invariantes en lugar de afirmar en bloque que "las
estadísticas no cambian". Esa precisión es deliberada.

**Alternativas descartadas.**

- *Rotar en el frontend*: duplica la transformación sin cambiar el resultado de 4 de 5 métricas.
- *Que la stats-api acepte `rotations`*: mueve la duplicación al backend sin ganancia.
- *Llamar a la qr-api solo para rotar y luego a la stats-api*: añade un salto de red completo para
  ahorrar unas ocho líneas de cliente.

**Sobre DRY.** Que Go y Node supieran rotar una matriz no habría violado el principio DRY. DRY
trata sobre **conocimiento de negocio duplicado**, no sobre líneas de código parecidas; una
rotación de 90° es una definición matemática estable, no una regla que pueda cambiar. De hecho, el
proyecto tiene duplicación legítima de ese tipo: la validación de la matriz vive en el frontend
(para la experiencia de uso) **y** en ambos backends (como contrato), y así debe ser.

---

## ADR-05 · Sobre qué matrices se calculan las estadísticas

**Problema.** La especificación indica que la segunda API calculará las métricas *"sobre los datos
de las matrices devueltas"*. La respuesta incluye la matriz de entrada, la rotada, Q y R: ¿cuáles entran
en el cálculo?

**Decisión.** Solo **Q y R**.

**Sustento.** Es la lectura estricta: "las matrices devueltas" por la factorización son
precisamente Q y R. La matriz original y la rotada viajan en la respuesta únicamente para
**trazabilidad** —permiten verificar la rotación de un vistazo y al frontend mostrar el antes y el
después—, pero no son resultado de la operación.

Es verificable en la respuesta: con una matriz 3×2, el campo `count` vale **15** = 9 celdas de Q
(3×3) + 6 de R (3×2). Si la entrada participara, sería 21.

---

## ADR-06 · Verificación de matriz diagonal con tolerancia relativa

**Problema.** Los requisitos piden *"verificar si alguna matriz es diagonal"*. La comparación natural
sería `valor === 0` para los elementos fuera de la diagonal, pero las matrices que llegan a la
stats-api provienen de una factorización: sus ceros son resultado de decenas de operaciones en
coma flotante.

**Decisión.** Comparar con una **tolerancia escalada con la magnitud de la matriz**:

```js
tolerancia = epsilon × Math.max(1, mayorValorAbsoluto)
```

**Sustento.** Donde matemáticamente hay un cero exacto, en coma flotante aparece un residuo del
orden de `1e-16`. Con `=== 0`, la API respondería "no es diagonal" sobre matrices que
demostrablemente lo son. Medido sobre `QᵀQ` —que **es** la identidad por definición— de dos
factorizaciones reales:

```
caso              peor residuo   con "=== 0"   con tolerancia
casi ortogonal    1.665e-16      NO diagonal   diagonal
rotación 30°      1.110e-16      NO diagonal   diagonal
```

**Por qué escalada y no un epsilon fijo.** "Cerca de cero" no significa nada en abstracto: solo
tiene sentido **relativo a los valores con los que se trabaja**. Un residuo de `4e-7` es ruido
entre valores de `2.5e6` y es un dato real entre valores de `8e-6`. Una tolerancia fija acierta
solo en el rango para el que se calibró.

**Por qué `max(1, ...)` y no solo `epsilon × magnitud`.** La guarda evita dos degeneraciones: con
una matriz de valores del orden de `1e-12`, la tolerancia caería a `2e-21` y un residuo normal de
redondeo la superaría; con la matriz nula, la tolerancia sería `0` y la comparación equivaldría de
nuevo a `=== 0`. La guarda garantiza que la tolerancia **nunca baje** del epsilon configurado,
solo pueda subir.

**Alcance de la métrica.** Las otras cuatro métricas —máximo, mínimo, promedio y suma— **no usan
tolerancia alguna**: son comparaciones y acumulaciones directas. El epsilon interviene únicamente
en la quinta, porque es la única que pregunta *"¿este valor **es** cero?"* en lugar de *"¿cuál es
el valor?"*.

**Matrices no cuadradas.** Una matriz rectangular **no puede** ser diagonal: la definición exige
que todo elemento fuera de la diagonal principal sea cero, lo que solo está definido para m = n.
El backend lo distingue, y la interfaz lo comunica como *"no aplica"* en lugar de un simple "no",
que sugeriría que la matriz fue evaluada y falló la prueba.

---

## ADR-07 · Errores por código estable, traducción en el frontend

**Problema.** ¿En qué idioma responde el backend cuando algo falla, y con qué nivel de detalle?

**Decisión.** El backend **nunca** devuelve texto destinado al usuario. Devuelve un **código
estable** y los datos necesarios para redactar el mensaje; el frontend mantiene el diccionario en
español.

```json
{ "error": { "code": "ERROR_INVALID_NUMBER", "details": { "row": 2, "column": 3 } } }
```

**Sustento.** Los códigos son **contrato de API**: los consume una máquina, no una persona.
Reescribir el mensaje en español no rompe ningún cliente, y la traducción vive donde está el
idioma del usuario. Es también lo que permite añadir otro idioma sin tocar el backend.

**Por qué `details` y no solo el código.** Es lo que hace que el mensaje sea **específico**. Con
`{ row: 2, column: 3 }`, el frontend redacta *"El valor de la fila 2, columna 3 no es un número
válido"*. Un diccionario que mapee código → texto fijo perdería esa precisión.

**Regla del frontend.** Ante un código sin traducción, se muestra un mensaje genérico digno; nunca
la etiqueta cruda `ERROR_*` en pantalla, que sería filtrar un detalle de implementación.

**Duplicación deliberada.** El catálogo de códigos existe en Go y en Node. No es un descuido:
ambos servicios exponen el mismo contrato al frontend, y extraerlo a una librería compartida
acoplaría dos servicios escritos en lenguajes distintos por una lista de constantes.

---

## ADR-08 · Precisión de punto flotante

**Decisión.** Trabajar internamente con `float64` **sin redondear**; redondear únicamente en la
capa de presentación.

**Sustento.** La factorización QR produce valores irracionales. Redondear en el backend
propagaría el error a cualquier cálculo posterior —las estadísticas se calculan sobre Q y R— y
degradaría la precisión que Householder se esfuerza en preservar. El frontend muestra cuatro
decimales y colapsa a `0` los valores por debajo de `1e-9`, porque mostrar `-2.7756e-17` en una
grilla de matriz confunde sin aportar nada.

Nota relacionada: la R que devuelve la qr-api tiene sus posiciones bajo la diagonal **anuladas
explícitamente**. Matemáticamente son cero exacto por construcción; anularlas hace que R sea
triangular superior de forma literal y no aproximada, tanto en el JSON como al verificar si es
diagonal.

---

## ADR-09 · JWT con emisor público

**Decisión.** Ambas APIs protegen sus endpoints de negocio con JWT (HS256, secreto compartido).
El endpoint emisor `POST /api/v1/auth/token` es **público**: emite un token sin pedir credenciales.

**Sustento.** Los requisitos piden *"aplicar un nivel de seguridad utilizando JWT para proteger las
consultas a las APIs"* — proteger las consultas, no implementar gestión de usuarios. Un sistema de
identidades completo requeriría registro, almacenamiento de contraseñas con hashing, base de datos
y recuperación de cuenta; nada de eso forma parte de los requisitos y todo desviaría el foco.

Lo que **sí** queda demostrado: firma y verificación HS256, middleware de protección en ambas
APIs, expiración y su distinción del token inválido, rechazo de `alg: none`, validación del emisor
y **propagación del token** en la llamada interna qr-api → stats-api.

Es una decisión consciente y acotada, no un descuido. Añadir credenciales sería sustituir el
cuerpo de `PostToken` por una consulta a base de datos, sin tocar el resto de la arquitectura: el
token ya transporta un `subject` y el middleware ya lo expone a los handlers.

**Endpoints públicos.** Solo la emisión de token y `/health`. Este último queda sin autenticación
a propósito: las plataformas de despliegue lo consultan sin credenciales, y exigir token lo
volvería inútil.

**Detalle de seguridad.** La verificación restringe el algoritmo a HS256 explícitamente. Sin esa
restricción, un token que declare `"alg": "none"` sería aceptado sin verificar la firma: es la
vulnerabilidad clásica de JWT, y hay una prueba dedicada a ella en ambas APIs.

---

## ADR-10 · Contenedores y despliegue

**Decisión.** Tres contenedores independientes, **multi-stage builds**, y un solo repositorio
(monorepo) desplegado en **Railway**.

**Por qué tres contenedores y no uno.** Runtimes distintos (Go, Node, Nginx), escalado
independiente, y aislamiento de fallos: si la stats-api cae, la qr-api sigue viva y responde
`ERROR_STATS_SERVICE_UNAVAILABLE` con estado 503 en lugar de arrastrar todo el sistema. Es también
lo que exige la plataforma, donde cada servicio se despliega por separado.

**Multi-stage.** La imagen de la qr-api pesa **24.5 MB** frente a los ~350 MB de
`golang:1.23-alpine`: la etapa de compilación produce un binario estático (`CGO_ENABLED=0`) y la
imagen final parte de Alpine desnudo. Menos peso es despliegue y arranque más rápidos.

**Usuario sin privilegios en los tres.** Por defecto los contenedores ejecutan como root; si una
vulnerabilidad permitiera ejecutar código, el atacante tendría root dentro del contenedor y, ante
un escape, en el anfitrión. No previene el fallo inicial: **limita el daño**. En Nginx implica
además servir en el puerto 8080, porque los puertos por debajo de 1024 están reservados a root.

**`dumb-init` en la stats-api.** En Linux, PID 1 ignora las señales que no maneja explícitamente,
y Node no maneja `SIGTERM`. Como PID 1 lo ignoraría y la plataforma acabaría enviando `SIGKILL`
tras el tiempo de espera, cortando peticiones en curso. Con `dumb-init` como PID 1, la señal se
reenvía a Node —que ya no es PID 1— y el contenedor para en **0.62 s** en lugar de ~10 s. Go no lo
necesita: su runtime maneja señales correctamente aun siendo PID 1.

**Binding IPv6 — obligatorio en Railway.** Su red privada entre servicios es **solo IPv6**,
mientras que la pública es IPv4. Un servidor que escuche en `:8080` o `0.0.0.0` solo acepta IPv4,
así que la llamada interna qr-api → stats-api daría timeout **únicamente en producción**. Ambas
APIs escuchan en `::`, que activa dual-stack: acepta IPv6 **y** IPv4.

En Fiber no basta con la dirección: su configuración usa `NetworkTCP4` por defecto e ignora el
formato de `[::]:8080`, abriendo un socket IPv4. Hay que forzar `Network: fiber.NetworkTCP`.
Verificado en local: ambas APIs escuchan en `::` y responden tanto en `127.0.0.1` como en `[::1]`.

**Un solo repositorio.** Railway apunta cada servicio a una subcarpeta mediante *Root Directory*.
Se prefiere a tres repositorios porque se clona una sola URL, un cambio que toca el
contrato de ambas APIs cabe en un único commit, y el `docker-compose.yml` y esta documentación
describen el sistema completo. Tres repositorios tienen sentido cuando equipos distintos son
dueños de cada servicio, que no es el caso.

**El `docker-compose.yml` se versiona** aunque Railway no lo use: es lo que permite clonar el
repositorio y levantar todo con un comando, sin instalar Go ni Node. Ambos entornos comparten los
mismos Dockerfiles y solo cambian las variables de entorno.

---

## ADR-11 · Estrategia de pruebas

**Decisión.** Verificar **propiedades matemáticas**, no valores esperados. 71 pruebas en Go y 51
en Node.

**Sustento.** La factorización QR **no es única**: multiplicar una columna de Q por −1 y la fila
correspondiente de R también por −1 deja el producto `Q·R` intacto. Un test que compare contra una
matriz fija fallaría ante implementaciones igualmente correctas, o ante un cambio de convención de
signos.

Lo que sí es único son las propiedades que definen la factorización:

1. **`Q·R ≈ A`** — reconstrucción.
2. **`QᵀQ ≈ I`** — ortogonalidad (la propiedad que Gram-Schmidt pierde).
3. **`R` triangular superior** — nada distinto de cero bajo la diagonal.

Con ese oráculo se pueden generar matrices aleatorias de cualquier tamaño y validarlas todas con
el mismo test: la suite incluye 14 formas escritas a mano más **200 matrices aleatorias**.

**Aislamiento del dominio.** La lógica matemática y la de estadísticas viven separadas de HTTP, lo
que permite probarlas sin levantar un servidor. Las pruebas de integración usan `app.Test()` en
Fiber y Supertest en Express, ambas en memoria, sin ocupar puertos ni depender de la red. La suite
completa de Node tarda menos de dos segundos.

**Dobles de prueba.** La qr-api define `StatsCalculator` como interfaz, no como tipo concreto, lo
que permite sustituir la stats-api por un doble en las pruebas del handler y verificar el flujo
completo —incluido que solo se envían Q y R— sin arrancar Node.
