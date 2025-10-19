# 🎯 Contexto y Arquitectura MVP — Agent666


## 📋 **¿Qué es Super Agente?**

Orquestador local que automatiza desarrollo de software mediante agentes IA. Permite que un agente ejecute tareas de programación de forma autónoma, integrándose opcionalmente con GitHub para colaboración en equipo.

**Problema que resuelve:**

* Automatizar tareas de desarrollo repetitivas
* Mantener contexto completo durante la implementación (el agente "recuerda" todo)
* Integrar ciclos de corrección automática (código → tests → fix → repetir)
* Balance entre trabajo local y colaboración remota

---

## 🎯 **Principios arquitectónicos del MVP**

### ✅ **Lo que SÍ es:**

* **App persistente interactiva:** Se ejecuta con `agent666` y mantiene un REPL/prompt interno para comandos
* **Sistema de cola automático:** Detecta automáticamente `docs/task/<issue>.md` en repos añadidos y los procesa secuencialmente
* **Watcher/Polling de tasks:** Monitorea continuamente los repositorios en busca de nuevas tareas
* **GitHub-first para trazabilidad:** Issues y transformaciones en GitHub Actions → visibilidad total del historial
* **Agente NATIVO en host:** Ejecuta directamente como comando del sistema (`droid`)
* **Ejecución en background del agente:** El agente procesa la cola automáticamente mientras el usuario mantiene acceso al prompt interactivo
* **Acceso interactivo al droid:** Comando `droid` (sin argumentos) para acceder a la sesión del agente del issue actual
* **Pipeline autocontenible:** El agente ejecuta todo el flujo en una sola pasada coherente (PRE → BUCLE → VALIDACIÓN → POST)
* **Flujo de decisión post-pipeline:** Antes de hacer push/PR, el usuario decide: hacer push y PR, abrir droid para continuar, o terminar ciclo sin push/PR
* **Mensajes en tiempo real:** La app muestra estado actual ("Trabajando en issue #123...")
* **Control del usuario vía prompt interno:**

  * `add-repo [path]` - Añadir repositorio (pregunta: GitHub habilitado, límite de intentos)
  * `rm-repo [path]` - Eliminar repositorio de la cola
  * `list` - Listar repositorios añadidos
  * `droid` - Acceder al droid del issue actual
  * `status` - Ver cola de trabajo y estado actual
  * `exit` - Salir de la app

### ❌ **Lo que NO es (en MVP):**

* ❌ **NO hay base de datos** (ni SQLite, ni persistencia compleja - solo archivo de configuración JSON)
* ❌ **NO hay agente dockerizado** (ejecuta nativo en el host)
* ❌ **NO hay UI web** (solo app CLI/REPL en MVP)
* ❌ **NO hay procesamiento paralelo de múltiples tareas simultáneas** (procesa una tarea por vez de la cola)
* ❌ **NO hay comandos globales independientes** (la app se ejecuta una vez y mantiene prompt interno)

### 🔮 **Escalabilidad futura (fuera de MVP):**

Cuando el MVP esté validado, se podrá añadir:

* Base de datos SQLite para persistencia avanzada y analytics
* Base de datos para aprendizaje de errores y finalizados
* Agente dockerizado para aislamiento y reproducibilidad
* UI web para visualización
* Trabajo en paralelo con múltiples pipelines simultáneos
* Procesamiento de múltiples repositorios en paralelo
* Sistema de prioridades en la cola
* Webhooks de GitHub para detección en tiempo real

**Filosofía MVP:** Validar la idea lo más rápido posible con la arquitectura más simple que funcione.



## 🛠️ **Stack tecnológico del MVP**

### Requisitos del usuario:

* **Node.js** (v18+)
* **Git** configurado
* **GitHub CLI (opcional):** si el usuario usa GitHub → **(`gh`)** autenticado
* **Agente IA instalado** (ej: `npm i -g opencli-code` o Claude Code según documentación)
* **Repositorio GitHub** **(opcional):** si el usuario usa GitHub → con permisos de escritura

### Tecnologías de la app:

