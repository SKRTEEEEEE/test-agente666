package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	// Get configuration from environment
	port := getEnv("PORT", "8082")
	mongoURL := getEnv("MONGO_URL", "mongodb://localhost:27017")
	natsURL := getEnv("NATS_URL", "nats://localhost:4222")
	dbName := getEnv("DB_NAME", "agent_intel")

	log.Println("Agent Intel Service starting...")
	log.Printf("Port: %s", port)
	log.Printf("MongoDB URL: %s", mongoURL)
	log.Printf("NATS URL: %s", natsURL)
	log.Printf("Database: %s", dbName)

	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURL))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongoClient.Disconnect(ctx)

	// Ping MongoDB
	if err := mongoClient.Ping(ctx, nil); err != nil {
		log.Fatalf("Failed to ping MongoDB: %v", err)
	}
	log.Println("Connected to MongoDB")

	db := mongoClient.Database(dbName)

	// Create indexes
	if err := createIndexes(ctx, db); err != nil {
		log.Printf("Warning: Failed to create indexes: %v", err)
	}

	// Create service instance
	service := &AgentIntelService{
		mongoDB:     db,
		mongoClient: mongoClient,
		natsURL:     natsURL,
	}

	// Start NATS event consumer
	consumer, err := NewEventConsumer(natsURL, db, mongoClient)
	if err != nil {
		log.Fatalf("Failed to create event consumer: %v", err)
	}
	defer consumer.Close()

	if err := consumer.Start(); err != nil {
		log.Fatalf("Failed to start event consumer: %v", err)
	}
	log.Println("Event consumer started")

	// Setup HTTP routes
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", healthHandler)

	// Queue endpoints
	mux.HandleFunc("/api/v1/queue/next", service.getNextTaskHandler)
	mux.HandleFunc("/api/v1/queue/status", service.queueStatusHandler)

	// Task management
	mux.HandleFunc("/api/v1/tasks/cancel", service.cancelTaskHandler)

	// Metrics
	mux.HandleFunc("/api/v1/metrics", service.metricsHandler)

	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("HTTP server listening on port %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Server stopped")
}

// createIndexes creates MongoDB indexes for better performance
func createIndexes(ctx context.Context, db *mongo.Database) error {
	pendingCol := db.Collection("pending_tasks")
	historyCol := db.Collection("task_history")

	// Index on task_id (unique)
	_, err := pendingCol.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    map[string]interface{}{"task_id": 1},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return err
	}

	// Index on repository
	_, err = pendingCol.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: map[string]interface{}{"repository": 1},
	})
	if err != nil {
		return err
	}

	// Index on status
	_, err = pendingCol.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: map[string]interface{}{"status": 1},
	})
	if err != nil {
		return err
	}

	// Index on created_at for sorting
	_, err = pendingCol.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: map[string]interface{}{"created_at": -1},
	})
	if err != nil {
		return err
	}

	// History indexes
	_, err = historyCol.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    map[string]interface{}{"task_id": 1},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return err
	}

	_, err = historyCol.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: map[string]interface{}{
			"repository": 1,
			"status":     1,
		},
	})
	if err != nil {
		return err
	}

	log.Println("MongoDB indexes created")
	return nil
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
