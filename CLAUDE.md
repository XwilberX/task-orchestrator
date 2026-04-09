# task-orchestrator

## ¿Qué es este proyecto?

**task-orchestrator** es un servicio centralizado de ejecución de tareas asíncronas. El objetivo es permitir que múltiples proyectos independientes (clientes) deleguen trabajo en segundo plano o programado a un único servicio compartido, eliminando la necesidad de que cada proyecto corra su propio Celery u otra infraestructura de workers.

Los clientes pueden pre-registrar definiciones de tareas y dispararlas por nombre, o enviar código ad-hoc inline con cada request. Todas las tareas corren en contenedores Docker aislados.

El proyecto tiene dos partes:

- **Backend** — servidor API en Go (`/backend`)
- **Frontend** — dashboard en SvelteKit (`/frontend`)

Ambos viven en el mismo monorepo y se orquestan juntos con Docker Compose.

---

## Estructura del monorepo

```
task-orchestrator/
├── backend/
├── frontend/
├── docker-compose.yml
├── docker-compose.dev.yml
└── .env.example
```

---

## Backend

### Stack tecnológico

| Responsabilidad | Librería |
|---|---|
| Lenguaje | Go 1.22+ |
| Router HTTP | `github.com/go-chi/chi/v5` |
| Driver MongoDB | `go.mongodb.org/mongo-driver/v2` |
| SDK Docker | `github.com/docker/docker/client` |
| Scheduler cron | `github.com/robfig/cron/v3` |
| JWT | `github.com/golang-jwt/jwt/v5` |
| Generación de UUIDs | `github.com/google/uuid` |
| Validación de requests | `github.com/go-playground/validator/v10` |
| Variables de entorno | `github.com/joho/godotenv` |
| Logging estructurado | `go.uber.org/zap` |
| Testing | `github.com/stretchr/testify` |
| Hot reload en desarrollo | `github.com/air-verse/air` |

### Estructura de carpetas (por capas)

```
backend/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── config/          # carga de variables de entorno con godotenv
│   ├── handlers/        # handlers HTTP (un archivo por recurso)
│   ├── services/        # lógica de negocio
│   ├── repositories/    # queries a MongoDB
│   ├── models/          # structs compartidos (Task, Definition, Schedule, Webhook...)
│   ├── scheduler/       # goroutines de cron y polling de la cola
│   ├── executor/        # ciclo de vida de contenedores Docker
│   ├── runtime/         # configuración de imágenes Docker por lenguaje
│   ├── logger/          # cliente HTTP para Victoria Logs
│   ├── webhook/         # dispatcher de webhooks
│   └── middleware/      # autenticación por API key, logging de requests, recovery
├── pkg/
│   └── response/        # helpers de respuesta estándar de la API
├── .air.toml
├── .env.example
├── Dockerfile
└── go.mod
```

### Variables de entorno

```env
PORT=8080
API_KEY=change-me-in-production

MONGO_URI=mongodb://mongo:27017
MONGO_DB=task_orchestrator

VICTORIA_LOGS_URL=http://victoria-logs:9428

DOCKER_HOST=unix:///var/run/docker.sock

MAX_CONCURRENT_TASKS=10
```

### Autenticación

API key global única. Todas las rutas `/api/*` requieren el header:

```
X-API-Key: <valor>
```

Retornar `401` si falta o es inválida. Las rutas del dashboard (`/`) no están protegidas.

### Formato estándar de respuestas

Todas las respuestas — éxito y error — usan este envelope:

```json
{
  "success": true,
  "data": { },
  "message": "mensaje legible opcional",
  "error": null
}
```

En caso de error:

```json
{
  "success": false,
  "data": null,
  "message": "Validación fallida",
  "error": "el campo 'runtime' es requerido"
}
```

Los códigos HTTP siempre deben ser semánticamente correctos (200, 201, 400, 401, 404, 409, 422, 500). Nunca retornar 200 con `success: false`.

Implementar un paquete `pkg/response` con helpers:

```go
response.OK(w, data, message)
response.Created(w, data)
response.BadRequest(w, err, message)
response.NotFound(w, message)
response.InternalError(w, err)
```

### Endpoints de la API

#### Definiciones de tareas

```
POST   /api/v1/definitions         Registrar una nueva definición
GET    /api/v1/definitions         Listar todas las definiciones
GET    /api/v1/definitions/:id     Obtener una definición por ID
PUT    /api/v1/definitions/:id     Actualizar una definición
DELETE /api/v1/definitions/:id     Eliminar una definición
```

