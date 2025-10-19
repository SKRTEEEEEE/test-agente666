# Agent666 Test Project

This repository contains various applications developed as part of the Agent666 testing framework.

## Quick Start with Docker Compose

To run all services together:

```bash
# Build and start all services
docker-compose up -d

# Check status
docker-compose ps

# View logs
docker-compose logs -f

# Stop all services
docker-compose down
```

Services will be available at:
- **app-go**: http://localhost:8083 (GitHub API service)
- **queue-go**: http://localhost:8081 (Task queue API service)
- **queue-worker-go**: Background worker for processing tasks
- **agent-intel-go**: http://localhost:8082 (Agent Intel Service - Priority queue management)
- **nats**: http://localhost:8222 (NATS JetStream monitoring)
- **mongodb**: localhost:27017 (MongoDB database)

## Projects

### app-go

A Go HTTP server that provides GitHub user issue information and basic health checks.

#### Features
- HTTP server running on port 8080
- Root endpoint (`/`) returns "Hello World!"
- Health check endpoint (`/health`) returns "OK"
- GitHub issues endpoint (`/issues/{user}`) returns issues from a user's public repositories, grouped by repository
  - Query parameter support: `?q=open` to filter only open issues
- GitHub pull requests endpoint (`/pr/{user}`) returns pull requests from a user's public repositories, grouped by repository
  - Query parameter support: `?q=open` to filter only open pull requests
- Fully containerized with Docker
- Comprehensive test suite (unit and integration tests)

#### GitHub API Rate Limits

The application supports GitHub Personal Access Tokens to increase API rate limits:
- **Without token**: 60 requests/hour
- **With token**: 5000 requests/hour

To use a GitHub token, set the `GITHUB_TOKEN` environment variable.

**How to create a GitHub Personal Access Token:**
1. Go to GitHub Settings → Developer settings → Personal access tokens → Tokens (classic)
2. Click "Generate new token (classic)"
3. Give it a name (e.g., "app-go-api")
4. No scopes are required for public repository access
5. Click "Generate token"
6. Copy the token and use it with the `-e GITHUB_TOKEN=...` flag

#### Running locally with Docker

Build the image:
```bash
docker build -t hello-world-go:latest ./app-go
```

Run the container without authentication (60 req/hour):
```bash
docker run -d -p 8080:8080 --name hello-world-go hello-world-go:latest
```

Run the container with GitHub token (5000 req/hour):
```bash
docker run -d -p 8080:8080 --name hello-world-go \
  -e GITHUB_TOKEN=your_github_token_here \
  hello-world-go:latest
```

Test the endpoints:
```bash
curl http://localhost:8080/                    # Returns: Hello World!
curl http://localhost:8080/health              # Returns: OK
curl http://localhost:8080/issues/SKRTEEEEEE      # Returns: JSON with all issues grouped by repository
curl http://localhost:8080/issues/SKRTEEEEEE?q=open  # Returns: JSON with only open issues
curl http://localhost:8080/pr/SKRTEEEEEE          # Returns: JSON with all pull requests grouped by repository
curl http://localhost:8080/pr/SKRTEEEEEE?q=open   # Returns: JSON with only open pull requests
```

The `/issues/{user}` endpoint returns a JSON array of repositories with issues, where each repository includes:
- Repository name, full name, and URL
- Repository description
- Star and fork counts
- Array of issues with details (number, title, state, URL, timestamps, creator)

The `/pr/{user}` endpoint returns a JSON array of repositories with pull requests, where each repository includes:
- Repository name, full name, and URL
- Repository description
- Star and fork counts
- Array of pull requests with details (number, title, state, URL, timestamps, creator, merged_at)

Stop the container:
```bash
docker stop hello-world-go
docker rm hello-world-go
```

#### Development

The application includes:
- `main.go` - Main application code
- `main_test.go` - Unit tests
- `integration_test.go` - Integration tests
- `Dockerfile` - Multi-stage Docker build with test execution
- `go.mod` / `go.sum` - Go module dependencies

#### Testing

Tests are automatically run during Docker build. To run tests manually:

```bash
docker run --rm -v "$(pwd)/app-go:/app" -w /app golang:1.21-alpine go test -v ./...
```

#### Linting

