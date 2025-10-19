package main

import (
	"log"
	"net/http"
	"os"
)

// Global task queue
var taskQueue *TaskQueue

func main() {
	// Initialize task queue
	taskQueue = NewTaskQueue()

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

	log.Printf("Queue service starting on port %s...", port)
	log.Printf("Endpoints:")
	log.Printf("  GET  /health - Health check")
	log.Printf("  GET  /api/queue/status - Get queue status")
	log.Printf("  GET  /api/tasks - List all tasks")
	log.Printf("  POST /api/tasks - Create a new task")
	log.Printf("  GET  /api/tasks/{id} - Get task by ID")
	log.Printf("  PATCH /api/tasks/{id}/status - Update task status")
	log.Printf("  DELETE /api/tasks/{id} - Delete task")

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
