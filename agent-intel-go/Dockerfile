# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Run unit tests only (skip integration tests that require external services)
RUN go test -v -short ./...

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o agent-intel .

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/agent-intel .

# Expose port
EXPOSE 8082

# Run the application
CMD ["./agent-intel"]
