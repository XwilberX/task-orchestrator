# task-orchestrator

Servicio centralizado de ejecución de tareas asíncronas. Permite que múltiples proyectos deleguen trabajo en segundo plano o programado a un único servicio compartido, eliminando la necesidad de que cada proyecto corra su propio Celery u otra infraestructura de workers.

Cada tarea corre en un contenedor Docker efímero y aislado.

---

## Stack

| Capa | Tecnología |
|---|---|
| Backend | Go 1.24+ |
| Base de datos | MongoDB 8 |
| Logs | Victoria Logs |
| Executor | Docker SDK (Go) |
| Cron | robfig/cron/v3 |
| Frontend | SvelteKit + Tailwind CSS v4 + shadcn-svelte |
| Dev hot-reload | Air (backend) + Vite (frontend) |

---

## Runtimes soportados

| Runtime | Imagen | Paquetes |
|---|---|---|
| `python` | `python:<version>-slim` | `pip install` |
| `nodejs` | `node:<version>-slim` | `npm install` |
| `go` | `golang:<version>-alpine` | stdlib (sin install) |
| `java` | `eclipse-temurin:<version>` | Maven Central (`groupId:artifactId:version`) |

Las versiones disponibles se consultan en tiempo real desde Docker Hub y se cachean 1 hora.

---

## Levantar en desarrollo

```bash
# 1. Variables de entorno
cp backend/.env.example backend/.env
# Editar: API_KEY=dev-key, GVISOR_RUNTIME="" (vacío = runc, sin gVisor)

# 2. Infraestructura + backend con hot-reload
docker compose -f docker-compose.dev.yml up

# 3. Frontend (en otra terminal)
cd frontend && bun install && bun run dev
```

Dashboard disponible en `http://localhost:5173`.

---

## API

Todas las rutas requieren el header:

```
X-API-Key: <valor>
```

Formato de respuesta estándar:

```json
{ "success": true, "data": {}, "message": "", "error": null }
```

### Definiciones

```
POST   /api/v1/definitions               Crear definición
GET    /api/v1/definitions               Listar
GET    /api/v1/definitions/:id           Obtener por ID
PUT    /api/v1/definitions/:id           Actualizar
DELETE /api/v1/definitions/:id           Eliminar
POST   /api/v1/definitions/:id/upload    Subir código como archivo
```

#### Upload de código (multipart/form-data)

Alternativa al campo `code` en JSON. Acepta archivos `.py`, `.js`, `.go`, `.java` hasta 5 MB.

```bash
curl -F "file=@script.py" \
  http://localhost:8080/api/v1/definitions/<id>/upload \
  -H "X-API-Key: dev-key"
```

Solo actualiza el campo `code` — el resto de la definición queda intacto.

### Tareas

```
POST   /api/v1/tasks              Despachar tarea
GET    /api/v1/tasks              Listar (filtros: status, runtime, from, to)
GET    /api/v1/tasks/:id          Detalle
DELETE /api/v1/tasks/:id          Cancelar (solo PENDING/QUEUED)
GET    /api/v1/tasks/:id/logs     Logs históricos
GET    /api/v1/tasks/:id/stream   SSE — logs en vivo
```

**Despachar desde definición:**
```json
{ "definition": "nombre-definicion", "input": {} }
```

**Tarea ad-hoc:**
```json
{
  "runtime": "python",
  "code": "print('hola')",
  "packages": "requests pandas",
  "timeout_seconds": 60,
  "memory_mb": 256,
  "network_enabled": false
}
```

### Schedules

```
POST   /api/v1/schedules                Crear
GET    /api/v1/schedules                Listar
PUT    /api/v1/schedules/:id            Actualizar
DELETE /api/v1/schedules/:id            Eliminar
PATCH  /api/v1/schedules/:id/toggle     Pausar / Reanudar
```

### Webhooks

```
POST   /api/v1/webhooks            Registrar
GET    /api/v1/webhooks            Listar
DELETE /api/v1/webhooks/:id        Eliminar
GET    /api/v1/webhooks/:id/logs   Historial de entregas
```

### Utilidades

```
GET /api/v1/runtimes          Versiones disponibles por runtime
GET /api/v1/metrics/summary   Métricas del día
GET /api/v1/events            SSE — eventos globales de cambio de estado
GET /health                   Health check
```

---

## Ciclo de vida de una tarea

```
PENDING → QUEUED → RUNNING → SUCCESS
                           → FAILED
                           → TIMEOUT
         → CANCELLED
```

- **PENDING** — recibida, evaluando concurrencia
- **QUEUED** — límite de concurrencia alcanzado, esperando slot
- **RUNNING** — contenedor activo, logs en streaming
- **SUCCESS / FAILED / TIMEOUT** — estados terminales

**Reintentos:** backoff exponencial configurable por definición (`max_retries`, `backoff_multiplier`).

---

## Variables de entorno (backend)

```env
PORT=8080
API_KEY=change-me-in-production

MONGO_URI=mongodb://mongo:27017
MONGO_DB=task_orchestrator

VICTORIA_LOGS_URL=http://victoria-logs:9428

DOCKER_HOST=unix:///var/run/docker.sock

MAX_CONCURRENT_TASKS=10

# Vacío en dev (runc), "runsc" en producción con gVisor
GVISOR_RUNTIME=
```

---

## Notas importantes

- Definiciones que usan `packages` necesitan `network_enabled: true` para que el contenedor pueda descargar dependencias.
- Java usa coordenadas Maven: `packages = "com.google.code.gson:gson:2.10.1"`. Los JARs se descargan desde Maven Central.
- El executor limita el body HTTP a **5 MB** por request.
- Los contenedores se eliminan automáticamente al terminar (con `Force: true`), incluso en caso de panic o timeout.
- Los logs de stdout/stderr se streamean en tiempo real a Victoria Logs etiquetados con `task_id`, `runtime` y `attempt`.
