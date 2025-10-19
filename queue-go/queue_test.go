package main

import (
	"encoding/json"
	"testing"
	"time"
)

// TestTaskCreation tests the creation of a Task
func TestTaskCreation(t *testing.T) {
	task := Task{
		ID:           "1",
		IssueID:      "2",
		Repository:   "/path/to/repo",
		TaskFilePath: "/path/to/repo/docs/task/2-mvp-go-queue.md",
		Status:       StatusPending,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if task.ID != "1" {
		t.Errorf("Expected task ID to be '1', got '%s'", task.ID)
	}

	if task.Status != StatusPending {
		t.Errorf("Expected task status to be '%s', got '%s'", StatusPending, task.Status)
	}

	if task.Repository == "" {
		t.Error("Expected repository path to be set")
	}
}

// TestQueueInitialization tests queue initialization
func TestQueueInitialization(t *testing.T) {
	queue := NewTaskQueue()

	if queue == nil {
		t.Fatal("Expected queue to be initialized")
	}

	if len(queue.tasks) != 0 {
		t.Errorf("Expected empty queue, got %d tasks", len(queue.tasks))
	}

	if queue.currentTask != nil {
		t.Error("Expected no current task on initialization")
	}
}

// TestEnqueueTask tests adding a task to the queue
func TestEnqueueTask(t *testing.T) {
	queue := NewTaskQueue()

	task := &Task{
		ID:           "1",
		IssueID:      "2",
		Repository:   "/path/to/repo",
		TaskFilePath: "/path/to/repo/docs/task/2-mvp-go-queue.md",
		Status:       StatusPending,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	queue.Enqueue(task)

	if len(queue.tasks) != 1 {
		t.Errorf("Expected 1 task in queue, got %d", len(queue.tasks))
	}

	if queue.tasks[0].ID != "1" {
		t.Errorf("Expected task ID '1', got '%s'", queue.tasks[0].ID)
	}
}

// TestDequeueTask tests removing a task from the queue
func TestDequeueTask(t *testing.T) {
	queue := NewTaskQueue()

	task1 := &Task{ID: "1", IssueID: "1", Status: StatusPending}
	task2 := &Task{ID: "2", IssueID: "2", Status: StatusPending}

	queue.Enqueue(task1)
	queue.Enqueue(task2)

	dequeuedTask := queue.Dequeue()

	if dequeuedTask == nil {
		t.Fatal("Expected to dequeue a task")
	}

	if dequeuedTask.ID != "1" {
		t.Errorf("Expected dequeued task ID '1', got '%s'", dequeuedTask.ID)
	}

	if len(queue.tasks) != 1 {
		t.Errorf("Expected 1 task remaining, got %d", len(queue.tasks))
	}
}

// TestDequeueEmptyQueue tests dequeuing from empty queue
func TestDequeueEmptyQueue(t *testing.T) {
	queue := NewTaskQueue()

	task := queue.Dequeue()

	if task != nil {
		t.Error("Expected nil when dequeuing from empty queue")
	}
}

// TestGetTaskByID tests retrieving a task by ID
func TestGetTaskByID(t *testing.T) {
	queue := NewTaskQueue()

	task := &Task{ID: "123", IssueID: "2", Status: StatusPending}
	queue.Enqueue(task)

	retrieved := queue.GetTaskByID("123")

	if retrieved == nil {
		t.Fatal("Expected to retrieve task")
	}

	if retrieved.ID != "123" {
		t.Errorf("Expected task ID '123', got '%s'", retrieved.ID)
	}
}

// TestGetTaskByIDNotFound tests retrieving non-existent task
func TestGetTaskByIDNotFound(t *testing.T) {
	queue := NewTaskQueue()

	task := queue.GetTaskByID("nonexistent")

	if task != nil {
		t.Error("Expected nil for non-existent task")
	}
}

// TestUpdateTaskStatus tests updating task status
func TestUpdateTaskStatus(t *testing.T) {
	queue := NewTaskQueue()

	task := &Task{
		ID:        "1",
		IssueID:   "2",
		Status:    StatusPending,
		UpdatedAt: time.Now().Add(-1 * time.Hour),
	}

	queue.Enqueue(task)

	oldUpdateTime := task.UpdatedAt
	time.Sleep(10 * time.Millisecond)

	err := queue.UpdateTaskStatus("1", StatusInProgress)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if task.Status != StatusInProgress {
		t.Errorf("Expected status '%s', got '%s'", StatusInProgress, task.Status)
	}

	if !task.UpdatedAt.After(oldUpdateTime) {
		t.Error("Expected UpdatedAt to be updated")
	}
}

// TestUpdateNonExistentTask tests updating non-existent task
func TestUpdateNonExistentTask(t *testing.T) {
	queue := NewTaskQueue()

	err := queue.UpdateTaskStatus("nonexistent", StatusInProgress)

	if err == nil {
		t.Error("Expected error when updating non-existent task")
	}
}

// TestListTasks tests listing all tasks
func TestListTasks(t *testing.T) {
	queue := NewTaskQueue()

	task1 := &Task{ID: "1", IssueID: "1", Status: StatusPending}
	task2 := &Task{ID: "2", IssueID: "2", Status: StatusInProgress}

	queue.Enqueue(task1)
	queue.Enqueue(task2)

	tasks := queue.ListTasks()

	if len(tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(tasks))
	}
}

// TestTaskStatusConstants tests task status constants
func TestTaskStatusConstants(t *testing.T) {
	statuses := []string{StatusPending, StatusInProgress, StatusCompleted, StatusFailed}

	for _, status := range statuses {
		if status == "" {
			t.Errorf("Expected non-empty status constant")
		}
	}
}

// TestTaskJSONSerialization tests JSON serialization of Task
func TestTaskJSONSerialization(t *testing.T) {
	task := Task{
		ID:           "1",
		IssueID:      "2",
		Repository:   "/path/to/repo",
		TaskFilePath: "/path/to/repo/docs/task/2-mvp-go-queue.md",
		Status:       StatusPending,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	data, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("Failed to marshal task: %v", err)
	}

	var unmarshaled Task
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal task: %v", err)
	}

	if unmarshaled.ID != task.ID {
		t.Errorf("Expected ID '%s', got '%s'", task.ID, unmarshaled.ID)
	}

	if unmarshaled.Status != task.Status {
		t.Errorf("Expected status '%s', got '%s'", task.Status, unmarshaled.Status)
	}
}

