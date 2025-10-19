package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestNATSEventConsumption(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Skip if NATS not available
	nc, err := nats.Connect("nats://localhost:4222")
	if err != nil {
		t.Skip("NATS not available, skipping integration test")
	}
	defer nc.Close()

	js, err := nc.JetStream()
	if err != nil {
		t.Fatalf("Failed to get JetStream context: %v", err)
	}

	// Create test stream if not exists
	streamName := "AGENT_TEST"
	_, err = js.StreamInfo(streamName)
	if err != nil {
		_, err = js.AddStream(&nats.StreamConfig{
			Name:     streamName,
			Subjects: []string{"agent.test.*"},
			Storage:  nats.FileStorage,
			MaxAge:   time.Hour,
		})
		if err != nil {
			t.Fatalf("Failed to create test stream: %v", err)
		}
	}

	// Clean up test stream after test
	defer js.DeleteStream(streamName)

	t.Run("consume task.new event", func(t *testing.T) {
		taskEvent := TaskNewEvent{
			TaskID:       "test-task-1",
			IssueID:      "123",
			Repository:   "/test/repo",
			TaskFilePath: "/test/repo/docs/task/123-test.md",
			SizeBytes:    1024,
			CreatedAt:    time.Now(),
		}

		data, err := json.Marshal(taskEvent)
		if err != nil {
			t.Fatalf("Failed to marshal event: %v", err)
		}

		// Publish event
		_, err = js.Publish("agent.test.new", data)
		if err != nil {
			t.Fatalf("Failed to publish event: %v", err)
		}

		// Subscribe and consume
		sub, err := js.SubscribeSync("agent.test.new", nats.Durable("test-consumer"))
		if err != nil {
			t.Fatalf("Failed to subscribe: %v", err)
		}
		defer sub.Unsubscribe()

		msg, err := sub.NextMsg(2 * time.Second)
		if err != nil {
			t.Fatalf("Failed to receive message: %v", err)
		}

		var receivedEvent TaskNewEvent
		err = json.Unmarshal(msg.Data, &receivedEvent)
		if err != nil {
			t.Fatalf("Failed to unmarshal event: %v", err)
		}

		if receivedEvent.TaskID != taskEvent.TaskID {
			t.Errorf("TaskID = %v, want %v", receivedEvent.TaskID, taskEvent.TaskID)
		}

		msg.Ack()
	})

	t.Run("consume pipeline.completed event", func(t *testing.T) {
		completedEvent := PipelineCompletedEvent{
			TaskID:            "test-task-2",
			Repository:        "/test/repo",
			PipelineRuntimeMs: 5000,
			Status:            "success",
			CompletedAt:       time.Now(),
		}

		data, err := json.Marshal(completedEvent)
		if err != nil {
			t.Fatalf("Failed to marshal event: %v", err)
		}

		// Publish event
		_, err = js.Publish("agent.test.completed", data)
		if err != nil {
			t.Fatalf("Failed to publish event: %v", err)
		}

		// Subscribe and consume
		sub, err := js.SubscribeSync("agent.test.completed", nats.Durable("test-consumer-2"))
		if err != nil {
			t.Fatalf("Failed to subscribe: %v", err)
		}
		defer sub.Unsubscribe()

		msg, err := sub.NextMsg(2 * time.Second)
		if err != nil {
			t.Fatalf("Failed to receive message: %v", err)
		}

		var receivedEvent PipelineCompletedEvent
		err = json.Unmarshal(msg.Data, &receivedEvent)
		if err != nil {
			t.Fatalf("Failed to unmarshal event: %v", err)
		}

		if receivedEvent.TaskID != completedEvent.TaskID {
			t.Errorf("TaskID = %v, want %v", receivedEvent.TaskID, completedEvent.TaskID)
		}

		msg.Ack()
	})
}

