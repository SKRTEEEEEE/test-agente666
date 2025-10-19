# queue-worker-go

Background worker service that consumes and processes tasks from NATS JetStream.

## Features
- Consumes tasks from NATS JetStream stream
- Durable consumer with automatic reconnection
- Processes tasks asynchronously
- Automatic message acknowledgment
- Retry logic for failed messages (up to 3 attempts)
- Graceful shutdown handling
- Fully containerized with Docker

## How it works

The worker continuously fetches messages from the NATS JetStream `TASKS` stream:

1. **Subscription**: Subscribes to the `tasks.new` subject with a durable consumer named `task-workers`
2. **Batch fetching**: Fetches up to 10 messages at a time for efficient processing
3. **Processing**: Processes each task (simulated in MVP, can be extended for real work)
4. **Acknowledgment**: Sends ACK to NATS after successful processing
5. **Retry**: If processing fails, message is redelivered (max 3 times)

## Configuration

Environment variables:
- `NATS_URL`: NATS server URL (default: `nats://localhost:4222`)

## Running standalone

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

## Logs

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
