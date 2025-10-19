package main

import "time"

// Task status constants
const (
	StatusPending    = "pending"
	StatusAssigned   = "assigned"
	StatusProcessing = "processing"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
	StatusCancelled  = "cancelled"
)

// Score weights (must sum to 1.0)
const (
	weightAge            = 0.35 // 35% - Task age (older = higher priority)
	weightRecentActivity = 0.25 // 25% - Recent repository activity
	weightRuntime        = 0.20 // 20% - Average runtime (faster = higher priority)
	weightLoad           = 0.10 // 10% - Current load (less busy = higher priority)
	weightSize           = 0.10 // 10% - Task size (smaller = higher priority)
)

// Normalization constants
const (
	maxAgeHours      = 168.0   // 7 days in hours
	maxRecentHours   = 168.0   // 7 days in hours
	maxRuntimeMs     = 2400000 // 40 minutes in milliseconds
	maxPendingTasks  = 10      // Maximum expected concurrent tasks per repo
	maxSizeBytes     = 1048576 // 1MB in bytes
	defaultRuntimeMs = 60000   // Default 1 minute if unknown
	defaultSizeBytes = 51200   // Default 50KB if unknown
)

// TaskMetrics represents a task with all metrics for prioritization
type TaskMetrics struct {
	TaskID            string    `json:"task_id" bson:"task_id"`
	IssueID           string    `json:"issue_id" bson:"issue_id"`
	Repository        string    `json:"repository" bson:"repository"`
	TaskFilePath      string    `json:"task_file_path" bson:"task_file_path"`
	CreatedAt         time.Time `json:"created_at" bson:"created_at"`
	LastSuccessAt     time.Time `json:"last_success_at,omitempty" bson:"last_success_at,omitempty"`
	AvgRuntimeMs      int64     `json:"avg_runtime_ms" bson:"avg_runtime_ms"`
	PendingTasksCount int       `json:"pending_tasks_count" bson:"pending_tasks_count"`
	SizeBytes         int64     `json:"size_bytes" bson:"size_bytes"`
	PipelineRuntimeMs int64     `json:"pipeline_runtime_ms,omitempty" bson:"pipeline_runtime_ms,omitempty"`
	AssignedAt        time.Time `json:"assigned_at,omitempty" bson:"assigned_at,omitempty"`
	Status            string    `json:"status" bson:"status"`
	ErrorMessage      string    `json:"error_message,omitempty" bson:"error_message,omitempty"`
	CancelReason      string    `json:"cancel_reason,omitempty" bson:"cancel_reason,omitempty"`
}

// TaskNewEvent represents the event when a new task is created
type TaskNewEvent struct {
	TaskID       string    `json:"task_id"`
	IssueID      string    `json:"issue_id"`
	Repository   string    `json:"repository"`
	TaskFilePath string    `json:"task_file_path"`
	SizeBytes    int64     `json:"size_bytes"`
	CreatedAt    time.Time `json:"created_at"`
}

// PipelineCompletedEvent represents the event when a pipeline completes
type PipelineCompletedEvent struct {
	TaskID            string    `json:"task_id"`
	Repository        string    `json:"repository"`
	PipelineRuntimeMs int64     `json:"pipeline_runtime_ms"`
	Status            string    `json:"status"` // "success" or "failure"
	CompletedAt       time.Time `json:"completed_at"`
	ErrorMessage      string    `json:"error_message,omitempty"`
}

// NextTaskResponse represents the response for the next task endpoint
type NextTaskResponse struct {
	Task  *TaskMetrics `json:"task,omitempty"`
	Score float64      `json:"score,omitempty"`
}

// CancelTaskRequest represents a request to cancel a task
type CancelTaskRequest struct {
	TaskID string `json:"task_id"`
	Reason string `json:"reason"`
}

// MetricsResponse represents system metrics
type MetricsResponse struct {
	TotalPending    int64  `json:"total_pending"`
	TotalProcessing int64  `json:"total_processing"`
	TotalCompleted  int64  `json:"total_completed"`
	TotalFailed     int64  `json:"total_failed"`
	AvgRuntimeMs    int64  `json:"avg_runtime_ms"`
	TasksProcessed  int64  `json:"tasks_processed"`
	Timestamp       string `json:"timestamp"`
}

// QueueStatusResponse represents the queue status
type QueueStatusResponse struct {
	TotalTasks    int64          `json:"total_tasks"`
	TasksByRepo   map[string]int `json:"tasks_by_repo"`
	TasksByStatus map[string]int `json:"tasks_by_status"`
	Timestamp     string         `json:"timestamp"`
}

// HealthStatus represents health check status
type HealthStatus struct {
	Status  string                 `json:"status"`
	MongoDB string                 `json:"mongodb"`
	NATS    string                 `json:"nats"`
	Details map[string]interface{} `json:"details,omitempty"`
}