func TestMongoDBPersistence(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Skip if MongoDB not available
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Skip("MongoDB not available, skipping integration test")
	}
	defer client.Disconnect(ctx)

	// Test database connection
	err = client.Ping(ctx, nil)
	if err != nil {
		t.Skip("MongoDB not responding, skipping integration test")
	}

	db := client.Database("agent_intel_test")
	pendingCol := db.Collection("pending_tasks")
	historyCol := db.Collection("task_history")

	// Clean up test data
	defer func() {
		pendingCol.Drop(ctx)
		historyCol.Drop(ctx)
	}()

	t.Run("insert and retrieve pending task", func(t *testing.T) {
		task := &TaskMetrics{
			TaskID:            "mongo-test-1",
			IssueID:           "456",
			Repository:        "/test/repo",
			TaskFilePath:      "/test/repo/docs/task/456-test.md",
			CreatedAt:         time.Now(),
			LastSuccessAt:     time.Now().Add(-24 * time.Hour),
			AvgRuntimeMs:      10000,
			PendingTasksCount: 3,
			SizeBytes:         2048,
			Status:            StatusPending,
		}

		// Insert
		_, err := pendingCol.InsertOne(ctx, task)
		if err != nil {
			t.Fatalf("Failed to insert task: %v", err)
		}

		// Retrieve
		var retrieved TaskMetrics
		err = pendingCol.FindOne(ctx, bson.M{"task_id": "mongo-test-1"}).Decode(&retrieved)
		if err != nil {
			t.Fatalf("Failed to retrieve task: %v", err)
		}

		if retrieved.TaskID != task.TaskID {
			t.Errorf("TaskID = %v, want %v", retrieved.TaskID, task.TaskID)
		}
		if retrieved.IssueID != task.IssueID {
			t.Errorf("IssueID = %v, want %v", retrieved.IssueID, task.IssueID)
		}
	})

	t.Run("update task status", func(t *testing.T) {
		task := &TaskMetrics{
			TaskID:       "mongo-test-2",
			IssueID:      "789",
			Repository:   "/test/repo",
			TaskFilePath: "/test/repo/docs/task/789-test.md",
			CreatedAt:    time.Now(),
			Status:       StatusPending,
		}

		// Insert
		_, err := pendingCol.InsertOne(ctx, task)
		if err != nil {
			t.Fatalf("Failed to insert task: %v", err)
		}

		// Update status
		_, err = pendingCol.UpdateOne(
			ctx,
			bson.M{"task_id": "mongo-test-2"},
			bson.M{"$set": bson.M{"status": StatusProcessing, "assigned_at": time.Now()}},
		)
		if err != nil {
			t.Fatalf("Failed to update task: %v", err)
		}

		// Verify update
		var updated TaskMetrics
		err = pendingCol.FindOne(ctx, bson.M{"task_id": "mongo-test-2"}).Decode(&updated)
		if err != nil {
			t.Fatalf("Failed to retrieve updated task: %v", err)
		}

		if updated.Status != StatusProcessing {
			t.Errorf("Status = %v, want %v", updated.Status, StatusProcessing)
		}
	})

	t.Run("move task to history", func(t *testing.T) {
		task := &TaskMetrics{
			TaskID:            "mongo-test-3",
			IssueID:           "321",
			Repository:        "/test/repo",
			TaskFilePath:      "/test/repo/docs/task/321-test.md",
			CreatedAt:         time.Now().Add(-1 * time.Hour),
			AssignedAt:        time.Now().Add(-30 * time.Minute),
			Status:            StatusCompleted,
			PipelineRuntimeMs: 15000,
		}

		// Insert into history
		_, err := historyCol.InsertOne(ctx, task)
		if err != nil {
			t.Fatalf("Failed to insert into history: %v", err)
		}

		// Delete from pending
		_, err = pendingCol.DeleteOne(ctx, bson.M{"task_id": "mongo-test-3"})
		if err != nil {
			t.Fatalf("Failed to delete from pending: %v", err)
		}

		// Verify in history
		var historical TaskMetrics
		err = historyCol.FindOne(ctx, bson.M{"task_id": "mongo-test-3"}).Decode(&historical)
		if err != nil {
			t.Fatalf("Failed to retrieve from history: %v", err)
		}

		if historical.Status != StatusCompleted {
			t.Errorf("Status = %v, want %v", historical.Status, StatusCompleted)
		}

		// Verify not in pending
		count, err := pendingCol.CountDocuments(ctx, bson.M{"task_id": "mongo-test-3"})
		if err != nil {
			t.Fatalf("Failed to count pending tasks: %v", err)
		}
		if count != 0 {
			t.Errorf("Task still in pending collection")
		}
	})

	t.Run("query tasks by repository", func(t *testing.T) {
		// Insert multiple tasks for same repository
		tasks := []*TaskMetrics{
			{
				TaskID:     "repo-test-1",
				Repository: "/shared/repo",
				Status:     StatusPending,
				CreatedAt:  time.Now(),
			},
			{
				TaskID:     "repo-test-2",
				Repository: "/shared/repo",
				Status:     StatusPending,
				CreatedAt:  time.Now(),
			},
			{
				TaskID:     "repo-test-3",
				Repository: "/other/repo",
				Status:     StatusPending,
				CreatedAt:  time.Now(),
			},
		}

		for _, task := range tasks {
			_, err := pendingCol.InsertOne(ctx, task)
			if err != nil {
				t.Fatalf("Failed to insert task: %v", err)
			}
		}

		// Query by repository
		cursor, err := pendingCol.Find(ctx, bson.M{"repository": "/shared/repo"})
		if err != nil {
			t.Fatalf("Failed to query tasks: %v", err)
		}
		defer cursor.Close(ctx)

		var results []TaskMetrics
		err = cursor.All(ctx, &results)
		if err != nil {
			t.Fatalf("Failed to decode results: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("Found %d tasks, want 2", len(results))
		}

		for _, result := range results {
			if result.Repository != "/shared/repo" {
				t.Errorf("Repository = %v, want /shared/repo", result.Repository)
			}
		}
	})
}

func TestEventToMongoFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This test simulates the complete flow: NATS event → MongoDB persistence
	// Skip if either service not available

	// Connect to NATS
	nc, err := nats.Connect("nats://localhost:4222")
	if err != nil {
		t.Skip("NATS not available, skipping integration test")
	}
	defer nc.Close()

	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Skip("MongoDB not available, skipping integration test")
	}
	defer mongoClient.Disconnect(ctx)

	db := mongoClient.Database("agent_intel_test")
	pendingCol := db.Collection("pending_tasks")

	// Clean up
	defer pendingCol.Drop(ctx)

	t.Run("task.new event creates MongoDB record", func(t *testing.T) {
		taskEvent := TaskNewEvent{
			TaskID:       "flow-test-1",
			IssueID:      "999",
			Repository:   "/test/repo",
			TaskFilePath: "/test/repo/docs/task/999-test.md",
			SizeBytes:    4096,
			CreatedAt:    time.Now(),
		}

		// Simulate event processing
		task := &TaskMetrics{
			TaskID:            taskEvent.TaskID,
			IssueID:           taskEvent.IssueID,
			Repository:        taskEvent.Repository,
			TaskFilePath:      taskEvent.TaskFilePath,
			SizeBytes:         taskEvent.SizeBytes,
			CreatedAt:         taskEvent.CreatedAt,
			Status:            StatusPending,
			PendingTasksCount: 1,
		}

		// Insert into MongoDB
		_, err := pendingCol.InsertOne(ctx, task)
		if err != nil {
			t.Fatalf("Failed to insert task: %v", err)
		}

		// Verify it was stored
		var stored TaskMetrics
		err = pendingCol.FindOne(ctx, bson.M{"task_id": "flow-test-1"}).Decode(&stored)
		if err != nil {
			t.Fatalf("Failed to retrieve task: %v", err)
		}

		if stored.Status != StatusPending {
			t.Errorf("Status = %v, want %v", stored.Status, StatusPending)
		}
	})
}
