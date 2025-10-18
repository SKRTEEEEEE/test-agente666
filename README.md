# Agent666 Test Project

This repository contains various applications developed as part of the Agent666 testing framework.

## Projects

### app-go

A Go HTTP server that provides GitHub user issue information and basic health checks.

#### Features
- HTTP server running on port 8080
- Root endpoint (`/`) returns "Hello World!"
- Health check endpoint (`/health`) returns "OK"
- GitHub issues endpoint (`/issues/{user}`) returns all issues from a user's public repositories, grouped by repository
- Fully containerized with Docker
- Comprehensive test suite (unit and integration tests)

#### Running locally with Docker

Build the image:
```bash
docker build -t hello-world-go:latest ./app-go
```

Run the container:
```bash
docker run -d -p 8080:8080 --name hello-world-go hello-world-go:latest
```

Test the endpoints:
```bash
curl http://localhost:8080/              # Returns: Hello World!
curl http://localhost:8080/health        # Returns: OK
curl http://localhost:8080/issues/octocat # Returns: JSON with issues grouped by repository
```

The `/issues/{user}` endpoint returns a JSON array of repositories with issues, where each repository includes:
- Repository name, full name, and URL
- Repository description
- Star and fork counts
- Array of issues with details (number, title, state, URL, timestamps, creator)

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

## License

This is a test project for Agent666.
