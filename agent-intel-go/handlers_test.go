package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	healthHandler(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Status = %v, want %v", resp.StatusCode, http.StatusOK)
	}

	var result map[string]interface{}
	err := json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result["status"] != "healthy" {
		t.Errorf("Status = %v, want healthy", result["status"])
	}

	// Check that MongoDB and NATS status are included
	if _, ok := result["mongodb"]; !ok {
		t.Error("MongoDB status missing from health check")
	}
	if _, ok := result["nats"]; !ok {
		t.Error("NATS status missing from health check")
	}
}

func TestGetNextTaskHandler(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create mock MongoDB client
	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Skip("MongoDB not available, skipping test")
	}
	defer client.Disconnect(ctx)

	db := client.Database("agent_intel_test")
	pendingCol := db.Collection("pending_tasks")
	defer pendingCol.Drop(ctx)

	// Insert test tasks
	testTasks := []*TaskMetrics{
		{
			TaskID:            "task-high-priority",
			Repository:        "test-repo",
			CreatedAt:         time.Now().Add(-48 * time.Hour), // Old = high priority
			LastSuccessAt:     time.Now().Add(-1 * time.Hour),
			AvgRuntimeMs:      5000,
			PendingTasksCount: 1,
			SizeBytes:         1024,
			Status:            StatusPending,
		},
		{
			TaskID:            "task-low-priority",
			Repository:        "test-repo",
			CreatedAt:         time.Now().Add(-1 * time.Hour), // New = low priority
			LastSuccessAt:     time.Now().Add(-72 * time.Hour),
			AvgRuntimeMs:      600000,
			PendingTasksCount: 10,
			SizeBytes:         1048576,
			Status:            StatusPending,
		},
		{
			TaskID:            "task-medium-priority",
			Repository:        "test-repo",
			CreatedAt:         time.Now().Add(-24 * time.Hour),
			LastSuccessAt:     time.Now().Add(-24 * time.Hour),
			AvgRuntimeMs:      60000,
			PendingTasksCount: 5,
			SizeBytes:         51200,
			Status:            StatusPending,
		},
	}

	for _, task := range testTasks {
		_, err := pendingCol.InsertOne(ctx, task)
		if err != nil {
			t.Fatalf("Failed to insert test task: %v", err)
		}
	}

	// Create test server with mock MongoDB
	service := &AgentIntelService{
		mongoDB:     db,
		mongoClient: client,
	}

	t.Run("get next task without repo_id filter", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/queue/next", nil)
		w := httptest.NewRecorder()

		service.getNextTaskHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Status = %v, want %v", resp.StatusCode, http.StatusOK)
		}

		var result NextTaskResponse
		err := json.NewDecoder(resp.Body).Decode(&result)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if result.Task == nil {
			t.Fatal("Expected task, got nil")
		}

		// Should return highest priority task (oldest)
		if result.Task.TaskID != "task-high-priority" {
			t.Errorf("TaskID = %v, want task-high-priority", result.Task.TaskID)
		}

		if result.Score <= 0 || result.Score > 1 {
			t.Errorf("Score = %v, must be between 0 and 1", result.Score)
		}
	})

	t.Run("get next task with repo_id filter", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/queue/next?repo_id=test-repo", nil)
		w := httptest.NewRecorder()

		service.getNextTaskHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Status = %v, want %v", resp.StatusCode, http.StatusOK)
		}

		var result NextTaskResponse
		err := json.NewDecoder(resp.Body).Decode(&result)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if result.Task == nil {
			t.Fatal("Expected task, got nil")
		}

		if result.Task.Repository != "test-repo" {
			t.Errorf("Repository = %v, want test-repo", result.Task.Repository)
		}
	})

	t.Run("get next task with no available tasks", func(t *testing.T) {
		// Clean up all tasks
		pendingCol.Drop(ctx)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/queue/next", nil)
		w := httptest.NewRecorder()

		service.getNextTaskHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Status = %v, want %v", resp.StatusCode, http.StatusNotFound)
		}

		var result map[string]string
		err := json.NewDecoder(resp.Body).Decode(&result)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if result["error"] != "no tasks available" {
			t.Errorf("Error message = %v, want 'no tasks available'", result["error"])
		}
	})
}

func TestCancelTaskHandler(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Skip("MongoDB not available, skipping test")
	}
	defer client.Disconnect(ctx)

	db := client.Database("agent_intel_test")
	pendingCol := db.Collection("pending_tasks")
	defer pendingCol.Drop(ctx)

	// Insert test task
	testTask := &TaskMetrics{
		TaskID:     "task-to-cancel",
		Repository: "test-repo",
		Status:     StatusPending,
		CreatedAt:  time.Now(),
	}
	_, err = pendingCol.InsertOne(ctx, testTask)
	if err != nil {
		t.Fatalf("Failed to insert test task: %v", err)
	}

	service := &AgentIntelService{
		mongoDB:     db,
		mongoClient: client,
	}

	t.Run("cancel existing task", func(t *testing.T) {
		reqBody := CancelTaskRequest{
			TaskID: "task-to-cancel",
			Reason: "Test cancellation",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/cancel", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		service.cancelTaskHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Status = %v, want %v", resp.StatusCode, http.StatusOK)
		}

		var result map[string]string
		err := json.NewDecoder(resp.Body).Decode(&result)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if result["message"] != "task cancelled successfully" {
			t.Errorf("Message = %v, want 'task cancelled successfully'", result["message"])
		}
	})

	t.Run("cancel non-existent task", func(t *testing.T) {
		reqBody := CancelTaskRequest{
			TaskID: "non-existent-task",
			Reason: "Test",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/cancel", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		service.cancelTaskHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Status = %v, want %v", resp.StatusCode, http.StatusNotFound)
		}
	})

	t.Run("cancel task with invalid request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/cancel", bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		service.cancelTaskHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Status = %v, want %v", resp.StatusCode, http.StatusBadRequest)
		}
	})
}

