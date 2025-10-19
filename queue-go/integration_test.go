package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestQueueFullWorkflow tests the complete workflow of the queue system
func TestQueueFullWorkflow(t *testing.T) {
	// Initialize fresh queue
	taskQueue = NewTaskQueue()

	// Create a test server
	mux := http.NewServeMux()
	mux.HandleFunc("/health", HealthHandler)
	mux.HandleFunc("/api/queue/status", GetQueueStatusHandler)
	mux.HandleFunc("/api/tasks", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			ListTasksHandler(w, r)
		} else if r.Method == "POST" {
			CreateTaskHandler(w, r)
		}
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	// 1. Check health
	resp, err := http.Get(server.URL + "/health")
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 2. Check initial queue status
	resp, err = http.Get(server.URL + "/api/queue/status")
	if err != nil {
		t.Fatalf("Failed to get queue status: %v", err)
	}
	defer resp.Body.Close()

	var status QueueStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		t.Fatalf("Failed to decode status: %v", err)
	}

	if status.TotalTasks != 0 {
		t.Errorf("Expected 0 total tasks, got %d", status.TotalTasks)
	}

	// 3. Create first task
	task1Req := CreateTaskRequest{
		IssueID:      "1",
		Repository:   "/repo1",
		TaskFilePath: "/repo1/docs/task/1-task.md",
	}
	body, _ := json.Marshal(task1Req)
	resp, err = http.Post(server.URL+"/api/tasks", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Failed to create task 1: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 4. Create second task
	task2Req := CreateTaskRequest{
		IssueID:      "2",
		Repository:   "/repo2",
		TaskFilePath: "/repo2/docs/task/2-task.md",
	}
	body, _ = json.Marshal(task2Req)
	resp, err = http.Post(server.URL+"/api/tasks", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Failed to create task 2: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 5. List all tasks
	resp, err = http.Get(server.URL + "/api/tasks")
	if err != nil {
		t.Fatalf("Failed to list tasks: %v", err)
	}
	defer resp.Body.Close()

	var tasks []Task
	if err := json.NewDecoder(resp.Body).Decode(&tasks); err != nil {
		t.Fatalf("Failed to decode tasks: %v", err)
	}

	if len(tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(tasks))
	}

	// 6. Check updated queue status
	resp, err = http.Get(server.URL + "/api/queue/status")
	if err != nil {
		t.Fatalf("Failed to get queue status: %v", err)
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		t.Fatalf("Failed to decode status: %v", err)
	}

	if status.TotalTasks != 2 {
		t.Errorf("Expected 2 total tasks, got %d", status.TotalTasks)
	}

	if status.PendingTasks != 2 {
		t.Errorf("Expected 2 pending tasks, got %d", status.PendingTasks)
	}
}

// TestConcurrentTaskCreation tests creating tasks concurrently
func TestConcurrentTaskCreation(t *testing.T) {
	taskQueue = NewTaskQueue()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/tasks", CreateTaskHandler)

	server := httptest.NewServer(mux)
	defer server.Close()

	// Create tasks concurrently
	done := make(chan bool)
	for i := 0; i < 5; i++ {
		go func(id int) {
			taskReq := CreateTaskRequest{
				IssueID:      string(rune('0' + id)),
				Repository:   "/repo",
				TaskFilePath: "/repo/docs/task/task.md",
			}
			body, _ := json.Marshal(taskReq)
			resp, err := http.Post(server.URL+"/api/tasks", "application/json", bytes.NewBuffer(body))
			if err != nil {
				t.Errorf("Failed to create task: %v", err)
			}
			resp.Body.Close()
			done <- true
		}(i)
	}

	// Wait for all requests
	for i := 0; i < 5; i++ {
		<-done
	}

	if taskQueue.Size() != 5 {
		t.Errorf("Expected 5 tasks, got %d", taskQueue.Size())
	}
}

