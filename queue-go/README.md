# queue-go

A Go HTTP server that provides task queue management for the Agent666 system with NATS JetStream integration.

## Features
- HTTP server running on port 8081
- RESTful API for task queue management
- **NATS JetStream integration** for distributed messaging
- Thread-safe queue operations with mutex locks
- Persistent task storage via JetStream
- Message delivery guarantees (at-least-once)
- Comprehensive test suite (unit, integration, and API tests)
- Health check endpoint
- Task status tracking (pending, in_progress, completed, failed)
- Graceful degradation to memory-only mode if Qdrant is unavailable
- Fully containerized with Docker
- API testing suite with HTTP examples (`api-test.http`)

## Architecture

The queue system uses NATS JetStream as a message broker:

```
┌─────────────┐         ┌──────────────┐         ┌──────────────┐
│  queue-go   │────────▶│     NATS     │────────▶│ queue-worker │
│  (API REST) │         │  JetStream   │         │  (Consumer)  │
│   :8081     │         │   :4222      │         │              │
└─────────────┘         └──────────────┘         └──────────────┘
     │                        │
     │                        ▼
     │                  Persistent Storage
     │                    (Volume)
     │
     └──▶ In-memory cache (backward compatibility)
```

**Flow:**
1. API receives HTTP request to create/update/delete task
2. Task is saved to in-memory cache
3. Message is published to NATS JetStream
4. Worker consumes message from JetStream
5. Worker processes task and acknowledges completion

## API Endpoints

**Health Check:**
- `GET /health` - Returns "OK" if service is running

**Queue Management:**
- `GET /api/queue/status` - Get queue statistics and current task
- `GET /api/tasks` - List all tasks in the queue
- `POST /api/tasks` - Create a new task
- `GET /api/tasks/{id}` - Get a specific task by ID
- `PATCH /api/tasks/{id}/status` - Update task status
- `DELETE /api/tasks/{id}` - Remove a task from the queue

## Running locally with Docker

Build the image:
```bash
docker build -t queue-go:latest ./queue-go
```

Run the container:
```bash
docker run -d -p 8081:8081 --name queue-go queue-go:latest
```

Test the endpoints:
```bash
# Health check
curl http://localhost:8081/health

# Get queue status
curl http://localhost:8081/api/queue/status

# Create a task
curl -X POST http://localhost:8081/api/tasks \
  -H "Content-Type: application/json" \
  -d '{"issue_id":"2","repository":"/test/repo","task_file_path":"/test/repo/docs/task/2-task.md"}'

# List all tasks
curl http://localhost:8081/api/tasks

# Get task by ID
curl http://localhost:8081/api/tasks/{task-id}

# Update task status
curl -X PATCH http://localhost:8081/api/tasks/{task-id}/status \
  -H "Content-Type: application/json" \
  -d '{"status":"in_progress"}'

# Delete a task
curl -X DELETE http://localhost:8081/api/tasks/{task-id}
```

## Response Examples

**Queue Status:**
```json
{
  "total_tasks": 5,
  "pending_tasks": 2,
  "in_progress_tasks": 1,
  "completed_tasks": 1,
  "failed_tasks": 1,
  "current_task": {
    "id": "abc-123",
    "issue_id": "2",
    "repository": "/test/repo",
    "task_file_path": "/test/repo/docs/task/2-task.md",
    "status": "in_progress",
    "created_at": "2025-10-19T12:00:00Z",
    "updated_at": "2025-10-19T12:05:00Z"
  }
}
```

**Task Object:**
```json
{
  "id": "abc-123",
  "issue_id": "2",
  "repository": "/test/repo",
  "task_file_path": "/test/repo/docs/task/2-task.md",
  "status": "pending",
  "created_at": "2025-10-19T12:00:00Z",
  "updated_at": "2025-10-19T12:00:00Z",
  "error_message": ""
}
```

## Task Statuses

- `pending` - Task is waiting to be processed
- `in_progress` - Task is currently being processed
- `completed` - Task finished successfully
- `failed` - Task failed with errors

## Development

The application includes:
- `main.go` - Main application entry point
- `queue.go` - Queue data structure and operations
- `handlers.go` - HTTP request handlers
- `persistence.go` - Qdrant vector database integration
- `queue_test.go` - Unit tests for queue operations
- `handlers_test.go` - Unit tests for HTTP handlers
- `integration_test.go` - Integration tests
- `api-test.http` - HTTP API examples for manual testing (use with REST Client extensions)
- `Dockerfile` - Multi-stage Docker build with test execution
- `go.mod` / `go.sum` - Go module dependencies

## Persistence

The queue service uses **Qdrant** vector database for persistence:
- Tasks are automatically saved to Qdrant when created, updated, or deleted
- On startup, the service loads all existing tasks from Qdrant
- If Qdrant is unavailable, the service falls back to memory-only mode
- Set `QDRANT_URL` environment variable to configure Qdrant location (default: `http://localhost:6333`)

**Benefits:**
- Tasks persist across service restarts
- Vector embeddings allow for future semantic search capabilities
- Scalable storage for large task queues

## Testing

Tests are automatically run during Docker build. To run tests manually:

```bash
cd queue-go
go test -v ./...
```

## Linting

Format code:
```bash
cd queue-go
go fmt ./...
```

Run static analysis:
```bash
cd queue-go
go vet ./...
```

Stop the container:
```bash
docker stop queue-go
docker rm queue-go
```
