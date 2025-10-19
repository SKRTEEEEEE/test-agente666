package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestHealthHandler tests the health endpoint
func TestHealthHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(HealthHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	expected := "OK"
	if rr.Body.String() != expected {
		t.Errorf("Handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}
}

// TestGetQueueStatusHandler tests getting queue status
func TestGetQueueStatusHandler(t *testing.T) {
	// Initialize global queue for testing
	taskQueue = NewTaskQueue()

	task := &Task{
		ID:           "1",
		IssueID:      "2",
		Repository:   "/test/repo",
		TaskFilePath: "/test/repo/docs/task/2-test.md",
		Status:       StatusPending,
	}
	taskQueue.Enqueue(task)

	req, err := http.NewRequest("GET", "/api/queue/status", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(GetQueueStatusHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response QueueStatusResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.TotalTasks != 1 {
		t.Errorf("Expected 1 total task, got %d", response.TotalTasks)
	}

	if response.PendingTasks != 1 {
		t.Errorf("Expected 1 pending task, got %d", response.PendingTasks)
	}
}

// TestListTasksHandler tests listing all tasks
func TestListTasksHandler(t *testing.T) {
	taskQueue = NewTaskQueue()

	task1 := &Task{ID: "1", IssueID: "1", Status: StatusPending}
	task2 := &Task{ID: "2", IssueID: "2", Status: StatusInProgress}

	taskQueue.Enqueue(task1)
	taskQueue.Enqueue(task2)

	req, err := http.NewRequest("GET", "/api/tasks", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(ListTasksHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var tasks []Task
	err = json.Unmarshal(rr.Body.Bytes(), &tasks)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(tasks))
	}
}

// TestCreateTaskHandler tests creating a new task
func TestCreateTaskHandler(t *testing.T) {
	taskQueue = NewTaskQueue()

	createReq := CreateTaskRequest{
		IssueID:      "2",
		Repository:   "/test/repo",
		TaskFilePath: "/test/repo/docs/task/2-test.md",
	}

	body, err := json.Marshal(createReq)
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("POST", "/api/tasks", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(CreateTaskHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusCreated)
	}

	var task Task
	err = json.Unmarshal(rr.Body.Bytes(), &task)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if task.IssueID != "2" {
		t.Errorf("Expected issue ID '2', got '%s'", task.IssueID)
	}

	if task.Status != StatusPending {
		t.Errorf("Expected status '%s', got '%s'", StatusPending, task.Status)
	}

	// Verify task was added to queue
	if taskQueue.Size() != 1 {
		t.Errorf("Expected 1 task in queue, got %d", taskQueue.Size())
	}
}

// TestCreateTaskHandlerInvalidJSON tests creating task with invalid JSON
func TestCreateTaskHandlerInvalidJSON(t *testing.T) {
	taskQueue = NewTaskQueue()

	req, err := http.NewRequest("POST", "/api/tasks", bytes.NewBufferString("invalid json"))
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(CreateTaskHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}
}

// TestCreateTaskHandlerMissingFields tests creating task with missing fields
func TestCreateTaskHandlerMissingFields(t *testing.T) {
	taskQueue = NewTaskQueue()

	createReq := CreateTaskRequest{
		IssueID: "2",
		// Missing Repository and TaskFilePath
	}

	body, err := json.Marshal(createReq)
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("POST", "/api/tasks", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(CreateTaskHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}
}

// TestGetTaskHandler tests getting a specific task
func TestGetTaskHandler(t *testing.T) {
	taskQueue = NewTaskQueue()

	task := &Task{ID: "123", IssueID: "2", Status: StatusPending}
	taskQueue.Enqueue(task)

	req, err := http.NewRequest("GET", "/api/tasks/123", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(GetTaskHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var returnedTask Task
	err = json.Unmarshal(rr.Body.Bytes(), &returnedTask)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if returnedTask.ID != "123" {
		t.Errorf("Expected task ID '123', got '%s'", returnedTask.ID)
	}
}

// TestGetTaskHandlerNotFound tests getting non-existent task
func TestGetTaskHandlerNotFound(t *testing.T) {
	taskQueue = NewTaskQueue()

	req, err := http.NewRequest("GET", "/api/tasks/nonexistent", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(GetTaskHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}
}

// TestUpdateTaskStatusHandler tests updating task status
func TestUpdateTaskStatusHandler(t *testing.T) {
	taskQueue = NewTaskQueue()

	task := &Task{ID: "123", IssueID: "2", Status: StatusPending}
	taskQueue.Enqueue(task)

	updateReq := UpdateTaskStatusRequest{
		Status: StatusInProgress,
	}

	body, err := json.Marshal(updateReq)
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("PATCH", "/api/tasks/123/status", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(UpdateTaskStatusHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var updatedTask Task
	err = json.Unmarshal(rr.Body.Bytes(), &updatedTask)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if updatedTask.Status != StatusInProgress {
		t.Errorf("Expected status '%s', got '%s'", StatusInProgress, updatedTask.Status)
	}
}

// TestDeleteTaskHandler tests deleting a task
func TestDeleteTaskHandler(t *testing.T) {
	taskQueue = NewTaskQueue()

	task := &Task{ID: "123", IssueID: "2", Status: StatusPending}
	taskQueue.Enqueue(task)

	req, err := http.NewRequest("DELETE", "/api/tasks/123", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(DeleteTaskHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNoContent {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusNoContent)
	}

	// Verify task was removed
	if taskQueue.GetTaskByID("123") != nil {
		t.Error("Expected task to be removed")
	}
}

// TestDeleteTaskHandlerNotFound tests deleting non-existent task
func TestDeleteTaskHandlerNotFound(t *testing.T) {
	taskQueue = NewTaskQueue()

	req, err := http.NewRequest("DELETE", "/api/tasks/nonexistent", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(DeleteTaskHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}
}
