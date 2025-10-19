# agent-intel-go

Agent Intel Service - MVP implementation for intelligent task prioritization and queue management.

## Features
- **Event-Driven Architecture**: Consumes events from NATS JetStream for task lifecycle management
- **Intelligent Prioritization**: 5-metric scoring engine for optimal task ordering
- **MongoDB Persistence**: Dual collections for pending tasks and historical data
- **RESTful API**: Endpoints for task retrieval, cancellation, and system metrics
- **Idempotent Event Processing**: Prevents duplicate task processing
- **Graceful Degradation**: Continues operation if external services fail temporarily
- **Health Monitoring**: Built-in health checks for MongoDB and NATS connectivity

## Architecture

The Agent Intel Service acts as the brain of the task management system:

```
┌───────────────┐         ┌──────────────┐         ┌─────────────────┐
│  Orchestrator │────────▶│     NATS     │────────▶│  Agent Intel    │
│      CLI      │         │  JetStream   │         │    Service      │
│               │         │              │         │    :8082        │
└───────────────┘         └──────────────┘         └─────────────────┘
     │                           │                          │
     │                           │                          ▼
     │                           │                    ┌──────────┐
     │                           │                    │ MongoDB  │
     │                           │                    │  - pending_tasks
     │                           │                    │  - task_history
     └───────────────────────────┴────────────────────┤          │
                 GET /queue/next                      └──────────┘
```

**Event Flow:**
1. Orchestrator publishes `agent.task.new` event to NATS
2. Agent Intel Service consumes event and stores task in MongoDB
3. System calculates priority score automatically
4. Orchestrator queries `GET /queue/next` to retrieve highest priority task
5. Orchestrator executes pipeline and publishes `agent.pipeline.completed`
6. Agent Intel Service updates metrics and moves task to history

## Prioritization Engine

Tasks are scored using 5 weighted metrics (0-1 scale, higher = higher priority):

| Metric | Weight | Description |
|--------|--------|-------------|
| **Age** | 35% | How long the task has been pending (older = higher priority) |
| **Recent Activity** | 25% | Repository's last successful execution (more recent = higher priority) |
| **Runtime** | 20% | Average execution time (faster = higher priority) |
| **Load** | 10% | Number of pending tasks for the repository (lower = higher priority) |
| **Size** | 10% | Task file size (smaller = higher priority) |

**Example Score Calculation:**
- Task A (created 2 days ago, 5 sec runtime, 1 pending task) → **Score: 0.85** ✅
- Task B (created 1 hour ago, 1 hour runtime, 10 pending tasks) → **Score: 0.25**

## API Endpoints

**Health and Monitoring:**
- `GET /health` - Service health status (MongoDB + NATS connectivity)
- `GET /api/v1/metrics` - System-wide metrics (tasks processed, avg runtime, etc.)
- `GET /api/v1/queue/status` - Queue statistics grouped by repository and status

**Task Management:**
- `GET /api/v1/queue/next` - Retrieve next highest priority task
  - Query param: `?repo_id={ID}` - Filter by repository
  - Returns: Task object with calculated priority score
- `POST /api/v1/tasks/cancel` - Cancel a pending task
  - Body: `{"task_id": "...", "reason": "..."}`

## MongoDB Collections

**pending_tasks:**
```json
{
  "task_id": "task-123",
  "issue_id": "456",
  "repository": "/test/repo",
  "task_file_path": "/test/repo/docs/task/456-task.md",
  "created_at": "2025-10-19T12:00:00Z",
  "last_success_at": "2025-10-18T10:00:00Z",
  "avg_runtime_ms": 15000,
  "pending_tasks_count": 3,
  "size_bytes": 2048,
  "status": "pending",
  "assigned_at": null
}
```

**task_history:**
```json
{
  "task_id": "task-123",
  "status": "completed",
  "pipeline_runtime_ms": 14500,
  "created_at": "2025-10-19T12:00:00Z",
  "assigned_at": "2025-10-19T12:05:00Z"
}
```

## Running Locally

With Docker Compose (recommended):
```bash
docker-compose up -d agent-intel-go mongodb nats
```

Standalone with local MongoDB and NATS:
```bash
cd agent-intel-go
go run . 
# Or with custom config:
PORT=8082 MONGO_URL=mongodb://localhost:27017 NATS_URL=nats://localhost:4222 go run .
```

## Testing

**Unit tests (no external dependencies):**
```bash
cd agent-intel-go
go test -v -short ./...
```

**Full integration tests (requires MongoDB + NATS):**
```bash
# Start services first
docker-compose up -d mongodb nats

# Run all tests
go test -v ./...
```

## Configuration

Environment variables:
- `PORT` - HTTP server port (default: `8082`)
- `MONGO_URL` - MongoDB connection string (default: `mongodb://localhost:27017`)
- `NATS_URL` - NATS server URL (default: `nats://localhost:4222`)
- `DB_NAME` - MongoDB database name (default: `agent_intel`)

## Development

The service includes:
- `main.go` - Application entry point and server setup
- `types.go` - Data structures and constants
- `scoring.go` - Priority calculation engine
- `handlers.go` - HTTP request handlers
- `consumer.go` - NATS event consumer
- `scoring_test.go` - Unit tests for prioritization logic
- `handlers_test.go` - API endpoint tests
- `integration_test.go` - End-to-end integration tests
- `api-test.http` - Manual API testing examples

**Linting:**
```bash
go fmt ./...
go vet ./...
```
