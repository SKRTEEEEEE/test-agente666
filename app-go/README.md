# Hello World Go Application

A simple HTTP server written in Go that responds with "Hello World!" on the root endpoint.

## Features

- **HTTP Server**: Runs on port 8080
- **Root Endpoint** (`/`): Returns "Hello World!"
- **Health Check** (`/health`): Returns "OK" for health monitoring
- **Dockerized**: Multi-stage Docker build for optimized image size
- **Tested**: Comprehensive unit and integration tests

## Quick Start

### Using Docker (Recommended)

Build and run:
```bash
docker build -t hello-world-go:latest .
docker run -d -p 8080:8080 --name hello-world-go hello-world-go:latest
```

Test:
```bash
curl http://localhost:8080/          # Returns: Hello World!
curl http://localhost:8080/health     # Returns: OK
```

### Using Go Locally

If you have Go installed:

```bash
go mod download
go run main.go
```

## Testing

### Unit Tests

```bash
go test -v ./...
```

### Integration Tests

```bash
go test -v -tags=integration ./...
```

### Docker Tests

Tests are automatically executed during the Docker build process.

## Code Quality

### Format Code
```bash
go fmt ./...
```

### Static Analysis
```bash
go vet ./...
```

## Project Structure

```
.
├── main.go              # Main application code
├── main_test.go         # Unit tests
├── integration_test.go  # Integration tests
├── go.mod              # Go module definition
├── go.sum              # Go module checksums
├── Dockerfile          # Multi-stage Docker build
├── .dockerignore       # Docker build exclusions
└── README.md           # This file
```

## API Endpoints

### GET /

Returns a "Hello World!" message.

**Response:**
```
Hello World!
```

### GET /health

Health check endpoint for monitoring.

**Response:**
```
OK
```

## Dependencies

- Go 1.21+
- [testify](https://github.com/stretchr/testify) - Testing assertions

## Docker Image

The Docker image uses a multi-stage build:
1. **Builder stage**: Compiles the Go application and runs tests
2. **Runtime stage**: Minimal Alpine Linux image with only the compiled binary

Final image size: ~10MB

## License

Part of Agent666 test project.
