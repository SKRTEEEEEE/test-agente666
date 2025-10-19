package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// AgentIntelService holds the service dependencies
type AgentIntelService struct {
	mongoDB     *mongo.Database
	mongoClient *mongo.Client
	natsURL     string
}

// healthHandler returns the health status of the service
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// TODO: Add actual health checks for MongoDB and NATS
	health := HealthStatus{
		Status:  "healthy",
		MongoDB: "connected",
		NATS:    "connected",
	}

	json.NewEncoder(w).Encode(health)
}

// getNextTaskHandler returns the next task with highest priority
func (s *AgentIntelService) getNextTaskHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get optional repo_id filter
	repoID := r.URL.Query().Get("repo_id")

	// Build query filter
	filter := bson.M{"status": StatusPending}
	if repoID != "" {
		filter["repository"] = repoID
	}

	// Fetch all pending tasks
	pendingCol := s.mongoDB.Collection("pending_tasks")
	cursor, err := pendingCol.Find(ctx, filter)
	if err != nil {
		http.Error(w, "Failed to fetch tasks", http.StatusInternalServerError)
		log.Printf("Error fetching tasks: %v", err)
		return
	}
	defer cursor.Close(ctx)

	var tasks []TaskMetrics
	if err := cursor.All(ctx, &tasks); err != nil {
		http.Error(w, "Failed to decode tasks", http.StatusInternalServerError)
		log.Printf("Error decoding tasks: %v", err)
		return
	}

	if len(tasks) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "no tasks available"})
		return
	}

	// Calculate scores and find highest priority task
	var bestTask *TaskMetrics
	var bestScore float64

	for i := range tasks {
		score := CalculateScore(&tasks[i])
		if bestTask == nil || score > bestScore {
			bestTask = &tasks[i]
			bestScore = score
		}
	}

	// Mark task as assigned
	now := time.Now()
	bestTask.Status = StatusAssigned
	bestTask.AssignedAt = now

	_, err = pendingCol.UpdateOne(
		ctx,
		bson.M{"task_id": bestTask.TaskID},
		bson.M{
			"$set": bson.M{
				"status":      StatusAssigned,
				"assigned_at": now,
			},
		},
	)
	if err != nil {
		log.Printf("Warning: Failed to update task status: %v", err)
	}

	// Return task with score
	response := NextTaskResponse{
		Task:  bestTask,
		Score: bestScore,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// cancelTaskHandler cancels a task
func (s *AgentIntelService) cancelTaskHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var req CancelTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.TaskID == "" {
		http.Error(w, "task_id is required", http.StatusBadRequest)
		return
	}

	pendingCol := s.mongoDB.Collection("pending_tasks")

	// Update task status to cancelled
	result, err := pendingCol.UpdateOne(
		ctx,
		bson.M{"task_id": req.TaskID},
		bson.M{
			"$set": bson.M{
				"status":        StatusCancelled,
				"cancel_reason": req.Reason,
			},
		},
	)
	if err != nil {
		http.Error(w, "Failed to cancel task", http.StatusInternalServerError)
		log.Printf("Error cancelling task: %v", err)
		return
	}

	if result.MatchedCount == 0 {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "task cancelled successfully",
		"task_id": req.TaskID,
	})
}

// metricsHandler returns system metrics
func (s *AgentIntelService) metricsHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pendingCol := s.mongoDB.Collection("pending_tasks")
	historyCol := s.mongoDB.Collection("task_history")

	// Count pending tasks
	pendingCount, err := pendingCol.CountDocuments(ctx, bson.M{"status": StatusPending})
	if err != nil {
		log.Printf("Error counting pending tasks: %v", err)
	}

	// Count processing tasks
	processingCount, err := pendingCol.CountDocuments(ctx, bson.M{"status": StatusProcessing})
	if err != nil {
		log.Printf("Error counting processing tasks: %v", err)
	}

	// Count completed tasks
	completedCount, err := historyCol.CountDocuments(ctx, bson.M{"status": StatusCompleted})
	if err != nil {
		log.Printf("Error counting completed tasks: %v", err)
	}

	// Count failed tasks
	failedCount, err := historyCol.CountDocuments(ctx, bson.M{"status": StatusFailed})
	if err != nil {
		log.Printf("Error counting failed tasks: %v", err)
	}

	// Calculate average runtime from completed tasks
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"status": StatusCompleted, "pipeline_runtime_ms": bson.M{"$gt": 0}}}},
		{{Key: "$group", Value: bson.M{
			"_id":         nil,
			"avg_runtime": bson.M{"$avg": "$pipeline_runtime_ms"},
		}}},
	}

	cursor, err := historyCol.Aggregate(ctx, pipeline)
	if err != nil {
		log.Printf("Error calculating avg runtime: %v", err)
	}

	var avgResult struct {
		AvgRuntime float64 `bson:"avg_runtime"`
	}
	if cursor != nil && cursor.Next(ctx) {
		cursor.Decode(&avgResult)
		cursor.Close(ctx)
	}

	metrics := MetricsResponse{
		TotalPending:    pendingCount,
		TotalProcessing: processingCount,
		TotalCompleted:  completedCount,
		TotalFailed:     failedCount,
		AvgRuntimeMs:    int64(avgResult.AvgRuntime),
		TasksProcessed:  completedCount + failedCount,
		Timestamp:       time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(metrics)
}

// queueStatusHandler returns queue status grouped by repository
func (s *AgentIntelService) queueStatusHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pendingCol := s.mongoDB.Collection("pending_tasks")

	// Count total tasks
	totalCount, err := pendingCol.CountDocuments(ctx, bson.M{})
	if err != nil {
		http.Error(w, "Failed to count tasks", http.StatusInternalServerError)
		log.Printf("Error counting tasks: %v", err)
		return
	}

	// Group by repository
	pipeline := mongo.Pipeline{
		{{Key: "$group", Value: bson.M{
			"_id":   "$repository",
			"count": bson.M{"$sum": 1},
		}}},
	}

	cursor, err := pendingCol.Aggregate(ctx, pipeline)
	if err != nil {
		http.Error(w, "Failed to aggregate tasks", http.StatusInternalServerError)
		log.Printf("Error aggregating tasks: %v", err)
		return
	}
	defer cursor.Close(ctx)

	tasksByRepo := make(map[string]int)
	for cursor.Next(ctx) {
		var result struct {
			ID    string `bson:"_id"`
			Count int    `bson:"count"`
		}
		if err := cursor.Decode(&result); err != nil {
			log.Printf("Error decoding result: %v", err)
			continue
		}
		tasksByRepo[result.ID] = result.Count
	}

	// Group by status
	statusPipeline := mongo.Pipeline{
		{{Key: "$group", Value: bson.M{
			"_id":   "$status",
			"count": bson.M{"$sum": 1},
		}}},
	}

	statusCursor, err := pendingCol.Aggregate(ctx, statusPipeline)
	if err != nil {
		log.Printf("Error aggregating by status: %v", err)
	}
	defer statusCursor.Close(ctx)

	tasksByStatus := make(map[string]int)
	if statusCursor != nil {
		for statusCursor.Next(ctx) {
			var result struct {
				ID    string `bson:"_id"`
				Count int    `bson:"count"`
			}
			if err := statusCursor.Decode(&result); err != nil {
				log.Printf("Error decoding status result: %v", err)
				continue
			}
			tasksByStatus[result.ID] = result.Count
		}
	}

	status := QueueStatusResponse{
		TotalTasks:    totalCount,
		TasksByRepo:   tasksByRepo,
		TasksByStatus: tasksByStatus,
		Timestamp:     time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(status)
}