Modelo de definición:

```json
{
  "id": "uuid",
  "name": "send-report-email",
  "description": "Envía el reporte semanal por email",
  "runtime": "python",
  "code": "import sys\nprint(sys.argv)",
  "packages": "requests pandas",
  "timeout_seconds": 120,
  "max_retries": 3,
  "backoff_multiplier": 5,
  "max_concurrent": 2,
  "memory_mb": 256,
  "cpu_shares": 512,
  "network_enabled": false,
  "created_at": "...",
  "updated_at": "..."
}
```

#### Ejecuciones de tareas

```
POST   /api/v1/tasks               Despachar una tarea (registrada o ad-hoc)
GET    /api/v1/tasks               Listar ejecuciones (filtros: status, definition_id, runtime, from, to)
GET    /api/v1/tasks/:id           Obtener detalle + metadata de una ejecución
DELETE /api/v1/tasks/:id           Cancelar una tarea PENDING o QUEUED
GET    /api/v1/tasks/:id/logs      Proxy de logs desde Victoria Logs para esta tarea
GET    /api/v1/tasks/:id/stream    Stream SSE de logs en vivo mientras la tarea está RUNNING
```

Payload para despachar una tarea registrada:

```json
{
  "definition": "send-report-email",
  "input": { "userId": 42 },
  "timeout_seconds": 120
}
```

Payload para despachar una tarea ad-hoc:

```json
{
  "runtime": "python",
  "code": "import sys\nprint('hello')",
  "args": ["world"],
  "packages": "requests pandas",
  "timeout_seconds": 60,
  "memory_mb": 256
}
```

#### Schedules (cron)

```
POST   /api/v1/schedules                  Crear un schedule
GET    /api/v1/schedules                  Listar schedules
GET    /api/v1/schedules/:id              Obtener un schedule
PUT    /api/v1/schedules/:id              Actualizar un schedule
DELETE /api/v1/schedules/:id              Eliminar un schedule
PATCH  /api/v1/schedules/:id/toggle       Pausar o reanudar un schedule
```

Modelo de schedule:

```json
{
  "id": "uuid",
  "definition_id": "uuid",
  "cron": "0 9 * * 1",
  "status": "active",
  "last_run_at": "...",
  "next_run_at": "...",
  "created_at": "..."
}
```

Usar `github.com/robfig/cron/v3` para parsear expresiones y scheduling. Las ejecuciones perdidas mientras el servicio estuvo caído NO se ejecutan retroactivamente.

#### Webhooks

```
POST   /api/v1/webhooks            Registrar un webhook
GET    /api/v1/webhooks            Listar webhooks
DELETE /api/v1/webhooks/:id        Eliminar un webhook
GET    /api/v1/webhooks/:id/logs   Historial de entregas de un webhook
```

Payload que se envía cuando una tarea llega a estado terminal (`SUCCESS`, `FAILED`, `TIMEOUT`):

```json
{
  "event": "task.completed",
  "task_id": "uuid",
  "definition": "send-report-email",
  "status": "SUCCESS",
  "started_at": "...",
  "finished_at": "...",
  "attempt": 1,
  "output": { "exit_code": 0 }
}
```

Entrega de webhooks: 3 intentos, 10s de backoff. Los fallos se loggean pero no afectan el estado de la tarea.

#### Métricas (para el home del dashboard)

```
GET /api/v1/metrics/summary
```

Respuesta:

```json
{
  "tasks_today": 142,
  "tasks_failed": 8,
  "tasks_queued": 3,
  "tasks_running": 2,
  "avg_duration_seconds": 4.7
}
```

### Ciclo de vida de una tarea

Estados posibles:

```
PENDING → QUEUED → RUNNING → SUCCESS
                           → FAILED
                           → TIMEOUT
         → CANCELLED
```

- `PENDING`: recibida, aún no evaluada contra los límites de concurrencia
- `QUEUED`: límite de concurrencia alcanzado, esperando un slot disponible
- `RUNNING`: contenedor activo
- `SUCCESS` / `FAILED` / `TIMEOUT`: estados terminales

### Política de reintentos

