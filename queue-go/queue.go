package main

import (
	"errors"
	"log"
	"sync"
	"time"
)

// Task status constants
const (
	StatusPending    = "pending"
	StatusInProgress = "in_progress"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
)

// Task represents a task in the queue
type Task struct {
	ID           string    `json:"id"`
	IssueID      string    `json:"issue_id"`
	Repository   string    `json:"repository"`
	TaskFilePath string    `json:"task_file_path"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	ErrorMessage string    `json:"error_message,omitempty"`
}

// TaskQueue represents the queue of tasks
type TaskQueue struct {
	tasks          []*Task
	currentTask    *Task
	mu             sync.RWMutex
	qdrant         *QdrantClient
	usePersistence bool
}

// NewTaskQueue creates a new task queue
func NewTaskQueue() *TaskQueue {
	q := &TaskQueue{
		tasks:  make([]*Task, 0),
		qdrant: NewQdrantClient(),
	}

	// Try to initialize Qdrant
	if q.qdrant.IsAvailable() {
		if err := q.qdrant.InitializeCollection(); err != nil {
			log.Printf("Warning: Failed to initialize Qdrant: %v", err)
			q.usePersistence = false
		} else {
			q.usePersistence = true
			log.Println("Qdrant persistence enabled")

			// Load existing tasks from Qdrant
			if err := q.loadTasksFromPersistence(); err != nil {
				log.Printf("Warning: Failed to load tasks from Qdrant: %v", err)
			}
		}
	} else {
		log.Println("Qdrant not available, running in memory-only mode")
		q.usePersistence = false
	}

	return q
}

// loadTasksFromPersistence loads all tasks from Qdrant into memory
func (q *TaskQueue) loadTasksFromPersistence() error {
	if !q.usePersistence {
		return nil
	}

	tasks, err := q.qdrant.ListAllTasks()
	if err != nil {
		return err
	}

	q.tasks = tasks
	log.Printf("Loaded %d tasks from Qdrant", len(tasks))
	return nil
}

// Enqueue adds a task to the queue
func (q *TaskQueue) Enqueue(task *Task) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.tasks = append(q.tasks, task)

	// Save to Qdrant if persistence is enabled
	if q.usePersistence {
		if err := q.qdrant.SaveTask(task); err != nil {
			log.Printf("Warning: Failed to save task to Qdrant: %v", err)
		}
	}
}

// Dequeue removes and returns the first task from the queue
func (q *TaskQueue) Dequeue() *Task {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.tasks) == 0 {
		return nil
	}

	task := q.tasks[0]
	q.tasks = q.tasks[1:]
	return task
}

// GetTaskByID returns a task by its ID
func (q *TaskQueue) GetTaskByID(id string) *Task {
	q.mu.RLock()
	defer q.mu.RUnlock()

	for _, task := range q.tasks {
		if task.ID == id {
			return task
		}
	}

	if q.currentTask != nil && q.currentTask.ID == id {
		return q.currentTask
	}

	return nil
}

// UpdateTaskStatus updates the status of a task
func (q *TaskQueue) UpdateTaskStatus(id string, status string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	var updatedTask *Task

	for _, task := range q.tasks {
		if task.ID == id {
			task.Status = status
			task.UpdatedAt = time.Now()
			updatedTask = task
			break
		}
	}

	if updatedTask == nil && q.currentTask != nil && q.currentTask.ID == id {
		q.currentTask.Status = status
		q.currentTask.UpdatedAt = time.Now()
		updatedTask = q.currentTask
	}

	if updatedTask == nil {
		return errors.New("task not found")
	}

	// Update in Qdrant if persistence is enabled
	if q.usePersistence {
		if err := q.qdrant.UpdateTask(updatedTask); err != nil {
			log.Printf("Warning: Failed to update task in Qdrant: %v", err)
		}
	}

	return nil
}

// ListTasks returns all tasks in the queue
func (q *TaskQueue) ListTasks() []*Task {
	q.mu.RLock()
	defer q.mu.RUnlock()

	// Create a copy to avoid external modifications
	tasks := make([]*Task, len(q.tasks))
	copy(tasks, q.tasks)
	return tasks
}

// Size returns the number of tasks in the queue
func (q *TaskQueue) Size() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.tasks)
}

// RemoveTask removes a task from the queue by ID
func (q *TaskQueue) RemoveTask(id string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	for i, task := range q.tasks {
		if task.ID == id {
			q.tasks = append(q.tasks[:i], q.tasks[i+1:]...)

			// Delete from Qdrant if persistence is enabled
			if q.usePersistence {
				if err := q.qdrant.DeleteTask(id); err != nil {
					log.Printf("Warning: Failed to delete task from Qdrant: %v", err)
				}
			}

			return nil
		}
	}

	return errors.New("task not found")
}

// SetCurrentTask sets the current task being processed
func (q *TaskQueue) SetCurrentTask(task *Task) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.currentTask = task
}

// GetCurrentTask returns the current task being processed
func (q *TaskQueue) GetCurrentTask() *Task {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.currentTask
}

// CountByStatus returns the count of tasks by status
func (q *TaskQueue) CountByStatus(status string) int {
	q.mu.RLock()
	defer q.mu.RUnlock()

	count := 0
	for _, task := range q.tasks {
		if task.Status == status {
			count++
		}
	}

	if q.currentTask != nil && q.currentTask.Status == status {
		count++
	}

	return count
}
