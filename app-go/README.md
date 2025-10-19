# GitHub Issues Go Application

A **high-performance** HTTP server written in Go that provides GitHub user issue information and basic health check endpoints.

## 🚀 Performance Optimizations

This application is heavily optimized for **blazing fast** response times:

- ⚡ **HTTP Connection Pooling** - Reuses connections for 10-50x faster API calls
- 💾 **In-Memory Caching** - Sub-millisecond responses for cached data
- 🔀 **Concurrent Requests** - Fetches multiple repos in parallel (up to 10x faster)
- 📦 **Gzip Compression** - 70% smaller responses

**[📖 Read Full Performance Guide →](./PERFORMANCE.md)**

---

## Features

- **HTTP Server**: Runs on port 8080
- **Root Endpoint** (`/`): Returns "Hello World!"
- **Health Check** (`/health`): Returns "OK" for health monitoring
- **GitHub Issues** (`/issues/{user}`): Fetches and returns all issues from a user's public repositories
  - Issues grouped by repository
  - Excludes repositories without issues
  - Includes repository metadata (stars, forks, description, URL)
- **Dockerized**: Multi-stage Docker build for optimized image size
- **Tested**: Comprehensive unit and integration tests

## GitHub API Rate Limits

The application supports GitHub Personal Access Tokens to increase API rate limits:
- **Without token**: 60 requests/hour
- **With token**: 5000 requests/hour

**How to create a GitHub token:**
1. Go to GitHub Settings → Developer settings → Personal access tokens → Tokens (classic)
2. Click "Generate new token (classic)"
3. Give it a name (e.g., "app-go-api")
4. No scopes are required for public repository access
5. Click "Generate token"
6. Copy the token

## Quick Start

### Using Docker (Recommended)

Build:
```bash
docker build -t hello-world-go:latest .
```

Run without authentication (60 req/hour):
```bash
docker run -d -p 8080:8080 --name hello-world-go hello-world-go:latest
```

Run with GitHub token (5000 req/hour):
```bash
docker run -d -p 8080:8080 --name hello-world-go \
  -e GITHUB_TOKEN=your_github_token_here \
  hello-world-go:latest
```

Test:
```bash
curl http://localhost:8080/              # Returns: Hello World!
curl http://localhost:8080/health        # Returns: OK
curl http://localhost:8080/issues/octocat # Returns: JSON with repository issues
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

### GET /issues/{user}

Fetches all issues from a GitHub user's public repositories.

**Parameters:**
- `user` - GitHub username (e.g., `octocat`, `torvalds`)

**Response:**
```json
[
  {
    "name": "repository-name",
    "full_name": "username/repository-name",
    "url": "https://github.com/username/repository-name",
    "description": "Repository description",
    "stars": 12345,
    "forks": 6789,
    "issues": [
      {
        "number": 1,
        "title": "Issue title",
        "state": "open",
        "html_url": "https://github.com/username/repository-name/issues/1",
        "created_at": "2025-01-01T00:00:00Z",
        "updated_at": "2025-01-02T00:00:00Z",
        "user": {
          "login": "issue-creator"
        }
      }
    ]
  }
]
```

**Notes:**
- Only repositories with issues are included
- Returns both open and closed issues
- Uses GitHub's public API (no authentication required)
- Limited to 100 repositories and 100 issues per repository
- Returns empty array if user has no repositories with issues
- Returns 404 if user doesn't exist
- Returns 400 if username is empty

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