* **Node.js CLI** (instalado globalmente vía npm)
* **readline** o **vorpal** (REPL/prompt interactivo interno - la app se mantiene ejecutándose)
* **inquirer** o **prompts** (para preguntas interactivas: GitHub habilitado, maxIterations, flujo de decisión)
* **chokidar** (file watcher para detectar nuevos `docs/task/<issue>.md` automáticamente)
* **EventEmitter** (comunicación entre cola, watcher, y pipeline)
* **simple-git** (para operaciones Git programáticas)
* **GitHub CLI (`gh`)** (para creación de PRs y sync cuando GitHub está habilitado)
* **child_process (spawn)** (para ejecutar agente en background)
* **node-pty** (para sesiones interactivas del terminal con droid - comando `droid`)
* **chalk** (para colores y mensajes en tiempo real "Trabajando en issue #123...")
* **ora** (spinners para procesos largos)
* **fs-extra** (operaciones de archivos mejoradas)
* **Sin frameworks pesados** (código simple y directo)

---

## 📂 **Estructura de archivos esperada**

```
repo/
├── todo.md                         # [INPUT] Usuario crea este archivo (opc)
├── docs/task/                      # [GENERADO] Tareas para el agente
│   |   <issue-id>-<issue-title>.md # Copia de issue para agente
│   |   end/<issue-id>-<issue-title>.md    # Reporte del agente en caso de finalizar
│   └── error/<issue-id>-<issue-title>.md  # Reporte del agente en caso de error
├── .github/
│   └── workflows/
│       ├── todo-to-issue.yml       # GitHub Action: todo.md → issue
│       └── issue-to-task.yml       # GitHub Action: issue → docs/task/<issue>.md
└── [código del proyecto]
```

### Convención de ramas:

* **Ramas del agente:** `agent666/issue-<issue-id>-<issue-desc>`
* **Ramas humanas:** `main`, `dev`, `qa`, `feature/*`, `fix/*`, etc…
* **Principio:** El agente NUNCA modifica ramas humanas directamente

---

## 📃 Formatos esperados

### Formato todo.md

Tendrá titles y list solo

```markdown
# [version] Titulo del issue
## Objective
Explicacion del objetivo
## Time 
Tiempo
## Apartado del body del issue
- Tarea del issue
	- Informacion de la tarea
		- Elemento lista dentro de la tarea
		- Otro elemento lista dentro de la tarea
	- Continua la informacion a continuacion de la lista anterior
		- Elemento de lista despues de informacion de la tarea
- Otra tarea del issue
## Otro apartado del body
```

Formato mínimo del todo.md

```markdown
# Titulo del issue
- Tarea del issue
```

### Formato issue

Si no hay `<version>` → v0.0.0

Si no hay `<Apartado del body del issue>` → <!-- ###  🖲️/💻/⛓️ Section Name —>

Si no hay `<Tiempo>` → 4-8h

`<numero-de-issue>`: proviene de GitHub

```markdown
# [<version>] <Titulo del issue> #<numero-de-issue>

## 🎯 Objective

<!-- Brief description of what needs to be accomplished -->

<Explicacion del objetivo>

## 🔑 Key Points

<Apartado del body del issue>

<!-- Key point what needs to be accomplished, representing the idea of this Task -->

- [ ] <Tarea del issue>
	<informacion de la tarea>
	- <Elemento lista dentro de la tarea>
	- <Otro elemento lista dentro de la tarea>
	<Continua la informacion a continuacion de la lista anterior>
	- <Elemento de lista despues de informacion de la tarea>
- [ ] <Otra tarea del issue>

## ⏱️Time

### 🤔 Estimated

<Tiempo>

### 😎 Real

🧠 _Tick this part just before you're going to close this issue - RECHECK_

## ✅ Definition of Done

🧠 _Tick this part just before you're going to close this issue - RECHECK_

<!-- Key point what needs to be accomplished, representing the must of this Task based on the Key Points -->

- [ ] <!-- Criterion 1 -->
- [ ] <!-- Criterion 2 -->
- [ ] Code tested and validated
- [ ] Documentation updated (if needed)
```



### Formato `docs/task/<issue>.md`

