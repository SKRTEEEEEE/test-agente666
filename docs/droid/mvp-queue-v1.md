

# 🧠 Propuesta Consolidada: MVP del Agent Intel Service

El **Agent Intel Service** es un microservicio independiente que centraliza la gestión de la cola y la lógica de priorización. Alimentándose del historial de ejecuciones del agente para optimizar futuras tareas y la lógica de priorización de la cola de tareas.

## 🎯 Objetivo y Roles del Servicio

| Rol | Objetivo |
| :--- | :--- |
| **Aprendizaje** | Analizar los resultados de la ejecución del agente (tiempo, iteraciones, éxito/falla) para calcular métricas de eficiencia. |
| **Priorización** | Usar las métricas de eficiencia y probabilidad de éxito para determinar de forma inteligente la siguiente tarea a procesar. |
| **Gobernanza** | Centralizar la gestión de configuraciones y comportamientos del agente para mantener la consistencia. |

## 🛠️ Stack Tecnológico del MVP

| Componente | Tecnología | Propósito |
| :--- | :--- | :--- |
| **Lenguaje del Servicio** | **Go (Golang)** | Alto rendimiento, concurrencia y excelente integración con NATS. |
| **Event Bus / Queue** | **NATS JetStream** | Backbone distribuido y duradero para el flujo de eventos de la cola. |
| **Base de Datos** | **MongoDB** (NoSQL) | Flexibilidad para almacenar el historial de tareas y estados complejos (escalable a PostgreSQL + pgvector). |
| **Infraestructura** | **Docker Compose** | Simplificar el despliegue de NATS, MongoDB y el servicio Go. |

---

## ⚙️ Arquitectura y Flujo de Eventos

El flujo se centra en reemplazar el *polling* local de archivos por un modelo de **Event-Driven Architecture (EDA)**.

### 1. Event Ingestion (NATS JetStream)

La CLI del Orquestador (Node.js) se convierte en un **publicador** de eventos:

| Evento (Subject) | Publicador | Propósito | Payload Clave |
| :--- | :--- | :--- | :--- |
| `agent.task.new` | Orquestador CLI | Notificar que un nuevo archivo `docs/task/*.md` está listo. | `issue_id`, `repo_path`, `agent_behavior` |
| `agent.pipeline.completed` | Orquestador CLI | Reportar el resultado final de la ejecución del agente. | `issue_id`, `status`, `pipeline_runtime_ms`, **`iterations_count`**, **`agent_version`**, `error_summary` |

### 2. Módulos del Agent Intel Service (Go)

El servicio Go es el **consumidor** principal de NATS y gestor de la base de datos.

#### A. Módulo de Persistencia y Aprendizaje
* **Consumo:** Se suscribe a los dos *Subjects* de NATS.
* **Almacenamiento (MongoDB):**
    * `pending_tasks`: Almacena tareas de `agent.task.new` y las elimina con `agent.pipeline.completed`.
    * `task_history`: Almacena el historial de ejecuciones de `agent.pipeline.completed` para el cálculo de métricas.

#### B. Módulo de Priorización (API REST)
* **Endpoint:** `GET /api/v1/queue/next?repo_id={ID}`
* **Lógica de Prioridad Total (PT):**
    1.  Obtener tareas pendientes de la app.
    2.  Consultar `task_history` para calcular métricas históricas para el `repo_id` y el `agent_behavior`:
        * **Probabilidad de Éxito (PE):** $(\text{Éxitos} / \text{Total}) \times 100$
        * **Eficiencia Promedio (EP):** $\text{Media}(1 / \text{Número de Iteraciones})$
    3.  Aplicar la fórmula: $PT = (0.6 \times PE) + (0.4 \times EP)$.
    4.  Retornar el `issue_id` con la **mayor PT**.

---

## ✅ Adiciones Fundamentales (Fiabilidad y Gobernanza)

Estas adiciones son cruciales para la robustez del servicio independiente.

### 1. Módulo de Fiabilidad (Gestión de Fallos)

* **Idempotencia:** Usar una combinación de `issue_id` y `timestamp` para evitar la duplicación de registros en `task_history` en caso de reintentos.
* **Dead Letter Queue (DLQ):** Configurar el *consumer* de NATS para que mueva mensajes fallidos de forma permanente a una *Subject* separada si excede el límite de reintentos.
* **Health Check:** Exponer un endpoint `/health` que reporte el estado de las conexiones con **NATS** y **MongoDB**.

### 2. Módulo de Gobernanza (Configuración Centralizada)

* **Catálogo de Comportamientos (MongoDB):** Almacenar los nombres, prompts base y metadatos de los comportamientos del agente (`full`, `config`, `esp`).
* **Versionamiento del Agente:** El **Agent Intel Service** usará el campo `agent_version` para **despriorizar** o ignorar el historial de versiones del agente que hayan demostrado ser problemáticas o ineficientes.
* **Endpoint de Configuración Dinámica:**
    * `GET /api/v1/agent/config?repo_id=XYZ`
    * Este endpoint permite al servicio de inteligencia **sugerir o forzar** la mejor configuración (ej: el límite de iteraciones) al Orquestador local basándose en el análisis histórico.

---

## 🔄 Flujo de la Cola Modificado

1.  **Orquestador CLI:** Detecta un nuevo `docs/task/*.md` y publica `agent.task.new` a NATS.
2.  **Agent Intel Service (Go):** Consume el evento y lo guarda en `pending_tasks` (MongoDB).
3.  **Orquestador CLI:** Cuando está libre, pregunta al servicio Go: `GET /api/v1/queue/next?repo_id=XYZ`.
4.  **Agent Intel Service (Go):** Ejecuta la **Lógica de Prioridad Total (PT)**, consulta `task_history` y retorna el `issue_id` más eficiente/probable de éxito.
5.  **Orquestador CLI:** Ejecuta la tarea.
6.  **Orquestador CLI:** Al finalizar, publica `agent.pipeline.completed` a NATS.
7.  **Agent Intel Service (Go):** Consume el evento, actualiza `task_history` (el modelo de aprendizaje) y elimina la tarea de `pending_tasks`.

