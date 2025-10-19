# Agent Intel Service - MVP Implementation

## Arquitectura y Stack Tecnológico

### Componentes principales
- **Lenguaje**: Go (Golang) para alto rendimiento y concurrencia
- **Event Bus**: NATS JetStream para mensajería distribuida y persistente
- **Base de Datos**: MongoDB para almacenar historial y cola de tareas
- **Infraestructura**: Docker Compose para despliegue reproducible
- **Orquestador CLI**: Node.js como cliente productor/consumidor de eventos

### Servicios del MVP
- **Agent Intel Service (Go)**: Servicio principal de gestión de cola y priorización
- **NATS JetStream**: Event bus distribuido con streams persistentes
- **MongoDB**: Base de datos documental para `pending_tasks` y `task_history`
- **Orquestador CLI**: Cliente que publica tareas y reporta resultados

## Módulos del Agent Intel Service

### 1. Módulo de Ingesta (Event Consumer)
- Escuchar eventos desde NATS JetStream (`agent.task.new`, `agent.pipeline.completed`)
- Implementar idempotencia para evitar procesamiento de eventos duplicados
- Insertar y actualizar datos en MongoDB
- Implementar DLQ (Dead Letter Queue) para mensajes fallidos
- Controlar timeouts y reintentos para tareas largas (1-40 min)
- Exponer métricas internas (tareas procesadas, errores, tiempo medio)

### 2. Módulo de Persistencia y Aprendizaje
- Gestionar colección `pending_tasks` con estados de tareas
- Gestionar colección `task_history` con historial de ejecuciones
- Registrar automáticamente datos clave:
  - `created_at`: Fecha de creación de la tarea
  - `pipeline_runtime_ms`: Duración total del pipeline
  - `last_success_at`: Última ejecución exitosa del repo
  - `pending_tasks_count`: Número de tareas pendientes por repo
  - `size_bytes`: Tamaño estimado del archivo de tarea
  - `status`: Estado actual (pending/assigned/processing/completed/failed/cancelled)
  - `assigned_at`: Timestamp de asignación

### 3. Módulo de Priorización (Scoring Engine)
- Calcular puntuación de prioridad basada en 5 métricas automáticas:
  1. **Antigüedad** (35%): Cuánto tiempo lleva pendiente → Más antigua = más prioridad
  2. **Actividad reciente** (25%): Último éxito del repo → Más reciente = más prioridad
  3. **Duración promedio** (20%): Tiempo medio de ejecución → Más corta = más prioridad
  4. **Carga actual** (10%): Número de tareas activas por repo → Menor carga = más prioridad
  5. **Tamaño de tarea** (10%): Peso del archivo .md → Más ligera = más prioridad
- Implementar endpoint `GET /api/v1/queue/next?repo_id={ID}`
- Normalizar valores para cálculo consistente del score

### 4. Módulo de Gobernanza y Fiabilidad
- Exponer endpoint `/health` para verificar estado de NATS y MongoDB
- Controlar parámetros globales de priorización
- Versionar comportamientos del agente
- Registrar logs estructurados
- Preparar métricas Prometheus-ready
- Preparar para cancelación manual de tareas (v2)
- Preparar para ajuste dinámico de pesos (v2)

## Implementación y Testing

### Desarrollo del servicio
- Crear estructura base del proyecto Go con módulos separados
- Implementar conexión con NATS JetStream y MongoDB
- Configurar streams y subjects en NATS:
  - `agent.task.new`
  - `agent.pipeline.completed`
  - `agent.task.dlq`
- Implementar consumidores de eventos con ACK y reintentos
- Desarrollar API REST con endpoints de cola

### Testing
- **Tests unitarios**: Validar lógica de priorización y cálculo de scores
- **Tests de integración**: Verificar flujo completo de eventos y persistencia
- **Tests de endpoints**: Validar API REST con casos reales
- **Tests de resiliencia**: Validar comportamiento con NATS/MongoDB caídos

### Dockerización
- Crear Dockerfile para Agent Intel Service
- Actualizar docker-compose.yml para incluir:
  - Agent Intel Service
  - NATS JetStream con persistencia
  - MongoDB con volúmenes
- Configurar health checks y dependencias entre servicios
- Validar cold start sin crashes

## Validación del MVP

### Flujo EDA (Event-Driven Architecture)
1. Orquestador CLI publica `agent.task.new` a NATS
2. Agent Intel Service consume evento y guarda en MongoDB
3. Sistema calcula score de prioridad automáticamente
4. Orquestador consulta `GET /queue/next` para obtener siguiente tarea
5. Orquestador ejecuta pipeline y publica `agent.pipeline.completed`
6. Agent Intel Service actualiza métricas y recalcula prioridades

### Criterios de aceptación
- ✅ Procesamiento correcto de eventos `task.new` y `pipeline.completed`
- ✅ Cálculo de prioridad usando las 5 métricas automáticas
- ✅ Mantenimiento de estados de tareas con timeouts e idempotencia
- ✅ Cold start exitoso sin crashes
- ✅ API REST funcional y estable
- ✅ Arquitectura desacoplada basada en eventos
- ✅ Preparación para features v2 (cancelación manual, métricas Prometheus, pesos dinámicos)
