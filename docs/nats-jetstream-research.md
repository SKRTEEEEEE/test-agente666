# NATS JetStream - Reporte de Investigación

## 📋 Resumen Ejecutivo

NATS JetStream es una capa de persistencia y streaming construida sobre NATS (Neural Autonomic Transport System), un sistema de mensajería de alto rendimiento. Proporciona funcionalidades avanzadas de mensajería con garantías de entrega, persistencia de mensajes, y capacidades de reproducción (replay).

## 🎯 ¿Qué es NATS JetStream?

JetStream añade las siguientes capacidades a NATS:
- **Persistencia**: Los mensajes se almacenan en disco/memoria
- **Entrega garantizada**: At-least-once, exactly-once delivery
- **Replay de mensajes**: Capacidad de reproducir mensajes históricos
- **Stream processing**: Procesamiento de flujos de datos
- **Consumer groups**: Múltiples consumidores con balanceo de carga
- **Message deduplication**: Eliminación de duplicados
- **Horizontal scaling**: Escalabilidad mediante clustering

## 🔍 Comparación: Sistema Actual vs. NATS JetStream

### Sistema Actual (queue-go con memoria)
```
Características:
✅ Simple y ligero
✅ Baja latencia
❌ Sin persistencia (se pierden datos al reiniciar)
❌ Sin garantías de entrega
❌ Limitado a un solo nodo
❌ Sin replay de mensajes
❌ Gestión manual de estados
```

### Con NATS JetStream
```
Características:
✅ Persistencia de mensajes
✅ Garantías de entrega (at-least-once, exactly-once)
✅ Clustering y alta disponibilidad
✅ Replay de mensajes históricos
✅ Consumer groups nativos
✅ Monitoreo y observabilidad integrados
✅ Escalabilidad horizontal
✅ Deduplicación automática de mensajes
```

## 💡 Mejoras que Aporta NATS JetStream

### 1. **Persistencia y Durabilidad**
- Los mensajes sobreviven a reinicios del sistema
- Configuración de retención por tamaño, tiempo o política
- Backup y disaster recovery

### 2. **Garantías de Entrega**
- **At-least-once**: Garantiza que el mensaje se entrega al menos una vez
- **Exactly-once**: Evita duplicados mediante deduplicación
- ACKs automáticos y manuales

### 3. **Escalabilidad**
- Clustering nativo de NATS
- Distribución de carga automática
- Failover automático

### 4. **Replay y Auditoría**
- Reproducción de eventos históricos
- Debugging de flujos de trabajo
- Auditoría completa de tareas

### 5. **Consumer Groups**
- Múltiples workers procesando en paralelo
- Balanceo de carga automático
- Tolerancia a fallos

### 6. **Monitoreo**
- Métricas integradas
- Estado del stream en tiempo real
- Observabilidad de consumers

## 🏗️ Arquitectura Propuesta

```
┌─────────────────────────────────────────────────────────────┐
│                      Docker Compose                          │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌──────────────┐         ┌──────────────┐                  │
│  │   app-go     │────────▶│   queue-go   │                  │
│  │  (Producer)  │         │  (API + Web) │                  │
│  │  :8080       │         │    :8081     │                  │
│  └──────────────┘         └───────┬──────┘                  │
│                                    │                         │
│                                    ▼                         │
│                          ┌─────────────────┐                │
│                          │  NATS JetStream │                │
│                          │   (Message Bus) │                │
│                          │      :4222      │                │
│                          │  Monitoring:8222│                │
│                          └────────┬────────┘                │
│                                   │                          │
│                                   ▼                          │
│                          ┌─────────────────┐                │
│                          │ queue-worker-go │                │
│                          │   (Consumer)    │                │
│                          │  Processes tasks│                │
│                          └─────────────────┘                │
│                                                               │
└─────────────────────────────────────────────────────────────┘
```

### Flujo de Trabajo

1. **Publicación de Tareas**:
   - `app-go` o API REST publica tareas en JetStream
   - Stream: `TASKS` con subjects: `tasks.new`, `tasks.update`

2. **Procesamiento**:
   - `queue-worker-go` consume tareas del stream
   - Procesa la tarea y actualiza el estado
   - ACK automático al completar

3. **API REST** (queue-go):
   - Interface HTTP para gestión manual
   - Consulta estado de streams y consumers
   - Operaciones CRUD sobre tareas

## 📝 Pasos para Implementar NATS JetStream

### Paso 1: Agregar NATS Server al Docker Compose

```yaml
services:
  nats:
    image: nats:latest
    container_name: agent666-nats
    ports:
      - "4222:4222"  # Client connections
      - "8222:8222"  # HTTP monitoring
      - "6222:6222"  # Cluster routing
    command: 
      - "--jetstream"
      - "--store_dir=/data"
      - "--max_memory_store=1GB"
      - "--max_file_store=10GB"
    volumes:
      - nats-data:/data
    networks:
      - agent666-network
    restart: unless-stopped

volumes:
  nats-data:
```

### Paso 2: Crear Cliente NATS en Go

**Dependencias**:
```go
go get github.com/nats-io/nats.go
```

**Estructura del Proyecto**:
```
queue-go/
├── nats/
│   ├── client.go       # Cliente NATS y conexión
│   ├── streams.go      # Definición de streams
│   ├── publisher.go    # Publicación de mensajes
│   └── consumer.go     # Consumo de mensajes
├── handlers.go         # Adaptadores HTTP -> NATS
├── main.go
└── ...
```

