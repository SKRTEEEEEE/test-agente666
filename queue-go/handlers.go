package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// CreateTaskRequest represents the request to create a task
type CreateTaskRequest struct {
	IssueID      string `json:"issue_id"`
	Repository   string `json:"repository"`
	TaskFilePath string `json:"task_file_path"`
}

// UpdateTaskStatusRequest represents the request to update task status
type UpdateTaskStatusRequest struct {
	Status string `json:"status"`
}

// QueueStatusResponse represents the queue status response
type QueueStatusResponse struct {
	TotalTasks      int   `json:"total_tasks"`
	PendingTasks    int   `json:"pending_tasks"`
	InProgressTasks int   `json:"in_progress_tasks"`
	CompletedTasks  int   `json:"completed_tasks"`
	FailedTasks     int   `json:"failed_tasks"`
	CurrentTask     *Task `json:"current_task,omitempty"`
}

// HealthHandler handles the health check endpoint
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "OK")
}

// GetQueueStatusHandler returns the current status of the queue
func GetQueueStatusHandler(w http.ResponseWriter, r *http.Request) {
	response := QueueStatusResponse{
		TotalTasks:      taskQueue.Size(),
		PendingTasks:    taskQueue.CountByStatus(StatusPending),
		InProgressTasks: taskQueue.CountByStatus(StatusInProgress),
		CompletedTasks:  taskQueue.CountByStatus(StatusCompleted),
		FailedTasks:     taskQueue.CountByStatus(StatusFailed),
		CurrentTask:     taskQueue.GetCurrentTask(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// ListTasksHandler returns all tasks in the queue
func ListTasksHandler(w http.ResponseWriter, r *http.Request) {
	tasks := taskQueue.ListTasks()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(tasks)
}

// CreateTaskHandler creates a new task and adds it to the queue
func CreateTaskHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateTaskRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.IssueID == "" || req.Repository == "" || req.TaskFilePath == "" {
		http.Error(w, "Missing required fields: issue_id, repository, task_file_path", http.StatusBadRequest)
		return
	}

	// Create new task
	task := &Task{
		ID:           uuid.New().String(),
		IssueID:      req.IssueID,
		Repository:   req.Repository,
		TaskFilePath: req.TaskFilePath,
		Status:       StatusPending,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Add to queue
	taskQueue.Enqueue(task)

	log.Printf("Task created: ID=%s, IssueID=%s, Repository=%s", task.ID, task.IssueID, task.Repository)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(task)
}

// GetTaskHandler returns a specific task by ID
func GetTaskHandler(w http.ResponseWriter, r *http.Request) {
	// Extract task ID from URL path
	path := strings.TrimPrefix(r.URL.Path, "/api/tasks/")
	taskID := strings.Split(path, "/")[0]

	task := taskQueue.GetTaskByID(taskID)
	if task == nil {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(task)
}

// UpdateTaskStatusHandler updates the status of a task
func UpdateTaskStatusHandler(w http.ResponseWriter, r *http.Request) {
	// Extract task ID from URL path
	path := strings.TrimPrefix(r.URL.Path, "/api/tasks/")
	parts := strings.Split(path, "/")
	taskID := parts[0]

	var req UpdateTaskStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate status
	validStatuses := []string{StatusPending, StatusInProgress, StatusCompleted, StatusFailed}
	isValid := false
	for _, status := range validStatuses {
		if req.Status == status {
			isValid = true
			break
		}
	}

	if !isValid {
		http.Error(w, "Invalid status", http.StatusBadRequest)
		return
	}

	// Update task status
	err := taskQueue.UpdateTaskStatus(taskID, req.Status)
	if err != nil {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	// Get updated task
	task := taskQueue.GetTaskByID(taskID)

	log.Printf("Task status updated: ID=%s, Status=%s", taskID, req.Status)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(task)
}

// DeleteTaskHandler removes a task from the queue
func DeleteTaskHandler(w http.ResponseWriter, r *http.Request) {
	// Extract task ID from URL path
	path := strings.TrimPrefix(r.URL.Path, "/api/tasks/")
	taskID := strings.Split(path, "/")[0]

	err := taskQueue.RemoveTask(taskID)
	if err != nil {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	log.Printf("Task deleted: ID=%s", taskID)

	w.WriteHeader(http.StatusNoContent)
}
