# Reporte Descriptivo: Arquitectura y Servicios del Sistema Agent666

## 📋 Índice
1. [Resumen Ejecutivo](#resumen-ejecutivo)
2. [Arquitectura General](#arquitectura-general)
3. [Infraestructura y Contenedores](#infraestructura-y-contenedores)
4. [Servicios Detallados](#servicios-detallados)
5. [Flujos de Interacción](#flujos-de-interacción)
6. [Bases de Datos y Persistencia](#bases-de-datos-y-persistencia)
7. [Patrones de Diseño y Tecnologías](#patrones-de-diseño-y-tecnologías)

---

## 🎯 Resumen Ejecutivo

El sistema **Agent666** es una plataforma de gestión de tareas automatizada diseñada para procesar issues de GitHub de manera inteligente y distribuida. El sistema se compone de 5 servicios principales que se comunican a través de **NATS JetStream** (sistema de mensajería distribuido) y utilizan **MongoDB** para persistencia de datos.

### Componentes Principales:
- **app-go**: API REST para consultar información de GitHub
- **queue-go**: API REST para gestión de cola de tareas
- **queue-worker-go**: Worker que consume y procesa tareas
- **agent-intel-go**: Motor de inteligencia para priorización de tareas
- **NATS**: Message broker para comunicación asíncrona
- **MongoDB**: Base de datos para persistencia

---

## 🏗️ Arquitectura General

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        SISTEMA AGENT666                                  │
│                                                                          │
│  ┌────────────────┐         ┌──────────────┐         ┌───────────────┐ │
│  │  Orchestrator  │────────▶│     NATS     │────────▶│ Agent Intel   │ │
│  │  (Externo)     │         │  JetStream   │         │   Service     │ │
│  │                │         │   :4222      │         │    :8082      │ │
│  └────────────────┘         └──────────────┘         └───────────────┘ │
│         │                          │                         │          │
│         │                          │                         ▼          │
│         │                          │                   ┌──────────┐    │
│         │                          │                   │ MongoDB  │    │
│         │                          │                   │  :27017  │    │
│         │                          │                   └──────────┘    │
│         │                          │                                    │
│         │                          ▼                                    │
│         │                   ┌──────────────┐                           │
│         │                   │  queue-go    │                           │
│         │                   │  (Queue API) │                           │
│         │                   │   :8081      │                           │
│         │                   └──────────────┘                           │
│         │                          │                                    │
│         │                          ▼                                    │
│         │                   ┌──────────────┐                           │
│         │                   │queue-worker  │                           │
│         │                   │   (Consumer) │                           │
│         │                   └──────────────┘                           │
│         │                                                               │
│         ▼                                                               │
│  ┌──────────────┐                                                      │
│  │   app-go     │                                                      │
│  │ (GitHub API) │                                                      │
│  │   :8083      │                                                      │
│  └──────────────┘                                                      │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

### Flujo de Comunicación:
1. **Orquestador externo** publica eventos de nuevas tareas en NATS
2. **agent-intel-go** consume eventos y almacena tareas en MongoDB
3. **Orquestador** consulta tareas prioritarias desde agent-intel-go
4. **queue-go** gestiona la cola de tareas (backward compatibility)
5. **queue-worker-go** procesa tareas desde NATS
6. **app-go** proporciona datos de GitHub cuando se necesitan

---

## 🐳 Infraestructura y Contenedores

### Docker Compose - Configuración Completa

El sistema utiliza **Docker Compose** para orquestar todos los servicios. Cada servicio se ejecuta en su propio contenedor y se comunica a través de una red compartida llamada `agent666-network`.

#### Contenedores Desplegados:

| Contenedor | Imagen | Puerto Expuesto | Propósito |
|-----------|--------|----------------|-----------|
| `agent666-nats` | nats:latest | 4222 (client)<br>8222 (monitoring)<br>6222 (cluster) | Message broker con JetStream |
| `agent666-app-go` | app-go:custom | 8083 → 8080 | API REST para GitHub |
| `agent666-mongodb` | mongo:7.0 | 27017 | Base de datos NoSQL |
| `agent666-agent-intel-go` | agent-intel-go:custom | 8082 | Motor de inteligencia y priorización |
| `agent666-queue-go` | queue-go:custom | 8081 | API REST de cola de tareas |
| `agent666-queue-worker-go` | queue-worker-go:custom | N/A | Worker en background |

#### Volúmenes Persistentes:

```yaml
volumes:
  nats-data:     # Almacena mensajes y streams de NATS
  mongodb_data:  # Almacena colecciones de MongoDB
```

Estos volúmenes garantizan que los datos persisten incluso si los contenedores se reinician.

#### Red Docker:

```yaml
networks:
  agent666-network:
    driver: bridge
```

Todos los servicios se comunican internamente usando nombres de contenedores como hostnames (ej: `mongodb:27017`, `nats:4222`).

---

## 📦 Servicios Detallados

### 1. 🤖 agent-intel-go - Motor de Inteligencia

**Puerto**: `8082`  
**Tecnología**: Go + MongoDB + NATS JetStream  
**Propósito**: Cerebro del sistema - gestiona priorización inteligente de tareas

#### Arquitectura Interna:

```
┌───────────────────────────────────────────────────────────────┐
│                    Agent Intel Service                        │
│                                                               │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐  │
│  │   Consumer   │    │   Handlers   │    │   Scoring    │  │
│  │  (consumer.go)│    │ (handlers.go)│    │ (scoring.go) │  │
│  │              │    │              │    │              │  │
│  │ - Events     │    │ - REST API   │    │ - Age (35%)  │  │
│  │ - MongoDB    │    │ - Metrics    │    │ - Activity(25%)│ │
│  │ - Idempotency│    │ - Status     │    │ - Runtime(20%)│ │
│  └──────────────┘    └──────────────┘    │ - Load (10%) │  │
│                                           │ - Size (10%) │  │
│                                           └──────────────┘  │
│                                                               │
│  ┌──────────────────────────────────────────────────────┐  │
│  │              MongoDB Collections                      │  │
│  │  - pending_tasks:  Tareas activas con métricas       │  │
│  │  - task_history:   Tareas completadas/fallidas       │  │
│  └──────────────────────────────────────────────────────┘  │
└───────────────────────────────────────────────────────────────┘
```

#### Funcionalidades Clave:

##### 📥 Event Consumer (consumer.go)
Consume 2 tipos de eventos de NATS:

**1. `agent.task.new`** - Nueva tarea creada
- Parsea el evento JSON con datos de la tarea
- Verifica idempotencia (evita duplicados)
- Calcula métricas iniciales del repositorio:
  - Número de tareas pendientes
  - Última ejecución exitosa
  - Runtime promedio histórico
- Inserta en colección `pending_tasks`

**2. `agent.pipeline.completed`** - Pipeline finalizado
- Actualiza runtime real de la tarea
- Mueve tarea de `pending_tasks` a `task_history`
- Recalcula métricas del repositorio
- Actualiza promedios de runtime para futuras priorizaciones

##### 🎯 Motor de Priorización (scoring.go)

Calcula un **score de 0 a 1** usando 5 métricas ponderadas:

| Métrica | Peso | Descripción | Cálculo |
|---------|------|-------------|---------|
| **Age** | 35% | Antigüedad de la tarea | Más antigua = más prioridad |
| **Recent Activity** | 25% | Actividad reciente del repo | Repo activo = más prioridad |
| **Runtime** | 20% | Tiempo promedio de ejecución | Más rápido = más prioridad |
| **Load** | 10% | Carga actual del repositorio | Menos tareas = más prioridad |
| **Size** | 10% | Tamaño del archivo de tarea | Más pequeño = más prioridad |

**Fórmula:**
```
Score = (Age × 0.35) + (Activity × 0.25) + (Runtime × 0.20) + (Load × 0.10) + (Size × 0.10)
```

**Ejemplo Real:**
```
Tarea A:
- Age: 2 días (0.85)
- Activity: Última ejecución hace 1 hora (0.95)
- Runtime: 5 segundos (0.99)
- Load: 1 tarea pendiente (0.90)
- Size: 50KB (0.95)
→ Score Final: 0.91 ✅ ALTA PRIORIDAD

Tarea B:
- Age: 1 hora (0.15)
- Activity: Última ejecución hace 7 días (0.0)
- Runtime: 30 minutos (0.50)
- Load: 10 tareas pendientes (0.0)
- Size: 500KB (0.50)
→ Score Final: 0.19 ❌ BAJA PRIORIDAD
```

##### 🌐 API REST (handlers.go)

| Endpoint | Método | Descripción |
|----------|--------|-------------|
| `/health` | GET | Health check (verifica MongoDB + NATS) |
| `/api/v1/queue/next` | GET | Obtiene tarea con mayor prioridad |
| `/api/v1/queue/status` | GET | Estadísticas de la cola |
| `/api/v1/metrics` | GET | Métricas del sistema |
| `/api/v1/tasks/cancel` | POST | Cancela una tarea pendiente |

**Ejemplo de respuesta `/api/v1/queue/next`:**
```json
{
  "task": {
    "task_id": "task-abc123",
    "issue_id": "456",
    "repository": "/user/repo",
    "task_file_path": "/user/repo/docs/task/456-task.md",
    "created_at": "2025-10-19T12:00:00Z",
    "status": "pending",
    "avg_runtime_ms": 15000,
    "pending_tasks_count": 3
  },
  "score": 0.87
}
```

#### Variables de Entorno:

```bash
PORT=8082                                # Puerto del servidor HTTP
MONGO_URL=mongodb://mongodb:27017       # URL de MongoDB
NATS_URL=nats://nats:4222               # URL de NATS
DB_NAME=agent_intel                      # Nombre de la base de datos
```

---

### 2. 📦 queue-go - API de Cola de Tareas

**Puerto**: `8081`  
**Tecnología**: Go + NATS JetStream  
**Propósito**: API REST para gestión CRUD de tareas + publicación a NATS

#### Arquitectura Interna:

```
┌─────────────────────────────────────────────────────────┐
│                    Queue Service                        │
│                                                         │
│  ┌──────────────┐    ┌──────────────┐                 │
│  │   Handlers   │    │  NATS Client │                 │
│  │              │    │              │                 │
│  │ - Create     │───▶│ - Publisher  │                 │
│  │ - List       │    │ - Stream:    │                 │
│  │ - Update     │    │   TASKS      │                 │
│  │ - Delete     │    │ - Subjects:  │                 │
│  │ - Status     │    │   tasks.*    │                 │
│  └──────────────┘    └──────────────┘                 │
│                                                         │
│  ┌──────────────────────────────────┐                 │
│  │    In-Memory Queue (Cache)       │                 │
│  │  - Backward compatibility        │                 │
│  │  - Fast local access             │                 │
│  └──────────────────────────────────┘                 │
└─────────────────────────────────────────────────────────┘
```

#### Funcionalidades Principales:

##### 📡 Publicación a NATS

Cuando se crea o actualiza una tarea, el servicio:
1. Guarda en caché local (in-memory)
2. **Publica mensaje a NATS JetStream**:
   - Stream: `TASKS`
   - Subjects:
     - `tasks.new` - Nueva tarea creada
     - `tasks.update` - Tarea actualizada
     - `tasks.delete` - Tarea eliminada
     - `tasks.status` - Cambio de estado

##### 🔌 API Endpoints:

| Endpoint | Método | Descripción | Publica a NATS |
|----------|--------|-------------|----------------|
| `/health` | GET | Health check | ❌ No |
| `/api/queue/status` | GET | Estadísticas de la cola | ❌ No |
| `/api/tasks` | GET | Listar todas las tareas | ❌ No |
| `/api/tasks` | POST | Crear nueva tarea | ✅ Sí → `tasks.new` |
| `/api/tasks/{id}` | GET | Obtener tarea por ID | ❌ No |
| `/api/tasks/{id}/status` | PATCH | Actualizar estado | ✅ Sí → `tasks.status` |
| `/api/tasks/{id}` | DELETE | Eliminar tarea | ✅ Sí → `tasks.delete` |

##### 📊 Estructura de Tarea:

```go
type Task struct {
    ID           string    `json:"id"`           // UUID generado
    IssueID      string    `json:"issue_id"`     // ID del issue de GitHub
    Repository   string    `json:"repository"`   // Ruta del repositorio
    TaskFilePath string    `json:"task_file_path"` // Ruta del archivo de tarea
    Status       string    `json:"status"`       // pending/in_progress/completed/failed
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
    ErrorMessage string    `json:"error_message,omitempty"`
}
```

#### Estados de Tarea:

```
pending ──▶ in_progress ──▶ completed
                │
                └──────────▶ failed
```

---

### 3. ⚙️ queue-worker-go - Procesador de Tareas

**Puerto**: N/A (servicio background)  
**Tecnología**: Go + NATS JetStream  
**Propósito**: Consumir y procesar tareas de la cola NATS

#### Arquitectura Interna:

```
┌─────────────────────────────────────────────────────┐
│              Queue Worker Service                   │
│                                                     │
│  ┌───────────────────────────────────────────┐    │
│  │       NATS Pull Subscriber                │    │
│  │                                           │    │
│  │  Consumer: task-workers (durable)        │    │
│  │  Stream:   TASKS                         │    │
│  │  Subject:  tasks.new                     │    │
│  │                                           │    │
│  │  ┌─────────────────────────────────┐    │    │
│  │  │     Processing Pipeline         │    │    │
│  │  │                                 │    │    │
│  │  │  1. Fetch messages (batch 10)  │    │    │
│  │  │  2. Process task                │    │    │
│  │  │  3. Execute pipeline            │    │    │
│  │  │  4. ACK/NAK message            │    │    │
│  │  │  5. Retry on failure (max 3)   │    │    │
│  │  └─────────────────────────────────┘    │    │
│  └───────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────┘
```

#### Características Clave:

##### 🔄 Consumer Durable
- **Nombre**: `task-workers`
- **Tipo**: Pull Subscriber (fetch bajo demanda)
- **Batch Size**: 10 mensajes por fetch
- **Ack Policy**: Manual (explícito)
- **Max Deliver**: 3 intentos
- **Ack Wait**: 30 segundos

##### ⚡ Procesamiento de Tareas

```go
// Pseudo-código del flujo
for {
    messages := sub.Fetch(10, timeout=5s)
    
    for msg in messages {
        task := parseTask(msg.Data)
        
        // Ejecutar pipeline (simulado en MVP)
        err := processTask(task)
        
        if err != nil {
            msg.Nak()  // Reintenta hasta 3 veces
        } else {
            msg.Ack()  // Marca como procesado
        }
    }
}
```

##### 🔌 Reconexión Automática

El worker incluye lógica de reconexión:
- **Max Reconnects**: 10 intentos
- **Reconnect Wait**: 2 segundos entre intentos
- **Handlers**: Logs automáticos de desconexión/reconexión

##### 📝 Logs de Procesamiento

```
2025/10/19 15:18:09 Queue Worker starting...
2025/10/19 15:18:09 Successfully connected to NATS
2025/10/19 15:18:09 Waiting for tasks...
2025/10/19 15:18:57 Received task: {"id":"abc-123",...}
2025/10/19 15:18:58 Task processed successfully
```

---

### 4. 🐙 app-go - GitHub API Service

**Puerto**: `8083` (interno: 8080)  
**Tecnología**: Go + GitHub REST API v3  
**Propósito**: Proxy optimizado para consultas a GitHub

#### Arquitectura Interna:

```
┌───────────────────────────────────────────────────────────┐
│                  GitHub API Service                       │
│                                                           │
│  ┌──────────────┐    ┌──────────────┐    ┌────────────┐ │
│  │   Handlers   │    │    Cache     │    │  HTTP      │ │
│  │              │    │              │    │  Client    │ │
│  │ - Issues     │───▶│ TTL: 5 min   │───▶│  Pooled    │ │
│  │ - PRs        │    │ In-Memory    │    │  Gzip      │ │
│  │ - Health     │    └──────────────┘    │  KeepAlive │ │
│  └──────────────┘                        └────────────┘ │
│                                                │          │
│                                                ▼          │
│                                        ┌──────────────┐  │
│                                        │   GitHub     │  │
│                                        │   API v3     │  │
│                                        └──────────────┘  │
└───────────────────────────────────────────────────────────┘
```

#### Endpoints Disponibles:

| Endpoint | Descripción | Query Params |
|----------|-------------|--------------|
| `/` | Mensaje de bienvenida | - |
| `/health` | Health check | - |
| `/issues/{user}` | Issues del usuario | `?q=open` |
| `/pr/{user}` | Pull requests | `?q=open` |

#### Optimizaciones de Performance:

##### 1. **Connection Pooling**
```go
Transport: &http.Transport{
    MaxIdleConns:        100,  // Conexiones inactivas totales
    MaxIdleConnsPerHost: 20,   // Conexiones por host
    IdleConnTimeout:     90s,  // Timeout de inactividad
    DisableKeepAlives:   false // Keep-Alive habilitado
}
```

##### 2. **Caché en Memoria**
- TTL: 5 minutos
- Thread-safe con `sync.RWMutex`
- Reduce llamadas a GitHub API

##### 3. **Compresión Gzip**
- Accept-Encoding: gzip
- Reduce tamaño de respuestas hasta 70%

##### 4. **Rate Limiting**

| Con Token | Sin Token |
|-----------|-----------|
| 5000 req/h | 60 req/h |

**Variable de entorno:**
```bash
GITHUB_TOKEN=ghp_xxxxxxxxxxxxx  # Personal Access Token
```

#### Ejemplo de Respuesta `/issues/{user}`:

```json
[
  {
    "name": "test-agente666",
    "full_name": "SKRTEEEEEE/test-agente666",
    "url": "https://github.com/SKRTEEEEEE/test-agente666",
    "description": "Test repository for Agent666",
    "stars": 5,
    "forks": 2,
    "issues": [
      {
        "number": 16,
        "title": "Implement Agent Intel Service",
        "state": "open",
        "html_url": "https://github.com/SKRTEEEEEE/test-agente666/issues/16",
        "created_at": "2025-10-19T10:00:00Z",
        "updated_at": "2025-10-19T15:00:00Z",
        "user": {
          "login": "SKRTEEEEEE"
        }
      }
    ]
  }
]
```

---

### 5. 📨 NATS JetStream - Message Broker

**Puertos**: 
- `4222` - Cliente
- `8222` - Monitoring HTTP
- `6222` - Cluster routing

**Tecnología**: NATS Server 2.x con JetStream habilitado  
**Propósito**: Sistema de mensajería distribuido con garantías de entrega

#### Configuración del Stream:

##### Stream: `TASKS`
```yaml
Name: TASKS
Subjects: [tasks.*]
Retention: WorkQueue      # Elimina mensajes después de ACK
Storage: File             # Persistencia en disco
MaxAge: 7 días           # Máximo 7 días de retención
Replicas: 1              # Sin replicación (dev)
```

##### Subjects (Temas):
```
tasks.new         → Nueva tarea creada (queue-go → queue-worker-go)
tasks.update      → Tarea actualizada
tasks.delete      → Tarea eliminada
tasks.status      → Cambio de estado
```

##### Stream: `AGENT`
```yaml
Name: AGENT
Subjects: [agent.*]
Storage: File
MaxAge: 7 días
Replicas: 1
```

##### Subjects:
```
agent.task.new              → Nueva tarea (orchestrator → agent-intel-go)
agent.pipeline.completed    → Pipeline finalizado (orchestrator → agent-intel-go)
```

#### Consumers Configurados:

| Consumer | Stream | Subject | Durable | Max Deliver | Ack Wait |
|----------|--------|---------|---------|-------------|----------|
| `task-workers` | TASKS | tasks.new | ✅ Yes | 3 | 30s |
| `agent-intel-task-new` | AGENT | agent.task.new | ✅ Yes | 3 | 30s |
| `agent-intel-pipeline-completed` | AGENT | agent.pipeline.completed | ✅ Yes | 3 | 30s |

#### Monitoring Endpoints:

| URL | Descripción |
|-----|-------------|
| http://localhost:8222/healthz | Health check |
| http://localhost:8222/varz | Server info |
| http://localhost:8222/connz | Conexiones activas |
| http://localhost:8222/jsz | JetStream info |

---

### 6. 🗄️ MongoDB - Base de Datos

**Puerto**: `27017`  
**Versión**: mongo:7.0  
**Propósito**: Persistencia de tareas y métricas

#### Base de Datos: `agent_intel`

##### Colección: `pending_tasks`

Tareas activas en el sistema:

```json
{
  "_id": ObjectId("..."),
  "task_id": "task-abc123",         // PK - Índice único
  "issue_id": "456",
  "repository": "/user/repo",       // Índice
  "task_file_path": "/user/repo/docs/task/456-task.md",
  "created_at": ISODate("2025-10-19T12:00:00Z"),
  "last_success_at": ISODate("2025-10-18T10:00:00Z"),
  "avg_runtime_ms": 15000,
  "pending_tasks_count": 3,
  "size_bytes": 2048,
  "status": "pending",              // Índice
  "assigned_at": null
}
```

**Índices creados:**
```javascript
db.pending_tasks.createIndex({ task_id: 1 }, { unique: true })
db.pending_tasks.createIndex({ repository: 1 })
db.pending_tasks.createIndex({ status: 1 })
db.pending_tasks.createIndex({ created_at: -1 })
```

##### Colección: `task_history`

Historial de tareas completadas/fallidas:

```json
{
  "_id": ObjectId("..."),
  "task_id": "task-abc123",         // PK - Índice único
  "issue_id": "456",
  "repository": "/user/repo",
  "task_file_path": "/user/repo/docs/task/456-task.md",
  "status": "completed",            // Índice compuesto
  "pipeline_runtime_ms": 14500,
  "created_at": ISODate("2025-10-19T12:00:00Z"),
  "assigned_at": ISODate("2025-10-19T12:05:00Z"),
  "error_message": null
}
```

**Índices creados:**
```javascript
db.task_history.createIndex({ task_id: 1 }, { unique: true })
db.task_history.createIndex({ repository: 1, status: 1 })
```

#### Volume Persistente:

```yaml
volumes:
  mongodb_data: /data/db
```

Los datos persisten incluso si el contenedor se destruye.

---

## 🔄 Flujos de Interacción

### Flujo 1: Creación de Nueva Tarea

```
┌─────────────┐
│ Orchestrator│
└──────┬──────┘
       │
       │ 1. Publish event: agent.task.new
       │    {task_id, issue_id, repository, size_bytes}
       ▼
┌─────────────┐
│    NATS     │ Stream: AGENT
│ JetStream   │ Subject: agent.task.new
└──────┬──────┘
       │
       │ 2. Consume event
       ▼
┌─────────────┐
│ agent-intel │
│   Service   │
└──────┬──────┘
       │
       │ 3. Check idempotency (evita duplicados)
       │ 4. Calculate metrics:
       │    - Pending count del repo
       │    - Last success timestamp
       │    - Avg runtime histórico
       │
       │ 5. Insert into pending_tasks
       ▼
┌─────────────┐
│   MongoDB   │
│pending_tasks│
└─────────────┘
```

**Detalles Técnicos:**

1. **Publicación**:
```javascript
// Orchestrator publica
nats.publish("agent.task.new", {
  "task_id": "task-abc123",
  "issue_id": "456",
  "repository": "/user/repo",
  "task_file_path": "/user/repo/docs/task/456-task.md",
  "size_bytes": 2048,
  "created_at": "2025-10-19T12:00:00Z"
})
```

2. **Consumo (agent-intel-go)**:
- Consumer durable: `agent-intel-task-new`
- Ack manual después de insertar en MongoDB
- Si falla la inserción, hace NAK para reintentar

3. **Idempotencia**:
```go
// Verifica si ya existe
count := pendingCol.CountDocuments({task_id: event.TaskID})
if count > 0 {
    log.Println("Task already exists, skipping")
    msg.Ack()  // ACK sin procesar
    return
}
```

---

### Flujo 2: Obtención de Tarea Prioritaria

```
┌─────────────┐
│ Orchestrator│
└──────┬──────┘
       │
       │ 1. GET /api/v1/queue/next
       ▼
┌─────────────┐
│ agent-intel │
│   Service   │
└──────┬──────┘
       │
       │ 2. Query MongoDB: find pending tasks
       │ 3. Calculate score for each task:
       │    Score = (Age × 0.35) + (Activity × 0.25) + 
       │            (Runtime × 0.20) + (Load × 0.10) + 
       │            (Size × 0.10)
       │ 4. Sort by score DESC
       │ 5. Return task with highest score
       │
       ▼
┌─────────────┐
│   MongoDB   │
│pending_tasks│
└─────────────┘
```

**Ejemplo de Cálculo de Score:**

```javascript
// Tarea en MongoDB
{
  "task_id": "task-abc123",
  "created_at": "2025-10-17T12:00:00Z",  // Hace 2 días
  "last_success_at": "2025-10-19T10:00:00Z",  // Hace 2 horas
  "avg_runtime_ms": 5000,  // 5 segundos
  "pending_tasks_count": 1,  // 1 tarea en cola
  "size_bytes": 51200  // 50KB
}

// Cálculo de métricas normalizadas (0-1):
age_score = (48 hours / 168 max hours) = 0.28
activity_score = 1.0 - (2 hours / 168 max hours) = 0.98
runtime_score = 1.0 - (5000ms / 2400000 max ms) = 0.998
load_score = 1.0 - (1 / 10 max) = 0.90
size_score = 1.0 - (51200 / 1048576 max) = 0.95

// Score final:
total_score = (0.28 × 0.35) + (0.98 × 0.25) + (0.998 × 0.20) + (0.90 × 0.10) + (0.95 × 0.10)
            = 0.098 + 0.245 + 0.199 + 0.090 + 0.095
            = 0.727  ← PRIORIDAD MEDIA-ALTA
```

---

### Flujo 3: Completación de Pipeline

```
┌─────────────┐
│ Orchestrator│
└──────┬──────┘
       │
       │ 1. Execute pipeline
       │ 2. Measure runtime
       │ 3. Publish event: agent.pipeline.completed
       │    {task_id, status, pipeline_runtime_ms}
       ▼
┌─────────────┐
│    NATS     │ Stream: AGENT
│ JetStream   │ Subject: agent.pipeline.completed
└──────┬──────┘
       │
       │ 4. Consume event
       ▼
┌─────────────┐
│ agent-intel │
│   Service   │
└──────┬──────┘
       │
       │ 5. Find task in pending_tasks
       │ 6. Update task status (completed/failed)
       │ 7. Move to task_history
       │ 8. Delete from pending_tasks
       │ 9. Recalculate metrics for repository:
       │    - New avg_runtime (last 10 tasks)
       │    - Update last_success_at
       │    - Update pending_tasks_count
       │
       ▼
┌─────────────┐   ┌─────────────┐
│   MongoDB   │   │   MongoDB   │
│ task_history│   │pending_tasks│
└─────────────┘   └─────────────┘
```

**Código de Actualización de Métricas:**

```go
// Calculate avg runtime from last 10 completed tasks
pipeline := mongo.Pipeline{
    {{"$match", bson.M{
        "repository": repository,
        "status": "completed",
        "pipeline_runtime_ms": bson.M{"$gt": 0},
    }}},
    {{"$sort", bson.M{"completed_at": -1}}},
    {{"$limit", 10}},
    {{"$group", bson.M{
        "_id": nil,
        "avg_runtime": bson.M{"$avg": "$pipeline_runtime_ms"},
    }}},
}

// Update all pending tasks for this repo
pendingCol.UpdateMany(
    bson.M{"repository": repository},
    bson.M{"$set": bson.M{
        "avg_runtime_ms": avgRuntime,
        "last_success_at": lastSuccessAt,
        "pending_tasks_count": pendingCount,
    }},
)
```

---

### Flujo 4: Procesamiento de Worker (Legacy)

```
┌─────────────┐
│  queue-go   │
│  (API REST) │
└──────┬──────┘
       │
       │ 1. POST /api/tasks
       │    {issue_id, repository, task_file_path}
       ▼
┌─────────────┐
│ In-Memory   │
│   Queue     │
└──────┬──────┘
       │
       │ 2. Publish to NATS
       │    Subject: tasks.new
       ▼
┌─────────────┐
│    NATS     │ Stream: TASKS
│ JetStream   │ Subject: tasks.new
└──────┬──────┘
       │
       │ 3. Pull Subscribe
       ▼
┌─────────────┐
│queue-worker │
│     go      │
└──────┬──────┘
       │
       │ 4. Fetch batch (10 messages)
       │ 5. Process each task
       │ 6. ACK message
       │
       └────▶ [Task Processing Pipeline]
```

**Este flujo es legacy** - el sistema moderno usa `agent-intel-go` como fuente de verdad.

---

### Flujo 5: Consulta de GitHub Issues

```
┌─────────────┐
│ Orchestrator│
│   / Client  │
└──────┬──────┘
       │
       │ 1. GET /issues/{user}?q=open
       ▼
┌─────────────┐
│   app-go    │
│ (Port 8083) │
└──────┬──────┘
       │
       │ 2. Check cache (TTL: 5 min)
       │
       ├──▶ Cache HIT ──▶ Return cached data
       │
       └──▶ Cache MISS
              │
              │ 3. Fetch user repos
              │    GET api.github.com/users/{user}/repos
              │
              │ 4. For each repo (parallel):
              │    GET api.github.com/repos/{owner}/{repo}/issues
              │
              │ 5. Aggregate results
              │ 6. Store in cache
              │ 7. Return JSON
              ▼
┌─────────────┐
│   GitHub    │
│   API v3    │
└─────────────┘
```

**Optimizaciones:**
- **Concurrent requests**: Usa goroutines para paralelizar llamadas
- **Connection pooling**: Reutiliza conexiones HTTP
- **Gzip compression**: Reduce bandwidth hasta 70%
- **Rate limit handling**: Retry automático con backoff

---

## 🗃️ Bases de Datos y Persistencia

### MongoDB - Esquemas Detallados

#### Colección: `pending_tasks`

**Propósito**: Almacenar tareas activas que necesitan ser procesadas

**Schema:**
```javascript
{
  _id: ObjectId,                    // MongoDB auto-generated
  task_id: String,                  // UUID único
  issue_id: String,                 // ID del issue de GitHub
  repository: String,               // Path del repositorio
  task_file_path: String,           // Ruta al archivo .md de la tarea
  created_at: ISODate,              // Timestamp de creación
  last_success_at: ISODate | null,  // Última ejecución exitosa del repo
  avg_runtime_ms: Number,           // Promedio de runtime del repo
  pending_tasks_count: Number,      // Número de tareas pendientes del repo
  size_bytes: Number,               // Tamaño del archivo de tarea
  pipeline_runtime_ms: Number | null, // Runtime de esta ejecución
  assigned_at: ISODate | null,      // Cuándo se asignó la tarea
  status: String,                   // pending|assigned|processing|completed|failed|cancelled
  error_message: String | null,     // Mensaje de error si falló
  cancel_reason: String | null      // Razón de cancelación
}
```

**Índices:**
```javascript
// Performance: búsqueda rápida por task_id
db.pending_tasks.createIndex({ task_id: 1 }, { unique: true })

// Performance: filtrar por repositorio
db.pending_tasks.createIndex({ repository: 1 })

// Performance: filtrar por estado
db.pending_tasks.createIndex({ status: 1 })

// Performance: ordenar por antigüedad
db.pending_tasks.createIndex({ created_at: -1 })
```

**Queries comunes:**
```javascript
// Obtener tarea con mayor prioridad
db.pending_tasks.find({ status: "pending" })
  .sort({ created_at: -1 })  // Más antigua primero
  .limit(1)

// Contar tareas pendientes de un repo
db.pending_tasks.countDocuments({
  repository: "/user/repo",
  status: { $in: ["pending", "assigned", "processing"] }
})
```

#### Colección: `task_history`

**Propósito**: Historial de tareas completadas para métricas y análisis

**Schema:**
```javascript
{
  _id: ObjectId,
  task_id: String,                  // UUID único
  issue_id: String,
  repository: String,
  task_file_path: String,
  status: String,                   // completed|failed
  pipeline_runtime_ms: Number,      // Runtime real de ejecución
  created_at: ISODate,              // Cuándo se creó
  assigned_at: ISODate,             // Cuándo se asignó
  completed_at: ISODate,            // Cuándo finalizó (auto-generated)
  error_message: String | null
}
```

**Índices:**
```javascript
db.task_history.createIndex({ task_id: 1 }, { unique: true })
db.task_history.createIndex({ repository: 1, status: 1 })
```

**Queries comunes:**
```javascript
// Calcular promedio de runtime de últimas 10 tareas
db.task_history.aggregate([
  { $match: { 
      repository: "/user/repo",
      status: "completed",
      pipeline_runtime_ms: { $gt: 0 }
  }},
  { $sort: { completed_at: -1 }},
  { $limit: 10 },
  { $group: {
      _id: null,
      avg_runtime: { $avg: "$pipeline_runtime_ms" }
  }}
])

// Obtener última ejecución exitosa del repo
db.task_history.findOne(
  { repository: "/user/repo", status: "completed" },
  { sort: { completed_at: -1 }}
)
```

---

### NATS JetStream - Streams y Mensajes

#### Stream: `AGENT`

**Configuración:**
```javascript
{
  name: "AGENT",
  subjects: ["agent.*"],
  retention: "limits",          // Retiene por tiempo
  max_age: 604800000000000,    // 7 días en nanosegundos
  storage: "file",              // Persistido en disco
  replicas: 1,
  discard: "old"                // Elimina mensajes viejos
}
```

**Subject: `agent.task.new`**

Mensaje publicado cuando se crea una nueva tarea:

```json
{
  "task_id": "task-abc123",
  "issue_id": "456",
  "repository": "/user/repo",
  "task_file_path": "/user/repo/docs/task/456-task.md",
  "size_bytes": 2048,
  "created_at": "2025-10-19T12:00:00Z"
}
```

**Subject: `agent.pipeline.completed`**

Mensaje publicado cuando finaliza un pipeline:

```json
{
  "task_id": "task-abc123",
  "repository": "/user/repo",
  "pipeline_runtime_ms": 14500,
  "status": "success",          // "success" | "failure"
  "completed_at": "2025-10-19T12:05:00Z",
  "error_message": null
}
```

#### Stream: `TASKS`

**Configuración:**
```javascript
{
  name: "TASKS",
  subjects: ["tasks.*"],
  retention: "workqueue",       // Elimina después de ACK
  max_age: 604800000000000,
  storage: "file",
  replicas: 1
}
```

**Subject: `tasks.new`**

Mensaje de nueva tarea (legacy):

```json
{
  "id": "abc-123",
  "issue_id": "456",
  "repository": "/user/repo",
  "task_file_path": "/user/repo/docs/task/456-task.md",
  "status": "pending",
  "created_at": "2025-10-19T12:00:00Z"
}
```

---

## 🧩 Patrones de Diseño y Tecnologías

### Patrones de Arquitectura

#### 1. **Event-Driven Architecture (EDA)**

El sistema usa eventos asincrónicos para desacoplar servicios:

```
Publisher ──▶ NATS ──▶ Consumer
(queue-go)    (Events)  (worker-go)
```

**Ventajas:**
- ✅ Desacoplamiento: los servicios no se conocen entre sí
- ✅ Escalabilidad: múltiples workers pueden consumir eventos
- ✅ Resiliencia: si un consumer falla, los mensajes se reintentarán
- ✅ Asincronía: no bloquea el flujo principal

#### 2. **CQRS (Command Query Responsibility Segregation)**

Separación de lectura y escritura:

**Commands (Escritura):**
- `POST /api/tasks` → Crea tarea → Publica evento
- `PATCH /api/tasks/{id}/status` → Actualiza estado → Publica evento

**Queries (Lectura):**
- `GET /api/v1/queue/next` → Lee desde MongoDB (source of truth)
- `GET /api/queue/status` → Lee desde cache en memoria

#### 3. **Priority Queue Pattern**

Sistema de priorización basado en métricas múltiples:

```
Incoming Tasks ──▶ Scoring Engine ──▶ Sorted Queue ──▶ Consumer
                   (5 metrics)         (highest first)
```

#### 4. **Idempotent Consumer Pattern**

Previene procesamiento duplicado de eventos:

```go
// Check if already processed
count := collection.CountDocuments(ctx, bson.M{"task_id": taskID})
if count > 0 {
    msg.Ack()  // Already processed, skip
    return
}
```

#### 5. **Circuit Breaker Pattern**

Manejo de fallos en conexiones externas:

```go
// Retry logic con backoff exponencial
func connectWithRetry(url string, maxRetries int) (*Connection, error) {
    for i := 0; i < maxRetries; i++ {
        conn, err := connect(url)
        if err == nil {
            return conn, nil
        }
        time.Sleep(time.Duration(i) * 2 * time.Second)  // Exponential backoff
    }
    return nil, ErrMaxRetriesExceeded
}
```

---

### Tecnologías y Stack

#### Backend:
- **Lenguaje**: Go 1.21+
- **HTTP Server**: net/http (standard library)
- **Concurrency**: Goroutines + Channels

#### Messaging:
- **NATS Server**: 2.x con JetStream
- **Client Library**: github.com/nats-io/nats.go

#### Persistence:
- **MongoDB**: 7.0 (NoSQL document database)
- **Driver**: go.mongodb.org/mongo-driver

#### External APIs:
- **GitHub API v3**: REST API con autenticación via token

#### Containerización:
- **Docker**: Multi-stage builds
- **Docker Compose**: Orquestación de servicios

#### Testing:
- **Unit tests**: go test + testify
- **Integration tests**: Docker + docker-compose para ambiente de pruebas

---

### Decisiones de Diseño Clave

#### ¿Por qué NATS JetStream?

| Criterio | NATS JetStream | RabbitMQ | Kafka |
|----------|---------------|----------|-------|
| **Latencia** | < 1ms | ~10ms | ~5ms |
| **Simplicidad** | ✅ Muy simple | ⚠️ Complejo | ❌ Muy complejo |
| **Persistencia** | ✅ File storage | ✅ Disk storage | ✅ Log storage |
| **Escalabilidad** | ✅ Horizontal | ⚠️ Vertical | ✅ Horizontal |
| **Peso** | ~15MB | ~200MB | ~500MB |
| **Mantenimiento** | ✅ Mínimo | ⚠️ Medio | ❌ Alto |

**Decisión**: NATS JetStream por su simplicidad, bajo overhead y excelente performance.

#### ¿Por qué MongoDB?

| Criterio | MongoDB | PostgreSQL | Redis |
|----------|---------|-----------|-------|
| **Esquema flexible** | ✅ Sí | ❌ No | ⚠️ Limitado |
| **Queries complejas** | ✅ Sí | ✅ Sí | ❌ No |
| **Índices** | ✅ Sí | ✅ Sí | ⚠️ Limitado |
| **Persistencia** | ✅ Disk | ✅ Disk | ⚠️ Opcional |
| **Agregaciones** | ✅ Pipeline | ✅ SQL | ❌ No |

**Decisión**: MongoDB por su flexibilidad de schema y pipeline de agregación potente para calcular métricas.

#### ¿Por qué Go?

- ✅ **Performance**: Compilado, concurrente por diseño
- ✅ **Simplicidad**: Sintaxis minimalista, fácil de mantener
- ✅ **Tooling**: Testing, profiling, benchmarking built-in
- ✅ **Deployment**: Binario estático, sin dependencias
- ✅ **Librerías**: Excelentes clients para NATS, MongoDB, HTTP

---

## 📊 Métricas y Monitoreo

### Health Checks

| Servicio | Endpoint | Verifica |
|----------|----------|----------|
| app-go | `/health` | HTTP server activo |
| queue-go | `/health` | HTTP server activo |
| agent-intel-go | `/health` | MongoDB + NATS conectados |
| NATS | `http://localhost:8222/healthz` | Server activo |

### Métricas Disponibles

#### Agent Intel Service (`GET /api/v1/metrics`)

```json
{
  "total_pending": 15,
  "total_processing": 2,
  "total_completed": 143,
  "total_failed": 7,
  "avg_runtime_ms": 18500,
  "tasks_processed": 150,
  "timestamp": "2025-10-19T15:30:00Z"
}
```

#### Queue Status (`GET /api/v1/queue/status`)

```json
{
  "total_tasks": 17,
  "tasks_by_repo": {
    "/user/repo1": 5,
    "/user/repo2": 12
  },
  "tasks_by_status": {
    "pending": 15,
    "processing": 2
  },
  "timestamp": "2025-10-19T15:30:00Z"
}
```

#### NATS Monitoring (`http://localhost:8222/jsz`)

```json
{
  "streams": [
    {
      "name": "AGENT",
      "messages": 1250,
      "bytes": 102400,
      "first_seq": 1,
      "last_seq": 1250
    },
    {
      "name": "TASKS",
      "messages": 0,
      "bytes": 0
    }
  ]
}
```

---

## 🚀 Uso y Comandos

### Inicio Rápido

```bash
# Levantar todos los servicios
docker-compose up -d

# Ver logs de todos los servicios
docker-compose logs -f

# Ver logs de un servicio específico
docker-compose logs -f agent-intel-go

# Ver estado de servicios
docker-compose ps

# Detener todos los servicios
docker-compose down

# Detener y eliminar volúmenes (limpieza completa)
docker-compose down -v
```

### Pruebas de API

#### 1. Crear Nueva Tarea (Agent Intel)

```bash
# Publicar evento (desde orchestrator)
curl -X POST http://localhost:8082/api/v1/tasks/new \
  -H "Content-Type: application/json" \
  -d '{
    "task_id": "task-abc123",
    "issue_id": "456",
    "repository": "/user/repo",
    "task_file_path": "/user/repo/docs/task/456-task.md",
    "size_bytes": 2048
  }'
```

#### 2. Obtener Tarea Prioritaria

```bash
# Obtener siguiente tarea
curl http://localhost:8082/api/v1/queue/next

# Obtener para repo específico
curl "http://localhost:8082/api/v1/queue/next?repo_id=/user/repo"
```

#### 3. Consultar Issues de GitHub

```bash
# Todos los issues de un usuario
curl http://localhost:8083/issues/SKRTEEEEEE

# Solo issues abiertos
curl http://localhost:8083/issues/SKRTEEEEEE?q=open

# Pull requests
curl http://localhost:8083/pr/SKRTEEEEEE?q=open
```

#### 4. Gestión de Cola (Legacy)

```bash
# Crear tarea
curl -X POST http://localhost:8081/api/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "issue_id": "456",
    "repository": "/user/repo",
    "task_file_path": "/user/repo/docs/task/456-task.md"
  }'

# Listar tareas
curl http://localhost:8081/api/tasks

# Ver estado de la cola
curl http://localhost:8081/api/queue/status
```

---

## 🔐 Seguridad y Consideraciones

### Autenticación

- **GitHub API**: Requiere token para rate limits altos
  ```bash
  docker run -e GITHUB_TOKEN=ghp_xxx app-go
  ```

### Rate Limiting

- **GitHub API**:
  - Sin token: 60 req/hora
  - Con token: 5000 req/hora

### Network Security

- Todos los servicios en red privada `agent666-network`
- Puertos expuestos solo para desarrollo local
- En producción: usar reverse proxy (Nginx) + TLS

---

## 📈 Escalabilidad

### Escalado Horizontal

#### Queue Workers:
```bash
# Levantar 3 workers
docker-compose up -d --scale queue-worker-go=3
```

NATS distribuirá mensajes entre los 3 workers automáticamente.

#### Agent Intel Service:
- Actualmente stateless (excepto MongoDB)
- Se puede escalar con load balancer
- MongoDB soporta replica sets para HA

### Performance Esperado

| Métrica | Valor |
|---------|-------|
| Latencia p50 (API) | < 50ms |
| Latencia p99 (API) | < 200ms |
| Throughput (tasks/s) | ~100-500 |
| NATS latency | < 1ms |

---

## 🐛 Troubleshooting

### Problema: NATS no conecta

```bash
# Verificar que NATS está corriendo
docker logs agent666-nats

# Verificar health
curl http://localhost:8222/healthz

# Reiniciar NATS
docker-compose restart nats
```

### Problema: MongoDB no accesible

```bash
# Ver logs
docker logs agent666-mongodb

# Verificar conectividad
docker exec -it agent666-mongodb mongosh
> show dbs
> use agent_intel
> show collections
```

### Problema: Worker no procesa tareas

```bash
# Ver logs del worker
docker logs -f agent666-queue-worker-go

# Verificar que está suscrito
# Deberías ver: "Successfully subscribed to task queue"

# Verificar mensajes en NATS
curl http://localhost:8222/jsz?streams=true
```

---

## 📝 Conclusiones

El sistema **Agent666** implementa una arquitectura moderna y distribuida basada en:

1. **Event-Driven Architecture**: Desacoplamiento mediante NATS JetStream
2. **Intelligent Prioritization**: Motor de scoring con 5 métricas
3. **Persistence**: MongoDB para métricas e historial
4. **Resilience**: Reintentos automáticos, consumers durables, idempotencia
5. **Observability**: Health checks, métricas, logging estructurado

### Próximos Pasos Recomendados:

- [ ] Implementar autenticación JWT para APIs
- [ ] Agregar Prometheus metrics para monitoreo avanzado
- [ ] Implementar Grafana dashboards
- [ ] Agregar tracing distribuido (Jaeger/Zipkin)
- [ ] Implementar circuit breakers con resilience4j
- [ ] Agregar tests de carga con k6

---

**Generado**: 2025-10-19  
**Versión**: 1.0  
**Autor**: Agent666 System Documentation
