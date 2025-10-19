package nats

import (
	"testing"
	"time"
)

func TestTaskMessage(t *testing.T) {
	msg := &TaskMessage{
		ID:           "test-id",
		IssueID:      "11",
		Repository:   "/test/repo",
		TaskFilePath: "/test/repo/docs/task/11-test.md",
		Status:       "pending",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if msg.ID != "test-id" {
		t.Errorf("Expected ID to be test-id, got %s", msg.ID)
	}

	if msg.IssueID != "11" {
		t.Errorf("Expected IssueID to be 11, got %s", msg.IssueID)
	}

	if msg.Status != "pending" {
		t.Errorf("Expected Status to be pending, got %s", msg.Status)
	}
}

func TestStatusUpdateMessage(t *testing.T) {
	msg := &StatusUpdateMessage{
		TaskID:    "test-id",
		Status:    "in_progress",
		UpdatedAt: time.Now(),
	}

	if msg.TaskID != "test-id" {
		t.Errorf("Expected TaskID to be test-id, got %s", msg.TaskID)
	}

	if msg.Status != "in_progress" {
		t.Errorf("Expected Status to be in_progress, got %s", msg.Status)
	}
}

func TestDeleteMessage(t *testing.T) {
	msg := &DeleteMessage{
		TaskID:    "test-id",
		DeletedAt: time.Now(),
	}

	if msg.TaskID != "test-id" {
		t.Errorf("Expected TaskID to be test-id, got %s", msg.TaskID)
	}

	if msg.DeletedAt.IsZero() {
		t.Error("Expected DeletedAt to be set")
	}
}

func TestConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"StreamName", StreamName, "TASKS"},
		{"SubjectTaskNew", SubjectTaskNew, "tasks.new"},
		{"SubjectTaskUpdate", SubjectTaskUpdate, "tasks.update"},
		{"SubjectTaskDelete", SubjectTaskDelete, "tasks.delete"},
		{"SubjectTaskStatus", SubjectTaskStatus, "tasks.status"},
		{"ConsumerName", ConsumerName, "task-workers"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Expected %s to be %s, got %s", tt.name, tt.expected, tt.constant)
			}
		})
	}
}
