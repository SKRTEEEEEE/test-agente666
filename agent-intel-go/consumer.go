package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/nats-io/nats.go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// EventConsumer handles NATS event consumption
type EventConsumer struct {
	nc          *nats.Conn
	js          nats.JetStreamContext
	mongoDB     *mongo.Database
	mongoClient *mongo.Client
}

// NewEventConsumer creates a new event consumer
func NewEventConsumer(natsURL string, mongoDB *mongo.Database, mongoClient *mongo.Client) (*EventConsumer, error) {
	// Connect to NATS
	nc, err := nats.Connect(natsURL,
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			log.Printf("NATS disconnected: %v", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Printf("NATS reconnected to %s", nc.ConnectedUrl())
		}),
	)
	if err != nil {
		return nil, err
	}

	// Get JetStream context
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, err
	}

	consumer := &EventConsumer{
		nc:          nc,
		js:          js,
		mongoDB:     mongoDB,
		mongoClient: mongoClient,
	}

	// Initialize stream
	if err := consumer.initializeStream(); err != nil {
		log.Printf("Warning: Failed to initialize stream: %v", err)
	}

	return consumer, nil
}

// initializeStream creates the AGENT stream if it doesn't exist
func (ec *EventConsumer) initializeStream() error {
	streamName := "AGENT"

	// Check if stream already exists
	_, err := ec.js.StreamInfo(streamName)
	if err == nil {
		log.Printf("Stream %s already exists", streamName)
		return nil
	}

	// Create stream
	_, err = ec.js.AddStream(&nats.StreamConfig{
		Name:     streamName,
		Subjects: []string{"agent.*"},
		Storage:  nats.FileStorage,
		MaxAge:   7 * 24 * time.Hour, // Keep messages for 7 days
		Replicas: 1,
	})
	if err != nil {
		return err
	}

	log.Printf("Created stream: %s", streamName)
	return nil
}

// Start begins consuming events
func (ec *EventConsumer) Start() error {
	// Subscribe to task.new events
	_, err := ec.js.Subscribe("agent.task.new", ec.handleTaskNew,
		nats.Durable("agent-intel-task-new"),
		nats.ManualAck(),
		nats.MaxDeliver(3),
		nats.AckWait(30*time.Second),
		nats.BindStream("AGENT"),
	)
	if err != nil {
		log.Printf("Failed to subscribe to agent.task.new: %v", err)
		return err
	}
	log.Println("Subscribed to agent.task.new")

	// Subscribe to pipeline.completed events
	_, err = ec.js.Subscribe("agent.pipeline.completed", ec.handlePipelineCompleted,
		nats.Durable("agent-intel-pipeline-completed"),
		nats.ManualAck(),
		nats.MaxDeliver(3),
		nats.AckWait(30*time.Second),
		nats.BindStream("AGENT"),
	)
	if err != nil {
		log.Printf("Failed to subscribe to agent.pipeline.completed: %v", err)
		return err
	}
	log.Println("Subscribed to agent.pipeline.completed")

	return nil
}

// handleTaskNew processes agent.task.new events
func (ec *EventConsumer) handleTaskNew(msg *nats.Msg) {
	var event TaskNewEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		log.Printf("Error unmarshaling task.new event: %v", err)
		msg.Nak()
		return
	}

	log.Printf("Received task.new event: %s", event.TaskID)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pendingCol := ec.mongoDB.Collection("pending_tasks")

	// Check for idempotency - if task already exists, skip
	count, err := pendingCol.CountDocuments(ctx, bson.M{"task_id": event.TaskID})
	if err != nil {
		log.Printf("Error checking task existence: %v", err)
		msg.Nak()
		return
	}
	if count > 0 {
		log.Printf("Task %s already exists, skipping (idempotent)", event.TaskID)
		msg.Ack()
		return
	}

	// Get pending task count for this repository
	repoPendingCount, err := pendingCol.CountDocuments(ctx, bson.M{
		"repository": event.Repository,
		"status":     bson.M{"$in": []string{StatusPending, StatusAssigned, StatusProcessing}},
	})
	if err != nil {
		log.Printf("Warning: Failed to count repo pending tasks: %v", err)
		repoPendingCount = 0
	}

	// Get last success time for this repository
	historyCol := ec.mongoDB.Collection("task_history")
	opts := options.FindOne().SetSort(bson.M{"completed_at": -1})
	var lastTask TaskMetrics
	err = historyCol.FindOne(ctx, bson.M{
		"repository": event.Repository,
		"status":     StatusCompleted,
	}, opts).Decode(&lastTask)

	lastSuccessAt := time.Time{}
	avgRuntime := int64(0)
	if err == nil {
		lastSuccessAt = lastTask.AssignedAt
		avgRuntime = lastTask.PipelineRuntimeMs
	}

	// Create task metrics
	task := &TaskMetrics{
		TaskID:            event.TaskID,
		IssueID:           event.IssueID,
		Repository:        event.Repository,
		TaskFilePath:      event.TaskFilePath,
		CreatedAt:         event.CreatedAt,
		LastSuccessAt:     lastSuccessAt,
		AvgRuntimeMs:      avgRuntime,
		PendingTasksCount: int(repoPendingCount),
		SizeBytes:         event.SizeBytes,
		Status:            StatusPending,
	}

	// Insert into pending_tasks collection
	_, err = pendingCol.InsertOne(ctx, task)
	if err != nil {
		log.Printf("Error inserting task: %v", err)

		// Check if it's a duplicate key error (race condition)
		if mongo.IsDuplicateKeyError(err) {
			log.Printf("Duplicate task detected (race condition), acknowledging")
			msg.Ack()
			return
		}

		msg.Nak()
		return
	}

	log.Printf("Task %s stored successfully", event.TaskID)
	msg.Ack()
}