- Por defecto: 3 intentos
- Backoff exponencial: `backoff_multiplier ^ intento` segundos (multiplicador por defecto: 5 → 5s, 25s, 125s)
- Tanto `FAILED` como `TIMEOUT` disparan reintentos
- Configurable por definición: `max_retries`, `backoff_multiplier`
- Tras agotar todos los intentos → `FAILED` permanente

### Control de concurrencia

- Pool global de workers controlado por la variable de entorno `MAX_CONCURRENT_TASKS`
- Una goroutine scheduler hace polling a MongoDB buscando tareas `QUEUED` y las despacha cuando se libera un slot
- El límite `max_concurrent` por definición también se aplica

### Aislamiento de tareas (executor Docker)

Cada tarea corre en su propio contenedor Docker efímero:

- Usar el SDK oficial de Go para Docker (`github.com/docker/docker/client`) — NO ejecutar el CLI de `docker` con shell
- El contenedor se destruye tras completar la ejecución, timeout o cancelación (usar siempre `defer` para la limpieza)
- Sin sistema de archivos compartido entre tareas
- Red deshabilitada por defecto (`network_enabled: false`), configurable por definición
- Límites de CPU y memoria aplicados via Docker `HostConfig`
- Los paquetes declarados en el campo `packages` se instalan al inicio del contenedor antes de correr el código del usuario (pip install, npm install, go get, etc.)
- stdout/stderr se streamean en tiempo real a Victoria Logs, etiquetados con `task_id`, `definition_name`, `runtime`, `attempt`

### Runtimes soportados

| Runtime | Imagen base | Comando de instalación |
|---|---|---|
| `python` | `python:3.11-slim` | `pip install <packages>` |
| `nodejs` | `node:20-slim` | `npm install <packages>` |
| `typescript` | `node:20-slim` | `npm install -g tsx && npm install <packages>` |
| `go` | `golang:1.22-alpine` | `go get <packages>` |
| `java` | `eclipse-temurin:21-alpine` | Solo archivos `.java` simples, sin Maven/Gradle por ahora |

La configuración de cada runtime vive en `internal/runtime/` como un struct que implementa una interfaz común.

### Logging con Victoria Logs

- Todo stdout/stderr de los contenedores → Victoria Logs via API HTTP de ingesta
- Logs del proceso del orquestador → `go.uber.org/zap` (logs JSON estructurados a stdout)
- `GET /api/v1/tasks/:id/logs` → obtiene logs históricos desde Victoria Logs y los retorna
- `GET /api/v1/tasks/:id/stream` → endpoint SSE que streamea output del contenedor en vivo mientras la tarea está `RUNNING`

### SSE para actualizaciones en tiempo real

El backend expone dos endpoints SSE:

```
GET /api/v1/tasks/:id/stream     Stream de logs en vivo para una tarea en ejecución
GET /api/v1/events               Eventos globales de cambio de estado (RUNNING, SUCCESS, FAILED, etc.)
```

`/api/v1/events` permite que el dashboard actualice los estados de las tareas en tiempo real sin hacer polling.

---

## Frontend

### Stack tecnológico

| Responsabilidad | Librería |
|---|---|
| Framework | SvelteKit (latest) |
| Estilos | Tailwind CSS v4 |
| Librería de componentes | shadcn-svelte |
| Iconos | lucide-svelte |
| Fetching de datos | TanStack Query para Svelte (`@tanstack/svelte-query`) |
| Formularios | Superforms (`sveltekit-superforms`) |
| Validación de formularios | Zod |
| Editor de código | Monaco Editor (`@monaco-editor/loader`) |
| Fechas | date-fns |
| Tiempo real | `EventSource` nativo del browser (SSE) |
| Testing | Playwright (e2e) |

### Estructura de carpetas (por capas)

