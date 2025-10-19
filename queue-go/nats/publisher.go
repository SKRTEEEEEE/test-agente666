package nats

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

// TaskMessage represents a task message for NATS
type TaskMessage struct {
	ID           string    `json:"id"`
	IssueID      string    `json:"issue_id"`
	Repository   string    `json:"repository"`
	TaskFilePath string    `json:"task_file_path"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	ErrorMessage string    `json:"error_message,omitempty"`
}

// StatusUpdateMessage represents a status update message
type StatusUpdateMessage struct {
	TaskID       string    `json:"task_id"`
	Status       string    `json:"status"`
	ErrorMessage string    `json:"error_message,omitempty"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// DeleteMessage represents a task deletion message
type DeleteMessage struct {
	TaskID    string    `json:"task_id"`
	DeletedAt time.Time `json:"deleted_at"`
}

// PublishNewTask publishes a new task to the stream
func (c *Client) PublishNewTask(task *TaskMessage) error {
	data, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	// Publish to JetStream with message deduplication
	_, err = c.js.Publish(SubjectTaskNew, data, nats.MsgId(task.ID))
	if err != nil {
		return fmt.Errorf("failed to publish task: %w", err)
	}

	return nil
}

// PublishTaskStatusUpdate publishes a task status update
func (c *Client) PublishTaskStatusUpdate(update *StatusUpdateMessage) error {
	data, err := json.Marshal(update)
	if err != nil {
		return fmt.Errorf("failed to marshal status update: %w", err)
	}

	subject := fmt.Sprintf("%s.%s", SubjectTaskStatus, update.Status)
	_, err = c.js.Publish(subject, data)
	if err != nil {
		return fmt.Errorf("failed to publish status update: %w", err)
	}

	return nil
}

// PublishTaskDelete publishes a task deletion message
func (c *Client) PublishTaskDelete(taskID string) error {
	msg := &DeleteMessage{
		TaskID:    taskID,
		DeletedAt: time.Now(),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal delete message: %w", err)
	}

	_, err = c.js.Publish(SubjectTaskDelete, data)
	if err != nil {
		return fmt.Errorf("failed to publish delete message: %w", err)
	}

	return nil
}

// GetStreamInfo returns information about the TASKS stream
func (c *Client) GetStreamInfo() (*nats.StreamInfo, error) {
	info, err := c.js.StreamInfo(StreamName)
	if err != nil {
		return nil, fmt.Errorf("failed to get stream info: %w", err)
	}
	return info, nil
}

// GetConsumerInfo returns information about the task consumer
func (c *Client) GetConsumerInfo() (*nats.ConsumerInfo, error) {
	info, err := c.js.ConsumerInfo(StreamName, ConsumerName)
	if err != nil {
		return nil, fmt.Errorf("failed to get consumer info: %w", err)
	}
	return info, nil
}