* El título ha de tener el formato **`<issue-id>-<issue-title>.md`**
* Ha de ser una copia del issue pero **sin los apartados (##) Time y Definition of Done**
* Ha de estar situado en `./docs/task/`

### Formato prompt

* **Ver en [prompt-template](../templates/prompt-template.md)**
* Para ello, se va a crear un formato fijo el cual, de forma dinámica, simplemente hace referencia al archivo `docs/task/<issue>.md` en el cual estarán los requerimientos actuales

#### Comportamiento

**El prompt tendrá diferentes `comportamiento/especialidad`**, que se configurara para cada repositorio:
* **full type:** Estos tipos de comportamiento serán completos, osea no se podrán marcar junto a otros
  * Sin comportamiento -> Le pasara solo la `docs/task/<issue>.md`
* **base type:** Estos tipos de comportamiento pueden ser complementados con `mod type`.
  * Default(sin indicar) -> El flujo completo 'base'
* **mod type:** Estos tipos complementan el comportamiento de los `base type`
  * Config -> Se le pasara un archivo `docs/agent666/config.md`. Esto le indicara al prompt, que encontrara configuraciones que modifican el comportamiento del flujo 'base', tanto restrictivas como aditivas, con **prioridad maxima** de aplicación sobre el flujo 'base'.
  * Esp -> Se le pasaran archivos de comportamiento especifico `docs/agent666/**--excepto config.md--`

---

## 🔄 **Pipeline default/prompt del agente (detalle técnico)**

El agente ejecuta **4 fases secuenciales** en una sola pasada:

### **PRE-BUCLE: Generar tests**

* Lee el contenido de `docs/task/<issue>.md`
* Genera tests ANTES de escribir código:

  * Tests unitarios
  * Tests de integración
  * Tests de endpoints (si aplica)
* Guarda los tests en la estructura esperada del proyecto
* **No avanza hasta tener tests escritos**

### **BUCLE: Código → Tests → Fix**

Iteración 1 a N (máximo configurable, ej: 10):

1. Generar/modificar código según el issue
2. Ejecutar tests
3. Generar/modificar docker files, según issue/necesidad, para ‘dockerizar’ app. Si hay más de un servicio debe implementar `compose.yml` obligatoriamente
4. Ejecutar app en docker
5. **Si tests pasan y docker se ejecuta:** salir del bucle ✅
6. **Si tests o docker falla:** analizar errores → corregir código → volver al paso 1
7. **Si llega al límite de la iteración:** genera un reporte en `./docs/task/error/<issue-id>-<issue-title>.md`

   * El error ha de tener la fecha y hora

**Contexto acumulativo:** El agente mantiene memoria de todas las iteraciones previas para no repetir errores.

### **VALIDACIÓN: Integración completa**

* Ejecutar validaciones adicionales según el proyecto:

  * Linting
  * Type checking
* Levantar servicios completos y validar flujo completo de la app usando curl

### **POST-BUCLE: Documentación**

* Actualizar README si hay cambios significativos
* **Git commit** con mensaje descriptivo

  * En el mensaje ha de escribir todos los feat, fix, etc que haya hecho
  * El commit lo ha de firmar al final con **CO-CREATED by Agent666 created by SKRTEEEEEE** y no CO-CREATED por droid
* Crear resumen de cambios para el PR en `./docs/task/end/<issue-id>-<issue-title>.md`

  * Con título standard commits → `pre-fix(version--si-hay): <issue-title>. Closes #<issue-id>`


## 🔀 **Flujo de decisión post-pipeline (si GitHub habilitado)**

Cuando el pipeline del agente termina exitosamente y el repositorio tiene GitHub habilitado, la CLI presenta al usuario las siguientes opciones:

### **Opciones disponibles:**

1. **Hacer push y PR**

   * Si el título de `docs/task/<issue>.md` tiene [version] hacer tag
   * Hace push de la rama `agent/<issue-id>-<issue-title>` al remoto
   * Crea un PR automáticamente usando `gh pr create`

     * El título del PR ha de seguir una estructura, misma que se usa como title del report del agente

       * Con título standard commits → `pre-fix(version--si-hay): <issue-title>. Closes #<issue-id>`
     * Si el título de `docs/task/<issue>.md` tiene [version] incluir como scope
     * Se ha de usar la información proporcionada por el agente para adjuntarla en el body
   * **Cierra la sesión del droid** para que pierda el contexto (proceso terminado)

2. **Abrir droid para continuar**

   * Abre sesión interactiva con el droid usando `node-pty`
   * El usuario puede hacer ajustes adicionales, preguntar al droid, o revisar código
   * Cuando el usuario cierra la sesión con `Ctrl+C`:

     * La CLI vuelve a presentar las mismas opciones (loop)
     * El contexto del droid se mantiene hasta que se elija "Hacer push y PR", "Abrir droid" o "Terminar sin push/PR"

3. **Terminar ciclo sin push/PR**

   * Mantiene los cambios locales en la rama del agente
   * **NO** hace push al remoto
   * **NO** crea PR
   * Cierra la sesión del droid
   * El usuario puede hacer push manual más tarde si lo desea, guardar información en local (todavía no habilitare el commando para hacer push y PR, en el futuro se podrá hacer por commando)

### **Flujo de decisión GitHub PR:**

```
Pipeline termina ✅
    ↓
¿GitHub habilitado?
    ↓ SÍ
Mostrar opciones:
    1. Hacer push y PR → Cierra droid, push, crea PR ✅
    2. Abrir droid → Sesión interactiva → Ctrl+C → VOLVER A MOSTRAR OPCIONES (loop)
    3. Terminar sin push/PR → Cierra droid, mantiene cambios locales ✅
    ↓ NO
Termina directamente (cambios locales guardados) ✅
```

### **Razón del diseño:**

* **Control total del usuario:** Puede revisar/ajustar antes de hacer público los cambios
* **Contexto persistente:** El droid mantiene memoria hasta que el usuario decida cerrar definitivamente
* **Flexibilidad:** Permite workflows híbridos (automático + manual)

---

## ⚙️ **Configuración necesaria**

### Configuración del repositorio:

La app debe configurar en los repositorios, si no existe y **el usuario usa GitHub para dicho repositorio**:

1. **GitHub Actions** (`.github/workflows/`)

   * `todo-to-issue.yml`: convierte `todo.md` a issue de GitHub
   * `issue-to-task.yml`: convierte issue a `docs/task/<issue>.md` con formato estándar

### Configuración del usuario:

El usuario debe configurar en los repositorios, si no lo tiene ya y **usa GitHub**:

* **Permisos de GitHub**
* El token de `gh` debe tener permisos de escritura en los repos
* Permisos para crear issues, branches y PRs
* En caso de que no los tenga y quiera usar GitHub, saltará una advertencia en la CLI

---

## 🎛️ GitHub Actions

La app, si el usuario configura un repositorio con opción de GitHub, deberá asegurarse de que tenga las GitHub Actions, y si no, crearlas.

Para la versión inicial, la app cargará tanto la action como el script necesario. Una vez testeado, se creará la GitHub Action personalizada (fuera de este MVP).

---

## 🚫 **Fuera de scope del MVP**

Estas funcionalidades **NO se implementan** en el MVP:

* ❌ Base de datos para persistir estado
* ❌ UI web o dashboard
* ❌ Ejecución en background o como servicio
* ❌ WebSockets o logs en tiempo real
* ❌ Docker para aislar el agente
* ❌ Docker-outside-of-Docker para levantar servicios
* ❌ Networks dedicadas por pipeline
* ❌ Queue de pipelines con prioridades
* ❌ Trabajo en paralelo (múltiples tareas simultáneas)
* ❌ Polling automático de GitHub issues
* ❌ Webhooks de GitHub
* ❌ Sync bidireccional complejo con GitHub
* ❌ Rollback automático si falla el agente
* ❌ Métricas o analytics
* ❌ Sistema de plugins o extensiones

**Regla de oro:** Si no está en el flujo básico (punto 1-8 del documento principal), no está en el MVP.

---

## 💡 **Por qué NO Docker en el MVP**

### Razones estratégicas:

1. **Simplicidad máxima:** Validar la idea rápido sin overhead de Docker
2. **Menos moving parts:** Sin Docker SDK, sin gestión de contenedores, sin networks
3. **Debugging inmediato:** Todo visible en terminal (stdout/stderr directo)
4. **Instalación trivial:** `npm i -g opencli-code && npm i -g super-agente` → listo
5. **Prototipado ágil:** Modificas código y reejecutas instantáneamente

### Lo que NO perdemos:

* ✅ El agente sigue siendo autocontenido (mantiene contexto, corrige errores)
* ✅ El flujo GitHub-first funciona igual
* ✅ El orquestador gestiona branches, commits, PRs
* ✅ La arquitectura es escalable (dockerizar después es un cambio localizado)

### Cuándo SÍ dockerizar (futuro):

* Múltiples pipelines en paralelo sin interferencias
* Reproducibilidad exacta (versiones específicas de Node/Python/etc.)
* Levantar servicios complejos (docker-compose)
* Distribuir ejecución en múltiples máquinas

Para el MVP, estas necesidades NO existen.

---

## 🎮 **Comandos de la app (REPL interno)**

La app se ejecuta con `agent666` y expone un **prompt interactivo interno** con los siguientes comandos:

### **Inicio de la app:**

```bash
# Usuario ejecuta (solo una vez)
$ agent666

# La app se inicia y muestra el prompt:
🤖 agent666> _
```

### **Comandos internos disponibles:**

1. **`add-repo [path]`**

   * Añade un repositorio a la cola de trabajo
   * Pregunta interactivamente:

     * ¿GitHub habilitado? (s/n)
     * ¿Límite de intentos en BUCLE? (default: 10)
     * ¿Que comportamiento ha de tener el agente? (default--vacio--|sin comportamiento)
       * --si selecciona default--
       * ¿Quieres añadir especificaciones de comportamiento? (no--vacio--|config|esp)
   * Guarda configuración en `.agent666rc.json`
   * El watcher comienza a monitorear `docs/task/<issue>.md` en ese repo

2. **`rm-repo [path]`**

   * Elimina un repositorio de la cola de trabajo
   * Detiene el watcher de ese repo
   * No elimina archivos, solo lo quita de la configuración

3. **`list`**

   * Lista todos los repositorios añadidos
   * Muestra configuración de cada uno: path, GitHub habilitado, límite de intentos, estado actual, (con Github habilitado) 'push and open PR' pendientes, comportamiento

4. **`status`**

   * Muestra estado actual de la cola de trabajo, issue en proceso y logs recientes

5. **`droid`**

   * Accede a la sesión interactiva del droid del issue actual
   * Sin argumentos (accede al droid que está trabajando)
   * Usa `node-pty` para sesión de terminal completa
   * Salir con `Ctrl+C` vuelve al prompt

6. **`exit`**

   * Sale de la app
   * Pregunta si desea detener el pipeline en proceso
   * Cierra watchers y procesos del droid
   * Guarda estado en `.agent666rc.json`



---

## 🎬 **Resumen ejecutivo**

Super Agente MVP es un **orquestador automático con prompt interactivo** que:

1. Se ejecuta con `agent666` y mantiene un **prompt interno (REPL)**
2. El usuario añade repositorios vía comandos internos (`add-repo`, `rm-repo`, `list`)
3. **Detecta automáticamente** `docs/task/<issue>.md` en repos añadidos
4. **Procesa la cola secuencialmente** sin intervención manual:

   * Crea branch dedicada `agent/issue-<id>`
   * Ejecuta agente NATIVO (4 fases: PRE → BUCLE → VALIDACIÓN → POST)
   * Hace commit automáticamente
5. **Muestra mensajes en tiempo real**: "Trabajando en issue #123..."
6. El usuario puede **acceder al droid** en cualquier momento con comando `droid`
7. Post-pipeline (si GitHub habilitado): flujo de decisión (push y PR, abrir droid, o terminar sin push/PR)
8. **Continúa con la siguiente tarea** de la cola automáticamente

**Características clave:**

* ✅ App persistente/interactiva
* ✅ Sistema de cola automático
* ✅ File watcher
* ✅ Acceso al droid en tiempo real
* ✅ Control total del usuario
* ❌ Sin base de datos compleja (solo JSON)
* ❌ Sin Docker (agente nativo en host)
* ❌ Procesamiento secuencial

La arquitectura es **simple para validar rápido**, diseñada para escalar progresivamente añadiendo capas de complejidad solo cuando se necesiten.