```
frontend/
├── src/
│   ├── lib/
│   │   ├── components/       # componentes UI reutilizables
│   │   │   ├── ui/           # componentes base de shadcn-svelte
│   │   │   ├── TaskStatusBadge.svelte
│   │   │   ├── RuntimeBadge.svelte
│   │   │   ├── CodeEditor.svelte
│   │   │   └── LogViewer.svelte
│   │   ├── stores/           # stores de Svelte para estado global
│   │   ├── services/         # funciones cliente de la API (un archivo por recurso)
│   │   │   ├── api.ts        # wrapper base de fetch con header de API key
│   │   │   ├── tasks.ts
│   │   │   ├── definitions.ts
│   │   │   ├── schedules.ts
│   │   │   └── webhooks.ts
│   │   ├── schemas/          # schemas Zod (compartidos entre validación y tipos)
│   │   └── utils/            # formato de fechas, duración, helpers varios
│   ├── routes/
│   │   ├── +layout.svelte    # sidebar + navegación superior
│   │   ├── +page.svelte      # home / métricas generales
│   │   ├── tasks/
│   │   │   ├── +page.svelte          # lista de ejecuciones con filtros
│   │   │   └── [id]/
│   │   │       └── +page.svelte      # detalle de tarea: logs, reintentos, metadata
│   │   ├── definitions/
│   │   │   ├── +page.svelte          # lista de definiciones
│   │   │   ├── new/
│   │   │   │   └── +page.svelte      # formulario crear definición
│   │   │   └── [id]/
│   │   │       └── +page.svelte      # formulario editar definición
│   │   ├── schedules/
│   │   │   └── +page.svelte          # lista de schedules + crear/editar inline
│   │   └── webhooks/
│   │       └── +page.svelte          # lista de webhooks + historial de entregas
├── tests/                    # tests e2e con Playwright
├── .env.example
├── svelte.config.js
├── tailwind.config.ts
├── vite.config.ts
└── package.json
```

### Cliente de la API

Todas las llamadas a la API pasan por `src/lib/services/api.ts`, un wrapper delgado sobre `fetch` que:

- Inyecta el header `X-API-Key` desde las variables de entorno `PUBLIC_API_URL` y `PUBLIC_API_KEY`
- Desempaqueta el envelope `{ success, data, error, message }`
- Lanza un error tipado si `success: false`

### Páginas del dashboard

#### Home (`/`)

Tarjetas de métricas en la parte superior:
- Tareas hoy
- Tareas fallidas
- Tareas en cola
- Corriendo actualmente
- Duración promedio

Debajo: tabla con las 20 ejecuciones más recientes mostrando badge de estado, badge de runtime, nombre de definición, duración y timestamp. Se refresca automáticamente via TanStack Query con un `staleTime` corto.

El stream SSE de `/api/v1/events` se conecta globalmente en el layout y actualiza el caché de TanStack Query en tiempo real cuando cambian los estados de las tareas.

#### Lista de tareas (`/tasks`)

Tabla paginada de ejecuciones con filtros:
- Estado (multi-select: PENDING, QUEUED, RUNNING, SUCCESS, FAILED, TIMEOUT, CANCELLED)
- Runtime (multi-select)
- Rango de fechas (from / to)

Cada fila enlaza a la página de detalle de la tarea.

#### Detalle de tarea (`/tasks/:id`)

- Badge de estado, runtime, nombre de definición, created at, started at, finished at, duración
- Payload de input (visor JSON)
- Historial de reintentos: tabla de intentos con estado, started at, exit code
- Visor de logs: si la tarea está `RUNNING`, se conecta a `GET /api/v1/tasks/:id/stream` via `EventSource` y streamea logs en vivo. Si la tarea está en estado terminal, obtiene los logs de `GET /api/v1/tasks/:id/logs` y los renderiza estáticamente.
- Botón de cancelar (visible solo si el estado es `PENDING` o `QUEUED`)

#### Definiciones (`/definitions`)

Tabla de todas las definiciones registradas con nombre, runtime, timeout, max reintentos, created at.
Acciones: editar, eliminar, disparar (despacha inmediatamente con input vacío).

#### Crear / editar definición (`/definitions/new`, `/definitions/:id`)

Campos del formulario:
- Nombre
- Descripción
- Runtime (select: python, nodejs, typescript, go, java)
- Código — editor Monaco. El lenguaje cambia automáticamente según el runtime seleccionado. TypeScript/JavaScript tienen IntelliSense completo. Los demás lenguajes tienen syntax highlighting únicamente.
- Paquetes — input de texto libre separado por espacios (ej: `requests pandas numpy`)
- Timeout (segundos)
- Max reintentos
- Multiplicador de backoff
- Max concurrente
- Memoria (MB)
- CPU shares
- Red habilitada (toggle)

El formulario usa Superforms + Zod para validación. Los errores se muestran inline debajo de cada campo.

#### Schedules (`/schedules`)

Tabla: nombre de definición, expresión cron, badge de estado (active/paused), último run, próximo run.
Acciones por fila:
- Pausar / Reanudar (PATCH toggle)
- Eliminar
- Formulario inline "Agregar schedule" al final: seleccionar definición, ingresar expresión cron, enviar.