// TestConcurrentEnqueue tests concurrent task enqueueing
func TestConcurrentEnqueue(t *testing.T) {
	queue := NewTaskQueue()

	done := make(chan bool)

	// Enqueue tasks concurrently
	for i := 0; i < 10; i++ {
		go func(id int) {
			task := &Task{
				ID:        string(rune('0' + id)),
				IssueID:   string(rune('0' + id)),
				Status:    StatusPending,
				CreatedAt: time.Now(),
			}
			queue.Enqueue(task)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	if len(queue.tasks) != 10 {
		t.Errorf("Expected 10 tasks, got %d", len(queue.tasks))
	}
}

// TestQueueSize tests getting queue size
func TestQueueSize(t *testing.T) {
	queue := NewTaskQueue()

	if queue.Size() != 0 {
		t.Errorf("Expected size 0, got %d", queue.Size())
	}

	task1 := &Task{ID: "1", IssueID: "1", Status: StatusPending}
	task2 := &Task{ID: "2", IssueID: "2", Status: StatusPending}

	queue.Enqueue(task1)
	queue.Enqueue(task2)

	if queue.Size() != 2 {
		t.Errorf("Expected size 2, got %d", queue.Size())
	}

	queue.Dequeue()

	if queue.Size() != 1 {
		t.Errorf("Expected size 1, got %d", queue.Size())
	}
}

// TestRemoveTask tests removing a specific task from queue
func TestRemoveTask(t *testing.T) {
	queue := NewTaskQueue()

	task1 := &Task{ID: "1", IssueID: "1", Status: StatusPending}
	task2 := &Task{ID: "2", IssueID: "2", Status: StatusPending}
	task3 := &Task{ID: "3", IssueID: "3", Status: StatusPending}

	queue.Enqueue(task1)
	queue.Enqueue(task2)
	queue.Enqueue(task3)

	err := queue.RemoveTask("2")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if queue.Size() != 2 {
		t.Errorf("Expected size 2, got %d", queue.Size())
	}

	// Verify task2 is removed
	if queue.GetTaskByID("2") != nil {
		t.Error("Expected task2 to be removed")
	}

	// Verify other tasks remain
	if queue.GetTaskByID("1") == nil {
		t.Error("Expected task1 to remain")
	}

	if queue.GetTaskByID("3") == nil {
		t.Error("Expected task3 to remain")
	}
}

// TestRemoveNonExistentTask tests removing non-existent task
func TestRemoveNonExistentTask(t *testing.T) {
	queue := NewTaskQueue()

	err := queue.RemoveTask("nonexistent")

	if err == nil {
		t.Error("Expected error when removing non-existent task")
	}
}