Format code:
```bash
docker run --rm -v "$(pwd)/app-go:/app" -w /app golang:1.21-alpine go fmt ./...
```

Run static analysis:
```bash
docker run --rm -v "$(pwd)/app-go:/app" -w /app golang:1.21-alpine go vet ./...
```

### queue-go

A Go HTTP server that provides task queue management for the Agent666 system with NATS JetStream integration.

#### Features
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

#### Architecture

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

#### API Endpoints

**Health Check:**
- `GET /health` - Returns "OK" if service is running

**Queue Management:**
- `GET /api/queue/status` - Get queue statistics and current task
- `GET /api/tasks` - List all tasks in the queue
- `POST /api/tasks` - Create a new task
- `GET /api/tasks/{id}` - Get a specific task by ID
- `PATCH /api/tasks/{id}/status` - Update task status
- `DELETE /api/tasks/{id}` - Remove a task from the queue

#### Running locally with Docker

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

#### Response Examples

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

#### Task Statuses

- `pending` - Task is waiting to be processed
- `in_progress` - Task is currently being processed
- `completed` - Task finished successfully
- `failed` - Task failed with errors

#### Development

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

#### Persistence

The queue service uses **Qdrant** vector database for persistence:
- Tasks are automatically saved to Qdrant when created, updated, or deleted
- On startup, the service loads all existing tasks from Qdrant
- If Qdrant is unavailable, the service falls back to memory-only mode
- Set `QDRANT_URL` environment variable to configure Qdrant location (default: `http://localhost:6333`)

**Benefits:**
- Tasks persist across service restarts
- Vector embeddings allow for future semantic search capabilities
- Scalable storage for large task queues

#### Testing

Tests are automatically run during Docker build. To run tests manually:

```bash
cd queue-go
go test -v ./...
```

#### Linting

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

### queue-worker-go

Background worker service that consumes and processes tasks from NATS JetStream.

#### Features
- Consumes tasks from NATS JetStream stream
- Durable consumer with automatic reconnection
- Processes tasks asynchronously
- Automatic message acknowledgment
- Retry logic for failed messages (up to 3 attempts)
- Graceful shutdown handling
- Fully containerized with Docker

#### How it works

The worker continuously fetches messages from the NATS JetStream `TASKS` stream:

1. **Subscription**: Subscribes to the `tasks.new` subject with a durable consumer named `task-workers`
2. **Batch fetching**: Fetches up to 10 messages at a time for efficient processing
3. **Processing**: Processes each task (simulated in MVP, can be extended for real work)
4. **Acknowledgment**: Sends ACK to NATS after successful processing
5. **Retry**: If processing fails, message is redelivered (max 3 times)

#### Configuration

Environment variables:
- `NATS_URL`: NATS server URL (default: `nats://localhost:4222`)

#### Running standalone

```bash
cd queue-worker-go
go run main.go
```

Or with Docker:

```bash
docker build -t queue-worker-go:latest ./queue-worker-go
docker run -d --name queue-worker \
  -e NATS_URL=nats://localhost:4222 \
  --network agent666-network \
  queue-worker-go:latest
```

#### Logs

View worker logs to see task processing:

```bash
docker logs -f agent666-queue-worker-go
```

Sample output:
```
2025/10/19 15:18:09 Queue Worker starting...
2025/10/19 15:18:09 Successfully connected to NATS
2025/10/19 15:18:09 Successfully subscribed to task queue
2025/10/19 15:18:09 Waiting for tasks...
2025/10/19 15:18:57 Received task: {"id":"abc-123",...}
2025/10/19 15:18:58 Task processed successfully
```

## NATS JetStream

The project uses NATS JetStream as a distributed messaging system for reliable task queue management.

### Why NATS JetStream?

**Benefits:**
- ✅ **Persistence**: Messages survive restarts
- ✅ **Delivery guarantees**: At-least-once delivery with acknowledgments
- ✅ **Scalability**: Horizontal scaling with multiple workers
- ✅ **Fault tolerance**: Automatic message redelivery on failure
- ✅ **Monitoring**: Built-in monitoring endpoint

**Comparison with in-memory queue:**

