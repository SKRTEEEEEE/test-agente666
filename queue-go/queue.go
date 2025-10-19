package main

import (
	"errors"
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
	tasks       []*Task
	currentTask *Task
	mu          sync.RWMutex
}

// NewTaskQueue creates a new task queue
func NewTaskQueue() *TaskQueue {
	return &TaskQueue{
		tasks: make([]*Task, 0),
	}
}

// Enqueue adds a task to the queue
func (q *TaskQueue) Enqueue(task *Task) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.tasks = append(q.tasks, task)
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

	for _, task := range q.tasks {
		if task.ID == id {
			task.Status = status
			task.UpdatedAt = time.Now()
			return nil
		}
	}

	if q.currentTask != nil && q.currentTask.ID == id {
		q.currentTask.Status = status
		q.currentTask.UpdatedAt = time.Now()
		return nil
	}

	return errors.New("task not found")
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
