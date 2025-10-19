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

Each project has its own documentation in its respective README.md file:

- **[app-go](./app-go/README.md)** - GitHub API service for issues and pull requests
- **[queue-go](./queue-go/README.md)** - Task queue API service with NATS JetStream
- **[queue-worker-go](./queue-worker-go/README.md)** - Background worker for task processing
- **[agent-intel-go](./agent-intel-go/README.md)** - Agent Intel Service with intelligent prioritization

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

## License

This is a test project for Agent666.