### Paso 3: Definir Streams y Subjects

```go
// Stream para tareas
Stream: TASKS
Subjects: 
  - tasks.new          # Nueva tarea
  - tasks.status.*     # Actualizaciones de estado (tasks.status.pending, etc.)
  - tasks.delete       # Eliminación de tarea

// Consumer groups
Consumer: task-workers
  - Durable: true
  - AckPolicy: Explicit
  - MaxDeliver: 3
  - DeliverPolicy: New
```

### Paso 4: Implementar Publisher (API REST)

```go
// queue-go/handlers.go
func CreateTaskHandler(w http.ResponseWriter, r *http.Request) {
    // 1. Validar request
    // 2. Crear task
    // 3. Publicar en NATS: subject "tasks.new"
    // 4. Retornar respuesta HTTP
}
```

### Paso 5: Implementar Worker/Consumer

Crear nuevo servicio `queue-worker-go`:
```go
// queue-worker-go/main.go
func main() {
    // 1. Conectar a NATS
    // 2. Suscribirse a "tasks.new"
    // 3. Procesar mensajes
    // 4. ACK mensajes procesados
}
```

### Paso 6: Migrar Endpoints REST

- `POST /api/tasks` → Publica en `tasks.new`
- `PATCH /api/tasks/{id}/status` → Publica en `tasks.status.{status}`
- `DELETE /api/tasks/{id}` → Publica en `tasks.delete`
- `GET /api/tasks` → Consulta metadata del stream
- `GET /api/queue/status` → Consulta estado del consumer

### Paso 7: Testing

**Unit Tests**:
- Publicación de mensajes
- Consumo de mensajes
- Manejo de errores y reintentos

**Integration Tests**:
- Flujo completo: Publicar → Consumir → ACK
- Persistencia tras reinicio
- Consumer groups

**API Tests**:
- Endpoints REST con NATS en background

### Paso 8: Monitoreo

Acceder a `http://localhost:8222` para:
- Estado de streams
- Consumers activos
- Mensajes pendientes
- Estadísticas de rendimiento

## 🔧 Configuraciones Recomendadas

### Stream Configuration
```go
&nats.StreamConfig{
    Name:        "TASKS",
    Subjects:    []string{"tasks.*", "tasks.status.*"},
    Retention:   nats.WorkQueuePolicy,  // Elimina mensajes al ser ACKed
    MaxAge:      7 * 24 * time.Hour,    // Retención máxima 7 días
    Storage:     nats.FileStorage,       // Persistencia en disco
    Replicas:    1,                      // Aumentar en producción
}
```

### Consumer Configuration
```go
&nats.ConsumerConfig{
    Durable:       "task-workers",
    AckPolicy:     nats.AckExplicitPolicy,
    MaxDeliver:    3,                    // Máximo 3 intentos
    AckWait:       30 * time.Second,     // Timeout para ACK
    DeliverPolicy: nats.DeliverNewPolicy,
}
```

## 📊 Casos de Uso Mejorados

### 1. **Tolerancia a Fallos**
- Worker crashea → Mensaje vuelve a la cola automáticamente
- NATS crashea → Mensajes persisten en disco

### 2. **Escalado Horizontal**
- Ejecutar múltiples workers
- Balanceo de carga automático

### 3. **Debugging y Auditoría**
- Reproducir tareas históricas
- Ver estado completo del pipeline
- Identificar cuellos de botella

### 4. **Priorización de Tareas**
- Streams separados por prioridad
- Consumers dedicados por tipo de tarea

## ⚠️ Consideraciones

### Ventajas
- ✅ Sistema robusto y probado en producción
- ✅ Alto rendimiento (millones de msg/s)
- ✅ Fácil de operar
- ✅ Excelente documentación
- ✅ Cliente Go oficial bien mantenido

### Desventajas
- ⚠️ Complejidad adicional vs. solución en memoria
- ⚠️ Requiere infraestructura adicional (contenedor NATS)
- ⚠️ Curva de aprendizaje inicial
- ⚠️ Overhead mínimo de latencia por persistencia

## 🎓 Recursos de Aprendizaje

- [NATS Documentation](https://docs.nats.io/)
- [JetStream Guide](https://docs.nats.io/nats-concepts/jetstream)
- [nats.go Client](https://github.com/nats-io/nats.go)
- [JetStream Examples](https://github.com/nats-io/nats.go/tree/main/examples/jetstream)

## 🚀 Próximos Pasos Recomendados

1. ✅ **Fase 1 - MVP**: Implementar JetStream básico con un worker
2. 📈 **Fase 2 - Escalado**: Múltiples workers y consumer groups
3. 🔄 **Fase 3 - Avanzado**: Clustering de NATS para HA
4. 📊 **Fase 4 - Observabilidad**: Integración con Prometheus/Grafana

## 💭 Conclusión

NATS JetStream es una solución ideal para convertir el sistema de colas en memoria en un sistema distribuido, escalable y tolerante a fallos. Las mejoras en confiabilidad, persistencia y capacidades de procesamiento justifican la complejidad adicional para un sistema de producción.

Para el proyecto Agent666, permitirá:
- **Persistencia de tareas** entre reinicios
- **Procesamiento distribuido** con múltiples workers
- **Auditoría completa** del pipeline de tareas
- **Escalabilidad** para manejar cargas mayores
