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
- **queue-go**: http://localhost:8081 (Task queue service)
- **qdrant**: http://localhost:6333 (Vector database - Web UI at http://localhost:6333/dashboard)

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

A Go HTTP server that provides task queue management for the Agent666 system with vector database persistence.

#### Features
- HTTP server running on port 8081
- RESTful API for task queue management
- **Vector database persistence using Qdrant** - Tasks persist across restarts
- Thread-safe queue operations with mutex locks
- Comprehensive test suite (unit, integration, and API tests)
- Health check endpoint
- Task status tracking (pending, in_progress, completed, failed)
- Graceful degradation to memory-only mode if Qdrant is unavailable
- Fully containerized with Docker
- API testing suite with HTTP examples (`api-test.http`)

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

## License

This is a test project for Agent666.
