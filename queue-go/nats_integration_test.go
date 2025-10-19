//go:build integration
// +build integration

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	natsClient "queue-go/nats"

	"github.com/nats-io/nats.go"
)

// TestNATSIntegration tests the full flow with NATS JetStream
func TestNATSIntegration(t *testing.T) {
	// Skip if NATS_URL not provided
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		t.Skip("Skipping integration test: NATS_URL not set")
	}

	// Initialize NATS client
	client, err := natsClient.NewClient(natsURL)
	if err != nil {
		t.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer client.Close()

	// Verify stream exists
	streamInfo, err := client.GetStreamInfo()
	if err != nil {
		t.Fatalf("Failed to get stream info: %v", err)
	}

	if streamInfo.Config.Name != "TASKS" {
		t.Errorf("Expected stream name to be TASKS, got %s", streamInfo.Config.Name)
	}

	t.Logf("Stream info: %+v", streamInfo.State)
}

// TestPublishAndConsume tests publishing and consuming messages
func TestPublishAndConsume(t *testing.T) {
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		t.Skip("Skipping integration test: NATS_URL not set")
	}

	// Initialize NATS client
	client, err := natsClient.NewClient(natsURL)
	if err != nil {
		t.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer client.Close()

	// Create a test task
	task := &natsClient.TaskMessage{
		ID:           "test-task-1",
		IssueID:      "999",
		Repository:   "/test/repo",
		TaskFilePath: "/test/repo/docs/task/999-test.md",
		Status:       "pending",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Publish task
	err = client.PublishNewTask(task)
	if err != nil {
		t.Fatalf("Failed to publish task: %v", err)
	}

	t.Log("Task published successfully")

	// Create consumer
	err = client.CreateConsumer()
	if err != nil {
		t.Fatalf("Failed to create consumer: %v", err)
	}

	// Subscribe and consume the message
	sub, err := client.Subscribe(func(receivedTask *natsClient.TaskMessage) error {
		if receivedTask.ID != task.ID {
			return fmt.Errorf("Expected task ID %s, got %s", task.ID, receivedTask.ID)
		}
		t.Logf("Received task: %+v", receivedTask)
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}
	defer sub.Unsubscribe()

	// Fetch and process one message
	msgs, err := sub.Fetch(1, nats.MaxWait(5*time.Second))
	if err != nil {
		t.Fatalf("Failed to fetch messages: %v", err)
	}

	if len(msgs) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(msgs))
	}

	// Parse and verify message
	var receivedTask natsClient.TaskMessage
	err = json.Unmarshal(msgs[0].Data, &receivedTask)
	if err != nil {
		t.Fatalf("Failed to unmarshal task: %v", err)
	}

	if receivedTask.ID != task.ID {
		t.Errorf("Expected task ID %s, got %s", task.ID, receivedTask.ID)
	}

	// Acknowledge message
	msgs[0].Ack()

	t.Log("Message consumed and acknowledged successfully")
}

// TestAPIWithNATS tests the HTTP API with NATS integration
func TestAPIWithNATS(t *testing.T) {
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		t.Skip("Skipping integration test: NATS_URL not set")
	}

	// Initialize task queue and NATS
	taskQueue = NewTaskQueue()
	var err error
	nats, err = natsClient.NewClient(natsURL)
	if err != nil {
		t.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nats.Close()

	// Test POST /api/tasks - should publish to NATS
	reqBody := `{
		"issue_id": "123",
		"repository": "/test/repo",
		"task_file_path": "/test/repo/docs/task/123-test.md"
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/tasks", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	CreateTaskHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}

	var task Task
	json.NewDecoder(resp.Body).Decode(&task)

	if task.IssueID != "123" {
		t.Errorf("Expected issue_id 123, got %s", task.IssueID)
	}

	t.Logf("Task created and published: %+v", task)

	// Verify message is in stream
	streamInfo, err := nats.GetStreamInfo()
	if err != nil {
		t.Fatalf("Failed to get stream info: %v", err)
	}

	if streamInfo.State.Msgs == 0 {
		t.Error("Expected at least 1 message in stream")
	}

	t.Logf("Stream has %d messages", streamInfo.State.Msgs)
}

// TestStatusUpdatePublish tests publishing status updates
func TestStatusUpdatePublish(t *testing.T) {
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		t.Skip("Skipping integration test: NATS_URL not set")
	}

	client, err := natsClient.NewClient(natsURL)
	if err != nil {
		t.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer client.Close()

	// Publish status update
	update := &natsClient.StatusUpdateMessage{
		TaskID:    "test-task-123",
		Status:    "in_progress",
		UpdatedAt: time.Now(),
	}

	err = client.PublishTaskStatusUpdate(update)
	if err != nil {
		t.Fatalf("Failed to publish status update: %v", err)
	}

	t.Log("Status update published successfully")
}

// TestDeletePublish tests publishing delete messages
func TestDeletePublish(t *testing.T) {
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		t.Skip("Skipping integration test: NATS_URL not set")
	}

	client, err := natsClient.NewClient(natsURL)
	if err != nil {
		t.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer client.Close()

	// Publish delete message
	err = client.PublishTaskDelete("test-task-123")
	if err != nil {
		t.Fatalf("Failed to publish delete message: %v", err)
	}

	t.Log("Delete message published successfully")
}