| Feature | In-Memory Queue | NATS JetStream |
|---------|----------------|----------------|
| Persistence | ❌ Lost on restart | ✅ Persisted to disk |
| Delivery guarantees | ❌ No guarantees | ✅ At-least-once |
| Scalability | ❌ Single instance | ✅ Multiple workers |
| Monitoring | ⚠️ Basic | ✅ Full metrics |
| Replay capability | ❌ No | ✅ Yes |

### Configuration

NATS JetStream is configured in `docker-compose.yml`:

```yaml
nats:
  image: nats:latest
  command: ["-js", "-sd", "/data", "-m", "8222"]
  ports:
    - "4222:4222"  # Client connections
    - "8222:8222"  # HTTP monitoring
  volumes:
    - nats-data:/data  # Persistent storage
```

### Stream Configuration

**Stream Name**: `TASKS`
- **Subjects**: `tasks.*` (tasks.new, tasks.update, tasks.delete, tasks.status)
- **Retention**: WorkQueue policy (messages deleted after acknowledgment)
- **Storage**: File (persisted to disk)
- **Max Age**: 7 days

**Consumer**: `task-workers`
- **Durable**: Yes (survives restarts)
- **Ack Policy**: Explicit (manual acknowledgment required)
- **Max Deliver**: 3 attempts
- **Ack Wait**: 30 seconds

### Monitoring

Access NATS monitoring at http://localhost:8222:

- **Health check**: http://localhost:8222/healthz
- **Server info**: http://localhost:8222/varz
- **Connection info**: http://localhost:8222/connz
- **JetStream info**: http://localhost:8222/jsz

### Testing the Message Flow

Create a task and watch it flow through the system:

```bash
# 1. Create a task
curl -X POST http://localhost:8081/api/tasks \
  -H "Content-Type: application/json" \
  -d '{"issue_id":"123","repository":"/test/repo","task_file_path":"/test/repo/task.md"}'

# 2. Check queue status
curl http://localhost:8081/api/queue/status

# 3. View worker logs to see processing
docker logs -f agent666-queue-worker-go

# 4. Check NATS stream info
curl http://localhost:8222/jsz?streams=true
```

### Troubleshooting

**NATS not connecting:**
```bash
# Check NATS logs
docker logs agent666-nats

# Verify NATS is healthy
curl http://localhost:8222/healthz
```

**Worker not processing tasks:**
```bash
# Check worker logs
docker logs agent666-queue-worker-go

# Verify worker is subscribed
# Should see: "Successfully subscribed to task queue"
```

**Messages stuck in queue:**
```bash
# Check stream state via monitoring endpoint
curl http://localhost:8222/jsz?streams=true

# Or restart worker to trigger redelivery
docker restart agent666-queue-worker-go
```

### Development

For local development without Docker:

```bash
# Terminal 1: Start NATS
docker run -p 4222:4222 -p 8222:8222 nats:latest -js -m 8222

# Terminal 2: Start queue API
cd queue-go
go run .

# Terminal 3: Start worker
cd queue-worker-go
go run .

# Terminal 4: Test
curl -X POST http://localhost:8081/api/tasks -H "Content-Type: application/json" -d '{"issue_id":"1","repository":"/test","task_file_path":"/test/task.md"}'
```

### agent-intel-go

Agent Intel Service - MVP implementation for intelligent task prioritization and queue management.

#### Features
- **Event-Driven Architecture**: Consumes events from NATS JetStream for task lifecycle management
- **Intelligent Prioritization**: 5-metric scoring engine for optimal task ordering
- **MongoDB Persistence**: Dual collections for pending tasks and historical data
- **RESTful API**: Endpoints for task retrieval, cancellation, and system metrics
- **Idempotent Event Processing**: Prevents duplicate task processing
- **Graceful Degradation**: Continues operation if external services fail temporarily
- **Health Monitoring**: Built-in health checks for MongoDB and NATS connectivity

#### Architecture

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

#### Prioritization Engine

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

#### API Endpoints

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

#### MongoDB Collections

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

#### Running Locally

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

#### Testing

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

#### Configuration

Environment variables:
- `PORT` - HTTP server port (default: `8082`)
- `MONGO_URL` - MongoDB connection string (default: `mongodb://localhost:27017`)
- `NATS_URL` - NATS server URL (default: `nats://localhost:4222`)
- `DB_NAME` - MongoDB database name (default: `agent_intel`)

#### Development

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

## License

This is a test project for Agent666.