func TestMetricsHandler(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Skip("MongoDB not available, skipping test")
	}
	defer client.Disconnect(ctx)

	db := client.Database("agent_intel_test")
	pendingCol := db.Collection("pending_tasks")
	historyCol := db.Collection("task_history")
	defer pendingCol.Drop(ctx)
	defer historyCol.Drop(ctx)

	// Insert test data
	pendingTasks := []*TaskMetrics{
		{TaskID: "pending-1", Status: StatusPending, CreatedAt: time.Now()},
		{TaskID: "pending-2", Status: StatusPending, CreatedAt: time.Now()},
		{TaskID: "processing-1", Status: StatusProcessing, CreatedAt: time.Now()},
	}
	for _, task := range pendingTasks {
		pendingCol.InsertOne(ctx, task)
	}

	historyTasks := []*TaskMetrics{
		{TaskID: "completed-1", Status: StatusCompleted, CreatedAt: time.Now(), PipelineRuntimeMs: 5000},
		{TaskID: "completed-2", Status: StatusCompleted, CreatedAt: time.Now(), PipelineRuntimeMs: 10000},
		{TaskID: "failed-1", Status: StatusFailed, CreatedAt: time.Now()},
	}
	for _, task := range historyTasks {
		historyCol.InsertOne(ctx, task)
	}

	service := &AgentIntelService{
		mongoDB:     db,
		mongoClient: client,
	}

	t.Run("get system metrics", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics", nil)
		w := httptest.NewRecorder()

		service.metricsHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Status = %v, want %v", resp.StatusCode, http.StatusOK)
		}

		var result MetricsResponse
		err := json.NewDecoder(resp.Body).Decode(&result)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if result.TotalPending != 2 {
			t.Errorf("TotalPending = %v, want 2", result.TotalPending)
		}

		if result.TotalProcessing != 1 {
			t.Errorf("TotalProcessing = %v, want 1", result.TotalProcessing)
		}

		if result.TotalCompleted != 2 {
			t.Errorf("TotalCompleted = %v, want 2", result.TotalCompleted)
		}

		if result.TotalFailed != 1 {
			t.Errorf("TotalFailed = %v, want 1", result.TotalFailed)
		}

		expectedAvg := (5000 + 10000) / 2
		if result.AvgRuntimeMs != int64(expectedAvg) {
			t.Errorf("AvgRuntimeMs = %v, want %v", result.AvgRuntimeMs, expectedAvg)
		}
	})
}

func TestQueueStatusHandler(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Skip("MongoDB not available, skipping test")
	}
	defer client.Disconnect(ctx)

	db := client.Database("agent_intel_test")
	pendingCol := db.Collection("pending_tasks")
	defer pendingCol.Drop(ctx)

	// Insert tasks for different repositories
	tasks := []*TaskMetrics{
		{TaskID: "repo1-task1", Repository: "repo-1", Status: StatusPending, CreatedAt: time.Now()},
		{TaskID: "repo1-task2", Repository: "repo-1", Status: StatusPending, CreatedAt: time.Now()},
		{TaskID: "repo2-task1", Repository: "repo-2", Status: StatusPending, CreatedAt: time.Now()},
	}
	for _, task := range tasks {
		pendingCol.InsertOne(ctx, task)
	}

	service := &AgentIntelService{
		mongoDB:     db,
		mongoClient: client,
	}

	t.Run("get queue status", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/queue/status", nil)
		w := httptest.NewRecorder()

		service.queueStatusHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Status = %v, want %v", resp.StatusCode, http.StatusOK)
		}

		var result QueueStatusResponse
		err := json.NewDecoder(resp.Body).Decode(&result)
		if err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if result.TotalTasks != 3 {
			t.Errorf("TotalTasks = %v, want 3", result.TotalTasks)
		}

		if len(result.TasksByRepo) != 2 {
			t.Errorf("TasksByRepo count = %v, want 2", len(result.TasksByRepo))
		}

		if result.TasksByRepo["repo-1"] != 2 {
			t.Errorf("repo-1 tasks = %v, want 2", result.TasksByRepo["repo-1"])
		}

		if result.TasksByRepo["repo-2"] != 1 {
			t.Errorf("repo-2 tasks = %v, want 1", result.TasksByRepo["repo-2"])
		}
	})
}
