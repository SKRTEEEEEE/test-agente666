# 📊 Data seleccionada para priorización del MVP

*(100% automática, sin intervención del usuario)*

---

## 🕓 **1. Antigüedad de la tarea**

* **Qué es:** Tiempo transcurrido desde que la tarea fue creada (`now - created_at`).
* **Cómo se obtiene:** Directamente del evento `agent.task.new`.
* **Por qué importa:**

  * Evita que tareas antiguas queden atascadas.
  * Garantiza un flujo justo en la cola.
* **Uso inmediato:**

  * A mayor antigüedad, mayor prioridad.
  * Puede aplicarse como factor base del score de prioridad.
* **Ejemplo:**

  ```
  age_score = (now - created_at).hours()
  ```

---

## ⚡ **2. Duración promedio del pipeline (runtime histórico)**

* **Qué es:** Promedio de `pipeline_runtime_ms` de tareas anteriores del mismo repo o comportamiento.
* **Cómo se obtiene:** A partir de los eventos `agent.pipeline.completed`.
* **Por qué importa:**

  * Las tareas más cortas pueden ejecutarse antes para maximizar throughput.
  * Las más largas pueden reservarse para momentos de baja carga.
* **Uso inmediato:**

  * Penalizar tareas con runtime medio alto.
  * Normalizar valores para no priorizar en exceso las más rápidas.
* **Ejemplo:**

  ```
  duration_score = 1 / average(pipeline_runtime_ms)
  ```

---

## 📂 **3. Actividad reciente del repositorio**

* **Qué es:** Tiempo desde la última tarea completada con éxito del mismo repo (`now - last_success_at`).
* **Cómo se obtiene:** Consultando `task_history` filtrado por `repo_id` y `status = success`.
* **Por qué importa:**

  * Un repo con actividad reciente indica contexto fresco o trabajo continuo.
  * Permite priorizar “sesiones activas”.
* **Uso inmediato:**

  * Disminuir prioridad si el repo está inactivo mucho tiempo.
  * Aumentarla si el repo está siendo usado ahora.
* **Ejemplo:**

  ```
  recent_activity_score = exp(-λ * hours_since_last_success)
  ```

---

## ⚙️ **4. Carga actual de tareas por repositorio**

* **Qué es:** Cantidad de tareas en estado pendiente para un mismo `repo_id`.
* **Cómo se obtiene:** Conteo en la colección `pending_tasks`.
* **Por qué importa:**

  * Evita saturar repos con demasiadas tareas en cola.
  * Distribuye el trabajo equitativamente entre repos activos.
* **Uso inmediato:**

  * Penalizar levemente repos con alta carga actual.
* **Ejemplo:**

  ```
  load_score = 1 / (1 + pending_tasks_count_for_repo)
  ```

---

## 💾 **5. Tamaño de la tarea**

* **Qué es:** Longitud del documento o número estimado de tokens/líneas/bytes a procesar.
* **Cómo se obtiene:**

  * En el evento `agent.task.new`, analizando el archivo `docs/task/*.md`.
  * Guardar en campo `size_bytes` o `lines_count`.
* **Por qué importa:**

  * Las tareas más grandes tienden a requerir más recursos.
  * Permite despachar antes las pequeñas para mantener la cola ágil.
* **Uso inmediato:**

  * Penalizar ligeramente tareas de mayor tamaño.
* **Ejemplo:**

  ```
  size_score = 1 / (1 + normalized_size)
  ```

---

# 🧮 Resumen final — métricas clave para el MVP

| # | Métrica                        | Fuente                                       | Uso principal                      | Tipo de efecto      |
| - | ------------------------------ | -------------------------------------------- | ---------------------------------- | ------------------- |
| 1 | ⏱️ Antigüedad                  | `created_at` (`task.new`)                    | Priorizar tareas más antiguas      | Aumenta prioridad   |
| 2 | ⚡ Duración promedio            | `pipeline_runtime_ms` (`pipeline.completed`) | Favorecer tareas más rápidas       | Aumenta prioridad   |
| 5 | 📂 Actividad reciente del repo | `last_success_at` (`task_history`)           | Repos activos tienen más prioridad | Aumenta prioridad   |
| 6 | ⚙️ Carga actual del repo       | Conteo en `pending_tasks`                    | Evitar saturación por repo         | Disminuye prioridad |
| 9 | 💾 Tamaño de tarea             | `size_bytes` (`task.new`)                    | Penalizar tareas grandes           | Disminuye prioridad |

