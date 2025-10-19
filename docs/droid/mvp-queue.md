

# 🧠 Propuesta Consolidada: MVP del Agent Intel Service

El **Agent Intel Service** es un micro-servicio independiente encargado de **gestionar la cola de tareas** y **optimizar su priorización**.
Se alimenta del historial de ejecuciones del agente y de eventos del sistema (`agent.task.new`, `agent.pipeline.completed`) para decidir **qué tarea debe ejecutarse a continuación** de forma inteligente.
Además, guarda métricas que servirán para optimizar futuras tareas en versiones posteriores.

---

## 🎯 Objetivo y Roles del Servicio

| Rol              | Objetivo                                                                                                                  |
| :--------------- | :------------------------------------------------------------------------------------------------------------------------ |
| **Aprendizaje**  | Registrar automáticamente las métricas clave de las ejecuciones para mejorar la toma de decisiones futuras.               |
| **Priorización** | Calcular dinámicamente qué tarea debe procesarse primero, basándose en datos objetivos y automáticos.                     |
| **Gobernanza**   | Centralizar la gestión de la cola y exponer una API estable para el Orquestador, asegurando consistencia en la ejecución. |

---

## 🧩 Arquitectura General de Servicios (MVP)

El MVP se compone de **cuatro servicios principales**, desplegados de forma independiente pero conectados entre sí mediante **NATS** y **API REST**.

```
┌────────────────────────────┐
│     Orquestador CLI        │
│ (Publica tareas, consulta  │
│  prioridades y resultados) │
└────────────┬───────────────┘
             │
             │ (Eventos NATS)
             ▼
     ┌───────────────────────┐
     │   NATS JetStream      │
     │ (event bus distribuido│
     │  y persistente)       │
     └──────┬────────┬──────┘
            │        │
            │        │
            ▼        ▼
┌──────────────────┐ ┌────────────────────┐
│ Agent Intel Svc  │ │      MongoDB       │
│  (Go)             │ │ (Historial + cola)│
│                   │ └────────────────────┘
│ - Gestiona cola   │
│ - Prioriza tareas │
│ - Exposición REST │
└───────────────────┘
```

---

## 🛠️ Stack Tecnológico

| Componente                | Tecnología          | Propósito                                                        |
| :------------------------ | :------------------ | :--------------------------------------------------------------- |
| **Lenguaje del Servicio** | **Go (Golang)**     | Alto rendimiento, concurrencia y excelente integración con NATS. |
| **Event Bus / Queue**     | **NATS JetStream**  | Backbone distribuido y duradero para el flujo de eventos.        |
| **Base de Datos**         | **MongoDB** (NoSQL) | Flexibilidad para almacenar el historial y estados complejos.    |
| **Infraestructura**       | **Docker Compose**  | Despliegue simple y reproducible.                                |
| **Orquestador CLI**       | **Node.js**         | Cliente que publica y consume eventos del sistema.               |

---

## 📦 Servicios del MVP

### 🧱 1. Agent Intel Service (Go)

* Consume eventos (`agent.task.new`, `agent.pipeline.completed`) desde NATS.
* Gestiona las colecciones `pending_tasks` y `task_history` en MongoDB.
* Calcula la prioridad de ejecución de tareas.
* Expone la API REST `/queue/next` y `/health`.

**Depende de:**

* `nats` (para eventos)
* `mongo` (para almacenamiento)

---

### 📡 2. NATS JetStream

Event bus distribuido.

* Gestiona el flujo de mensajes entre el Orquestador y el Intel Service.
* Proporciona **streams persistentes** y **acknowledgements**.
* Soporta **DLQ (Dead Letter Queue)** para mensajes fallidos.

**Subjects principales:**

* `agent.task.new`
* `agent.pipeline.completed`
* `agent.task.dlq` *(opcional para fallos persistentes)*

---

### 🗄️ 3. MongoDB

Base de datos documental donde se almacenan:

* **`pending_tasks`** → Tareas pendientes con estados.
* **`task_history`** → Historial de ejecuciones (tiempo, timestamps, etc.).
* **`agent_behaviors`** *(futuro)* para versionar configuraciones del agente.

**Accedido exclusivamente por:** Agent Intel Service.

---

### 💻 4. Orquestador CLI (Node.js)

Cliente productor y consumidor de eventos.

* Publica tareas nuevas (`agent.task.new`).
* Reporta resultados (`agent.pipeline.completed`).
* Consulta el endpoint `/queue/next` del Intel Service para pedir la siguiente tarea.

---

## ⚙️ Módulos internos del Agent Intel Service

### 1. **Módulo de Ingesta (Event Consumer)**

