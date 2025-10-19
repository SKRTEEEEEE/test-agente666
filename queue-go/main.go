package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	natsClient "queue-go/nats"
)

// Global task queue (kept for backward compatibility and caching)
var taskQueue *TaskQueue

// Global NATS client
var nats *natsClient.Client

func main() {
	// Initialize task queue
	taskQueue = NewTaskQueue()

	// Initialize NATS client
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}

	var err error
	nats, err = natsClient.NewClient(natsURL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nats.Close()

	log.Println("Successfully initialized NATS JetStream client")

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down gracefully...")
		nats.Close()
		os.Exit(0)
	}()

	// Setup HTTP routes
	http.HandleFunc("/health", HealthHandler)
	http.HandleFunc("/api/queue/status", GetQueueStatusHandler)
	http.HandleFunc("/api/tasks", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			ListTasksHandler(w, r)
		case http.MethodPost:
			CreateTaskHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	http.HandleFunc("/api/tasks/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			GetTaskHandler(w, r)
		case http.MethodPatch:
			UpdateTaskStatusHandler(w, r)
		case http.MethodDelete:
			DeleteTaskHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	log.Printf("Queue API service starting on port %s...", port)
	log.Printf("NATS URL: %s", natsURL)
	log.Printf("Endpoints:")
	log.Printf("  GET  /health - Health check")
	log.Printf("  GET  /api/queue/status - Get queue status")
	log.Printf("  GET  /api/tasks - List all tasks")
	log.Printf("  POST /api/tasks - Create a new task (publishes to NATS)")
	log.Printf("  GET  /api/tasks/{id} - Get task by ID")
	log.Printf("  PATCH /api/tasks/{id}/status - Update task status (publishes to NATS)")
	log.Printf("  DELETE /api/tasks/{id} - Delete task (publishes to NATS)")

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
