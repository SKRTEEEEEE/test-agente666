# docs: Create MVP todo roadmap for Agent Intel Service. Closes #3291901

## Resumen de Cambios

Se ha creado exitosamente el archivo `todo.md` en la raíz del proyecto, cumpliendo con los requisitos del issue #3291901. El documento proporciona una hoja de ruta completa para la implementación del MVP del Agent Intel Service.

## Contenido Implementado

### 📋 Estructura del todo.md

El documento se organizó en **3 secciones principales**, siguiendo el formato solicitado:

#### 1. **Arquitectura y Stack Tecnológico**
- Componentes principales del sistema (Go, NATS JetStream, MongoDB, Docker)
- Descripción de los 4 servicios del MVP
- Stack tecnológico detallado

#### 2. **Módulos del Agent Intel Service**
- **Módulo de Ingesta**: Event Consumer con NATS JetStream
- **Módulo de Persistencia**: Gestión de colecciones MongoDB
- **Módulo de Priorización**: Scoring engine con 5 métricas automáticas
- **Módulo de Gobernanza**: Health checks y métricas

#### 3. **Implementación y Testing**
- Desarrollo del servicio con tareas específicas
- Estrategia de testing (unitarios, integración, endpoints, resiliencia)
- Dockerización y configuración de servicios

#### 4. **Validación del MVP**
- Flujo EDA (Event-Driven Architecture) completo
- 7 criterios de aceptación claramente definidos

## Detalles Técnicos

### Métricas de Priorización (5 automáticas)
1. **Antigüedad** (35%) - Tareas más antiguas tienen mayor prioridad
2. **Actividad reciente** (25%) - Repos activos tienen mayor prioridad
3. **Duración promedio** (20%) - Tareas más cortas tienen mayor prioridad
4. **Carga actual** (10%) - Menor carga por repo tiene mayor prioridad
5. **Tamaño de tarea** (10%) - Tareas más ligeras tienen mayor prioridad

### Datos Registrados Automáticamente
- `created_at`: Fecha de creación
- `pipeline_runtime_ms`: Duración del pipeline
- `last_success_at`: Última ejecución exitosa
- `pending_tasks_count`: Tareas pendientes por repo
- `size_bytes`: Tamaño del archivo de tarea
- `status`: Estado actual (pending/assigned/processing/completed/failed/cancelled)
- `assigned_at`: Timestamp de asignación

### Arquitectura EDA
```
Orquestador CLI → NATS (agent.task.new)
                 ↓
        Agent Intel Service
                 ↓
              MongoDB
                 ↓
        Cálculo de Score
                 ↓
     API REST /queue/next
                 ↓
          Orquestador CLI
                 ↓
    NATS (agent.pipeline.completed)
```

## Archivos Modificados

### Nuevos Archivos
- ✅ `todo.md` - Hoja de ruta completa del MVP (103 líneas)

## Validación

- ✅ Archivo creado en la raíz del proyecto
- ✅ Formato markdown correcto
- ✅ Estructura dividida en secciones lógicas
- ✅ Basado en documentación del MVP en `./docs/droid/`
- ✅ Incluye tareas específicas y medibles
- ✅ Define criterios de aceptación claros

## Commit

```
docs: create MVP todo roadmap for Agent Intel Service

- Add comprehensive todo.md with MVP implementation plan
- Document architecture, stack, and service modules
- Define implementation roadmap with testing strategy
- Include EDA flow and acceptance criteria

CO-CREATED by Agent666 created by SKRTEEEEEE
```

**Commit hash**: `8306990`

## Próximos Pasos

Con el `todo.md` completado, el equipo de desarrollo puede:

1. **Comenzar implementación del Agent Intel Service**
   - Seguir el roadmap definido en cada módulo
   - Implementar tests antes de código (TDD approach)

2. **Configurar infraestructura**
   - Configurar NATS JetStream con persistencia
   - Configurar MongoDB con colecciones definidas

3. **Desarrollar módulos en orden**
   - Módulo de Ingesta (Event Consumer)
   - Módulo de Persistencia
   - Módulo de Priorización
   - Módulo de Gobernanza

4. **Validar MVP completo**
   - Ejecutar flujo EDA end-to-end
   - Verificar todos los criterios de aceptación

---

**Nota**: Este documento fue generado automáticamente siguiendo el pipeline de Agent666.
**Fecha**: 19/10/2025
**Issue**: #3291901