* Escucha eventos desde NATS JetStream.
* Inserta o actualiza datos en MongoDB.
* Implementa **idempotencia**: evita procesar eventos duplicados (`issue_id + agent_version`).
* Implementa **DLQ** para mensajes fallidos.
* Controla **timeouts y reintentos** para tareas largas (1–40 min).
* Expone métricas internas (tareas procesadas, errores, tiempo medio de ingesta).

---

### 2. **Módulo de Persistencia y Aprendizaje**

Gestiona las colecciones:

* `pending_tasks`
* `task_history`

Registra automáticamente **datos clave sin intervención del usuario**:

| Campo                 | Descripción                                                                                  | Fuente                     |
| :-------------------- | :------------------------------------------------------------------------------------------- | :------------------------- |
| `created_at`          | Fecha de creación de la tarea (antigüedad)                                                   | `agent.task.new`           |
| `pipeline_runtime_ms` | Duración total del pipeline                                                                  | `agent.pipeline.completed` |
| `last_success_at`     | Última ejecución exitosa del repo                                                            | `task_history`             |
| `pending_tasks_count` | Nº de tareas pendientes por repo                                                             | `pending_tasks`            |
| `size_bytes`          | Tamaño estimado del archivo de tarea                                                         | Análisis local del `.md`   |
| `status`              | Estado actual (`pending` / `assigned` / `processing` / `completed` / `failed` / `cancelled`) | Control interno            |
| `assigned_at`         | Timestamp de asignación                                                                      | Control interno            |

---

### 3. **Módulo de Priorización (Scoring Engine)**

Calcula una **puntuación de prioridad** basada en **5 métricas automáticas**:

| Métrica               | Descripción                   | Lógica de prioridad          |
| :-------------------- | :---------------------------- | :--------------------------- |
| ⏱️ Antigüedad         | Cuánto tiempo lleva pendiente | Más antigua → más prioridad  |
| ⚡ Duración promedio   | Tiempo medio de ejecución     | Más corta → más prioridad    |
| 📂 Actividad reciente | Último éxito del repo         | Más reciente → más prioridad |
| ⚙️ Carga actual       | Nº de tareas activas por repo | Menor carga → más prioridad  |
| 💾 Tamaño de tarea    | Peso del `.md`                | Más ligera → más prioridad   |

**Fórmula inicial (MVP):**

```text
priority_score =
  (0.35 * normalized_age) +
  (0.25 * normalized_activity) +
  (0.20 * normalized_duration_inverse) +
  (0.10 * normalized_load_inverse) +
  (0.10 * normalized_size_inverse)
```

**Endpoint principal:**

```
GET /api/v1/queue/next?repo_id={ID}
→ { "issue_id": "xyz123", "priority_score": 0.87 }
```

---

### 4. **Módulo de Gobernanza y Fiabilidad**

* Exponer `/health` (estado de NATS y MongoDB).
* Controlar parámetros globales de priorización.
* Versionar comportamientos del agente.
* Registrar logs estructurados y métricas Prometheus-ready.
* Preparar para **cancelación manual de tareas** y **ajuste dinámico de pesos** en iteraciones futuras.

---

## 🔄 Flujo General (EDA)

```
       ┌────────────────────────────┐
       │     Orquestador CLI        │
       │ (Publica `task.new` y      │
       │  reporta `pipeline.completed`)│
       └────────────┬───────────────┘
                    │
           1️⃣ agent.task.new
                    │
                    ▼
           ┌─────────────────┐
           │ Agent Intel Svc │
           │ - Guarda en DB  │
           │ - Calcula score │
           │ - Controla status, timeout, idempotencia │
           └───────┬─────────┘
                   │
           2️⃣ GET /queue/next
                   │
                   ▼
       ┌────────────────────────────┐
       │ Orquestador recibe tarea   │
       │ Ejecuta y reporta evento   │
       │ `agent.pipeline.completed` │
       └────────────┬───────────────┘
                    │
                    ▼
           ┌─────────────────┐
           │ Agent Intel Svc │
           │ - Actualiza DB  │
           │ - Recalcula métricas │
           │ - Gestiona retries y timeouts │
           └─────────────────┘
```

---

## 🚀 Foco del MVP

✔ Procesar eventos `task.new` y `pipeline.completed`.
✔ Calcular prioridad usando métricas automáticas.
✔ Mantener **estados de tareas**, **timeouts**, **reintentos**, **idempotencia**.
✔ Permitir **cold start** sin crashes.
✔ Preparar **cancelación manual**, métricas Prometheus y pesos dinámicos para v2.
✔ Ejecutar todo sobre una arquitectura desacoplada, basada en eventos.