// handlePipelineCompleted processes agent.pipeline.completed events
func (ec *EventConsumer) handlePipelineCompleted(msg *nats.Msg) {
	var event PipelineCompletedEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		log.Printf("Error unmarshaling pipeline.completed event: %v", err)
		msg.Nak()
		return
	}

	log.Printf("Received pipeline.completed event: %s (status: %s)", event.TaskID, event.Status)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pendingCol := ec.mongoDB.Collection("pending_tasks")
	historyCol := ec.mongoDB.Collection("task_history")

	// Find the task in pending collection
	var task TaskMetrics
	err := pendingCol.FindOne(ctx, bson.M{"task_id": event.TaskID}).Decode(&task)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Task might already be in history (idempotency check)
			count, _ := historyCol.CountDocuments(ctx, bson.M{"task_id": event.TaskID})
			if count > 0 {
				log.Printf("Task %s already in history, skipping (idempotent)", event.TaskID)
				msg.Ack()
				return
			}
			log.Printf("Task %s not found in pending collection", event.TaskID)
		} else {
			log.Printf("Error finding task: %v", err)
		}
		msg.Nak()
		return
	}

	// Update task with completion data
	task.PipelineRuntimeMs = event.PipelineRuntimeMs
	if event.Status == "success" {
		task.Status = StatusCompleted
	} else {
		task.Status = StatusFailed
		task.ErrorMessage = event.ErrorMessage
	}

	// Move task to history
	_, err = historyCol.InsertOne(ctx, task)
	if err != nil {
		// Check for duplicate
		if mongo.IsDuplicateKeyError(err) {
			log.Printf("Task %s already in history (race condition)", event.TaskID)
			// Still remove from pending
		} else {
			log.Printf("Error inserting into history: %v", err)
			msg.Nak()
			return
		}
	}

	// Remove from pending
	_, err = pendingCol.DeleteOne(ctx, bson.M{"task_id": event.TaskID})
	if err != nil {
		log.Printf("Warning: Failed to delete task from pending: %v", err)
	}

	// Update metrics for this repository
	ec.updateRepositoryMetrics(ctx, event.Repository)

	log.Printf("Task %s moved to history with status %s", event.TaskID, task.Status)
	msg.Ack()
}

// updateRepositoryMetrics recalculates metrics for all pending tasks in a repository
func (ec *EventConsumer) updateRepositoryMetrics(ctx context.Context, repository string) {
	pendingCol := ec.mongoDB.Collection("pending_tasks")
	historyCol := ec.mongoDB.Collection("task_history")

	// Get pending count for this repo
	pendingCount, err := pendingCol.CountDocuments(ctx, bson.M{
		"repository": repository,
		"status":     bson.M{"$in": []string{StatusPending, StatusAssigned, StatusProcessing}},
	})
	if err != nil {
		log.Printf("Error counting pending tasks for %s: %v", repository, err)
		return
	}

	// Get last success and average runtime
	opts := options.FindOne().SetSort(bson.M{"completed_at": -1})
	var lastTask TaskMetrics
	err = historyCol.FindOne(ctx, bson.M{
		"repository": repository,
		"status":     StatusCompleted,
	}, opts).Decode(&lastTask)

	lastSuccessAt := time.Time{}
	if err == nil {
		lastSuccessAt = lastTask.AssignedAt
	}

	// Calculate average runtime from last 10 completed tasks
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"repository":          repository,
			"status":              StatusCompleted,
			"pipeline_runtime_ms": bson.M{"$gt": 0},
		}}},
		{{Key: "$sort", Value: bson.M{"completed_at": -1}}},
		{{Key: "$limit", Value: 10}},
		{{Key: "$group", Value: bson.M{
			"_id":         nil,
			"avg_runtime": bson.M{"$avg": "$pipeline_runtime_ms"},
		}}},
	}

	cursor, err := historyCol.Aggregate(ctx, pipeline)
	if err != nil {
		log.Printf("Error calculating avg runtime for %s: %v", repository, err)
		return
	}
	defer cursor.Close(ctx)

	var avgResult struct {
		AvgRuntime float64 `bson:"avg_runtime"`
	}
	avgRuntime := int64(0)
	if cursor.Next(ctx) {
		cursor.Decode(&avgResult)
		avgRuntime = int64(avgResult.AvgRuntime)
	}

	// Update all pending tasks for this repository
	_, err = pendingCol.UpdateMany(
		ctx,
		bson.M{"repository": repository, "status": bson.M{"$ne": StatusCancelled}},
		bson.M{"$set": bson.M{
			"last_success_at":     lastSuccessAt,
			"avg_runtime_ms":      avgRuntime,
			"pending_tasks_count": int(pendingCount),
		}},
	)
	if err != nil {
		log.Printf("Error updating metrics for %s: %v", repository, err)
	}
}

// Close closes the NATS connection
func (ec *EventConsumer) Close() {
	if ec.nc != nil {
		ec.nc.Close()
	}
}