#### Webhooks (`/webhooks`)

Tabla de webhooks registrados: URL, created at, último estado de entrega.
Acciones: eliminar, expandir historial de entregas (últimas 20 entregas con timestamp, HTTP status, respuesta).
Formulario inline para registrar una nueva URL de webhook.

### Componente editor Monaco

`src/lib/components/CodeEditor.svelte`:

- Carga Monaco via `@monaco-editor/loader` (CDN, sin necesidad de webpack)
- Acepta prop `language` — cambia el modelo de lenguaje de Monaco cuando cambia
- Para `typescript` y `javascript`: habilitar IntelliSense integrado (comportamiento por defecto de Monaco)
- Para los demás (`python`, `go`, `java`): configurar el lenguaje correcto solo para syntax highlighting, sin LSP
- Emite evento `change` con el valor actual del editor
- El tema claro/oscuro sigue la preferencia del sistema via `prefers-color-scheme`

### Tiempo real con SSE

- El layout global se conecta a `GET /api/v1/events` al montarse
- Con cada evento, se invalida la query key relevante de TanStack Query para que la UI se actualice automáticamente
- El visor de logs en el detalle de tarea usa una conexión `EventSource` separada a `GET /api/v1/tasks/:id/stream`
- Al desmontar el componente, siempre llamar `eventSource.close()`

---

## Docker Compose

### Producción (`docker-compose.yml`)

Servicios:
- `orchestrator` — binario Go, puerto 8080, monta el socket de Docker
- `dashboard` — SvelteKit compilado con Node adapter, puerto 3000
- `mongo` — MongoDB 7, volumen nombrado
- `victoria-logs` — VictoriaLogs latest, puerto 9428, volumen nombrado

### Desarrollo (`docker-compose.dev.yml`)

Mismos servicios pero:
- `orchestrator` usa `air` para hot reload, monta el código fuente
- `dashboard` corre `npm run dev`, monta el código fuente, puerto 5173

### El contenedor del orquestador debe montar el socket de Docker:

```yaml
volumes:
  - /var/run/docker.sock:/var/run/docker.sock
```

Esto permite que el proceso Go levante contenedores hermanos para la ejecución de tareas.

---

## Notas de implementación

- El executor debe manejar la limpieza de contenedores incluso en caso de panics — usar siempre `defer` para eliminar contenedores
- Los payloads de input/output de las tareas se guardan en MongoDB como documentos BSON sin esquema fijo
- Todos los timestamps se almacenan y retornan en UTC ISO 8601
- Los IDs son UUIDs (v4) generados con `github.com/google/uuid`
- Los logs estructurados del proceso del orquestador usan `go.uber.org/zap` en modo JSON
- El paquete `internal/runtime/` define una interfaz `Runtime` con los métodos: `Image() string`, `InstallCommand(packages string) string`, `RunCommand(code string, args []string) []string`
- Nunca ejecutar el CLI de `docker` via shell — siempre usar el SDK de Go para Docker

---

## Orden de construcción sugerido

1. Scaffold del monorepo + configuración de Docker Compose (prod + dev)
2. Backend: config, conexión a MongoDB, paquete base de response
3. Backend: CRUD de definiciones de tareas (handlers + service + repository)
4. Backend: executor Docker — tareas ad-hoc, sin cola aún
5. Backend: máquina de estados de tareas, lógica de reintentos, enforcement de timeout
6. Backend: cola de concurrencia (goroutine scheduler + estado QUEUED)
7. Backend: integración con Victoria Logs (escritura + lectura)
8. Backend: endpoints SSE (logs en vivo + eventos globales)
9. Backend: scheduler de cron
10. Backend: dispatcher de webhooks
11. Backend: endpoint de métricas resumen
12. Frontend: scaffold de SvelteKit + Tailwind + shadcn-svelte
13. Frontend: cliente API + configuración de TanStack Query
14. Frontend: layout + navegación
15. Frontend: página home (métricas + tareas recientes)
16. Frontend: lista de tareas + filtros
17. Frontend: detalle de tarea + visor de logs + SSE
18. Frontend: CRUD de definiciones + editor Monaco
19. Frontend: página de schedules
20. Frontend: página de webhooks
21. Tests e2e con Playwright para los flujos críticos