// TestTaskLifecycle tests the complete lifecycle of a task
func TestTaskLifecycle(t *testing.T) {
	taskQueue = NewTaskQueue()

	// Create task
	task := &Task{
		ID:           "lifecycle-test",
		IssueID:      "1",
		Repository:   "/test",
		TaskFilePath: "/test/docs/task/1.md",
		Status:       StatusPending,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	taskQueue.Enqueue(task)

	// Verify task is pending
	if task.Status != StatusPending {
		t.Errorf("Expected status %s, got %s", StatusPending, task.Status)
	}

	// Update to in-progress
	time.Sleep(10 * time.Millisecond)
	err := taskQueue.UpdateTaskStatus("lifecycle-test", StatusInProgress)
	if err != nil {
		t.Fatalf("Failed to update to in-progress: %v", err)
	}

	if task.Status != StatusInProgress {
		t.Errorf("Expected status %s, got %s", StatusInProgress, task.Status)
	}

	// Update to completed
	time.Sleep(10 * time.Millisecond)
	err = taskQueue.UpdateTaskStatus("lifecycle-test", StatusCompleted)
	if err != nil {
		t.Fatalf("Failed to update to completed: %v", err)
	}

	if task.Status != StatusCompleted {
		t.Errorf("Expected status %s, got %s", StatusCompleted, task.Status)
	}

	// Verify UpdatedAt was updated
	if task.UpdatedAt.Before(task.CreatedAt) {
		t.Error("UpdatedAt should be after CreatedAt")
	}
}

// TestQueueOrdering tests FIFO ordering of the queue
func TestQueueOrdering(t *testing.T) {
	taskQueue = NewTaskQueue()

	// Add tasks in order
	for i := 1; i <= 5; i++ {
		task := &Task{
			ID:        string(rune('0' + i)),
			IssueID:   string(rune('0' + i)),
			Status:    StatusPending,
			CreatedAt: time.Now(),
		}
		taskQueue.Enqueue(task)
		time.Sleep(1 * time.Millisecond) // Ensure different timestamps
	}

	// Dequeue and verify order
	for i := 1; i <= 5; i++ {
		task := taskQueue.Dequeue()
		if task == nil {
			t.Fatalf("Expected task at position %d, got nil", i)
		}

		expectedID := string(rune('0' + i))
		if task.ID != expectedID {
			t.Errorf("Expected task ID %s at position %d, got %s", expectedID, i, task.ID)
		}
	}

	// Verify queue is empty
	if taskQueue.Size() != 0 {
		t.Errorf("Expected empty queue, got size %d", taskQueue.Size())
	}
}

// TestErrorHandling tests error handling in the queue
func TestErrorHandling(t *testing.T) {
	taskQueue = NewTaskQueue()

	// Test updating non-existent task
	err := taskQueue.UpdateTaskStatus("nonexistent", StatusInProgress)
	if err == nil {
		t.Error("Expected error when updating non-existent task")
	}

	// Test removing non-existent task
	err = taskQueue.RemoveTask("nonexistent")
	if err == nil {
		t.Error("Expected error when removing non-existent task")
	}

	// Test getting non-existent task
	task := taskQueue.GetTaskByID("nonexistent")
	if task != nil {
		t.Error("Expected nil when getting non-existent task")
	}

	// Test dequeuing from empty queue
	task = taskQueue.Dequeue()
	if task != nil {
		t.Error("Expected nil when dequeuing from empty queue")
	}
}

// TestAPIErrorResponses tests error responses from API handlers
func TestAPIErrorResponses(t *testing.T) {
	taskQueue = NewTaskQueue()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/tasks/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			GetTaskHandler(w, r)
		} else if r.Method == "DELETE" {
			DeleteTaskHandler(w, r)
		} else if r.Method == "PATCH" {
			UpdateTaskStatusHandler(w, r)
		}
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	// Test getting non-existent task
	resp, err := http.Get(server.URL + "/api/tasks/nonexistent")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Test deleting non-existent task
	req, _ := http.NewRequest("DELETE", server.URL+"/api/tasks/nonexistent", nil)
	client := &http.Client{}
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

// TestQueueStatusCalculations tests queue status calculations
func TestQueueStatusCalculations(t *testing.T) {
	taskQueue = NewTaskQueue()

	// Add tasks with different statuses
	taskQueue.Enqueue(&Task{ID: "1", IssueID: "1", Status: StatusPending})
	taskQueue.Enqueue(&Task{ID: "2", IssueID: "2", Status: StatusPending})
	taskQueue.Enqueue(&Task{ID: "3", IssueID: "3", Status: StatusInProgress})
	taskQueue.Enqueue(&Task{ID: "4", IssueID: "4", Status: StatusCompleted})
	taskQueue.Enqueue(&Task{ID: "5", IssueID: "5", Status: StatusFailed})

	mux := http.NewServeMux()
	mux.HandleFunc("/api/queue/status", GetQueueStatusHandler)

	server := httptest.NewServer(mux)
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/queue/status")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	var status QueueStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if status.TotalTasks != 5 {
		t.Errorf("Expected 5 total tasks, got %d", status.TotalTasks)
	}

	if status.PendingTasks != 2 {
		t.Errorf("Expected 2 pending tasks, got %d", status.PendingTasks)
	}

	if status.InProgressTasks != 1 {
		t.Errorf("Expected 1 in-progress task, got %d", status.InProgressTasks)
	}

	if status.CompletedTasks != 1 {
		t.Errorf("Expected 1 completed task, got %d", status.CompletedTasks)
	}

	if status.FailedTasks != 1 {
		t.Errorf("Expected 1 failed task, got %d", status.FailedTasks)
	}
}
