package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

// QdrantClient handles interactions with Qdrant vector database
type QdrantClient struct {
	baseURL        string
	collectionName string
	httpClient     *http.Client
}

// QdrantPoint represents a point in Qdrant
type QdrantPoint struct {
	ID      string                 `json:"id"`
	Vector  []float32              `json:"vector"`
	Payload map[string]interface{} `json:"payload"`
}

// QdrantSearchResponse represents Qdrant search response
type QdrantSearchResponse struct {
	Result []struct {
		ID      string                 `json:"id"`
		Score   float32                `json:"score"`
		Payload map[string]interface{} `json:"payload"`
	} `json:"result"`
}

// NewQdrantClient creates a new Qdrant client
func NewQdrantClient() *QdrantClient {
	baseURL := os.Getenv("QDRANT_URL")
	if baseURL == "" {
		baseURL = "http://localhost:6333"
	}

	return &QdrantClient{
		baseURL:        baseURL,
		collectionName: "tasks",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// InitializeCollection creates the collection if it doesn't exist
func (c *QdrantClient) InitializeCollection() error {
	// Create collection with vector size 384 (all-MiniLM-L6-v2 embedding size)
	createCollectionPayload := map[string]interface{}{
		"vectors": map[string]interface{}{
			"size":     384,
			"distance": "Cosine",
		},
	}

	payload, err := json.Marshal(createCollectionPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s/collections/%s", c.baseURL, c.collectionName)
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// If Qdrant is not available, log warning but don't fail
		log.Printf("Warning: Qdrant not available at %s: %v", c.baseURL, err)
		return nil
	}
	defer resp.Body.Close()

	// 200 (created) or 409 (already exists) are both OK
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusConflict {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create collection: status %d, body: %s", resp.StatusCode, string(body))
	}

	log.Printf("Qdrant collection '%s' initialized successfully", c.collectionName)
	return nil
}

// SaveTask saves a task to Qdrant
func (c *QdrantClient) SaveTask(task *Task) error {
	// Generate a simple embedding based on task properties
	// In production, you would use a real embedding model
	vector := c.generateSimpleEmbedding(task)

	point := QdrantPoint{
		ID:     task.ID,
		Vector: vector,
		Payload: map[string]interface{}{
			"issue_id":       task.IssueID,
			"repository":     task.Repository,
			"task_file_path": task.TaskFilePath,
			"status":         task.Status,
			"created_at":     task.CreatedAt.Format(time.RFC3339),
			"updated_at":     task.UpdatedAt.Format(time.RFC3339),
			"error_message":  task.ErrorMessage,
		},
	}

	payload := map[string]interface{}{
		"points": []QdrantPoint{point},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal point: %w", err)
	}

	url := fmt.Sprintf("%s/collections/%s/points?wait=true", c.baseURL, c.collectionName)
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// If Qdrant is not available, log warning but continue
		log.Printf("Warning: Failed to save to Qdrant: %v", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to save task: status %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetTask retrieves a task from Qdrant by ID
func (c *QdrantClient) GetTask(id string) (*Task, error) {
	url := fmt.Sprintf("%s/collections/%s/points/%s", c.baseURL, c.collectionName, id)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.New("task not found")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get task: status %d, body: %s", resp.StatusCode, string(body))
	}

	var response struct {
		Result struct {
			ID      string                 `json:"id"`
			Payload map[string]interface{} `json:"payload"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return c.payloadToTask(id, response.Result.Payload)
}

// ListAllTasks retrieves all tasks from Qdrant
func (c *QdrantClient) ListAllTasks() ([]*Task, error) {
	url := fmt.Sprintf("%s/collections/%s/points/scroll", c.baseURL, c.collectionName)

	payload := map[string]interface{}{
		"limit":        1000,
		"with_payload": true,
		"with_vector":  false,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list tasks: status %d, body: %s", resp.StatusCode, string(body))
	}

	var response struct {
		Result struct {
			Points []struct {
				ID      string                 `json:"id"`
				Payload map[string]interface{} `json:"payload"`
			} `json:"points"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	tasks := make([]*Task, 0, len(response.Result.Points))
	for _, point := range response.Result.Points {
		task, err := c.payloadToTask(point.ID, point.Payload)
		if err != nil {
			log.Printf("Warning: Failed to convert point %s to task: %v", point.ID, err)
			continue
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// DeleteTask deletes a task from Qdrant
func (c *QdrantClient) DeleteTask(id string) error {
	payload := map[string]interface{}{
		"points": []string{id},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s/collections/%s/points/delete?wait=true", c.baseURL, c.collectionName)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete task: status %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// UpdateTask updates a task in Qdrant (same as SaveTask)
func (c *QdrantClient) UpdateTask(task *Task) error {
	return c.SaveTask(task)
}

// Helper function to convert payload to Task
func (c *QdrantClient) payloadToTask(id string, payload map[string]interface{}) (*Task, error) {
	task := &Task{ID: id}

	if issueID, ok := payload["issue_id"].(string); ok {
		task.IssueID = issueID
	}

	if repository, ok := payload["repository"].(string); ok {
		task.Repository = repository
	}

	if taskFilePath, ok := payload["task_file_path"].(string); ok {
		task.TaskFilePath = taskFilePath
	}

	if status, ok := payload["status"].(string); ok {
		task.Status = status
	}

	if createdAt, ok := payload["created_at"].(string); ok {
		t, err := time.Parse(time.RFC3339, createdAt)
		if err == nil {
			task.CreatedAt = t
		}
	}

	if updatedAt, ok := payload["updated_at"].(string); ok {
		t, err := time.Parse(time.RFC3339, updatedAt)
		if err == nil {
			task.UpdatedAt = t
		}
	}

	if errorMessage, ok := payload["error_message"].(string); ok {
		task.ErrorMessage = errorMessage
	}

	return task, nil
}

// Generate a simple embedding based on task properties
// In production, use a proper embedding model
func (c *QdrantClient) generateSimpleEmbedding(task *Task) []float32 {
	// Create a 384-dimensional vector (standard for many embedding models)
	vector := make([]float32, 384)

	// Simple hash-based embedding for demonstration
	// In production, use a real embedding model like sentence-transformers
	text := fmt.Sprintf("%s %s %s %s", task.IssueID, task.Repository, task.TaskFilePath, task.Status)

	for i, char := range text {
		idx := i % 384
		vector[idx] += float32(char) / 1000.0
	}

	// Normalize the vector
	var magnitude float32
	for _, val := range vector {
		magnitude += val * val
	}
	magnitude = float32(1.0 / (1.0 + magnitude))

	for i := range vector {
		vector[i] *= magnitude
	}

	return vector
}

// IsAvailable checks if Qdrant is available
func (c *QdrantClient) IsAvailable() bool {
	resp, err := c.httpClient.Get(c.baseURL + "/collections")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
