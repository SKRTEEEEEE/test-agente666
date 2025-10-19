package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"
)

const (
	// Stream and subject names
	StreamName     = "TASKS"
	SubjectTaskNew = "tasks.new"
	ConsumerName   = "task-workers"
)

func main() {
	// Get NATS URL from environment
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}

	log.Printf("Queue Worker starting...")
	log.Printf("NATS URL: %s", natsURL)

	// Connect to NATS with retry logic
	nc, err := connectWithRetry(natsURL, 10, 5*time.Second)
	if err != nil {
		log.Fatalf("Failed to connect to NATS after retries: %v", err)
	}
	defer nc.Close()

	log.Println("Successfully connected to NATS")

	// Get JetStream context
	js, err := nc.JetStream()
	if err != nil {
		log.Fatalf("Failed to get JetStream context: %v", err)
	}

	// Ensure stream exists (should be created by queue-go, but double check)
	ensureStream(js)

	// Create or get consumer
	ensureConsumer(js)

	// Subscribe to tasks
	sub, err := js.PullSubscribe(SubjectTaskNew, ConsumerName)
	if err != nil {
		log.Fatalf("Failed to subscribe: %v", err)
	}

	log.Println("Successfully subscribed to task queue")
	log.Println("Waiting for tasks...")

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start processing messages
	go processMessages(sub)

	// Wait for shutdown signal
	<-sigChan
	log.Println("Shutting down gracefully...")
	sub.Unsubscribe()
	nc.Close()
}

// connectWithRetry attempts to connect to NATS with retries
func connectWithRetry(url string, maxRetries int, delay time.Duration) (*nats.Conn, error) {
	var nc *nats.Conn
	var err error

	for i := 0; i < maxRetries; i++ {
		nc, err = nats.Connect(
			url,
			nats.MaxReconnects(10),
			nats.ReconnectWait(2*time.Second),
			nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
				if err != nil {
					log.Printf("NATS disconnected: %v", err)
				}
			}),
			nats.ReconnectHandler(func(nc *nats.Conn) {
				log.Printf("NATS reconnected to %s", nc.ConnectedUrl())
			}),
		)

		if err == nil {
			return nc, nil
		}

		log.Printf("Connection attempt %d/%d failed: %v. Retrying in %v...", i+1, maxRetries, err, delay)
		time.Sleep(delay)
	}

	return nil, err
}

// ensureStream ensures the TASKS stream exists
func ensureStream(js nats.JetStreamContext) {
	streamConfig := &nats.StreamConfig{
		Name:        StreamName,
		Description: "Task queue for Agent666",
		Subjects:    []string{"tasks.*"},
		Retention:   nats.WorkQueuePolicy,
		MaxAge:      7 * 24 * time.Hour,
		Storage:     nats.FileStorage,
		Replicas:    1,
		Discard:     nats.DiscardOld,
	}

	_, err := js.StreamInfo(StreamName)
	if err == nats.ErrStreamNotFound {
		log.Printf("Stream %s not found, creating...", StreamName)
		_, err = js.AddStream(streamConfig)
		if err != nil {
			log.Printf("Warning: failed to create stream: %v", err)
		} else {
			log.Printf("Stream %s created successfully", StreamName)
		}
	}
}

// ensureConsumer ensures the consumer exists
func ensureConsumer(js nats.JetStreamContext) {
	consumerConfig := &nats.ConsumerConfig{
		Durable:       ConsumerName,
		Description:   "Durable consumer for processing tasks",
		AckPolicy:     nats.AckExplicitPolicy,
		MaxDeliver:    3,
		AckWait:       30 * time.Second,
		DeliverPolicy: nats.DeliverNewPolicy,
		FilterSubject: SubjectTaskNew,
		MaxAckPending: 100,
		ReplayPolicy:  nats.ReplayInstantPolicy,
	}

	_, err := js.ConsumerInfo(StreamName, ConsumerName)
	if err == nats.ErrConsumerNotFound {
		log.Printf("Consumer %s not found, creating...", ConsumerName)
		_, err = js.AddConsumer(StreamName, consumerConfig)
		if err != nil {
			log.Printf("Warning: failed to create consumer: %v", err)
		} else {
			log.Printf("Consumer %s created successfully", ConsumerName)
		}
	}
}

// processMessages continuously processes messages from the queue
func processMessages(sub *nats.Subscription) {
	for {
		// Fetch a batch of messages (max 10 at a time)
		msgs, err := sub.Fetch(10, nats.MaxWait(5*time.Second))
		if err != nil {
			// Timeout is expected when no messages available
			if err != nats.ErrTimeout {
				log.Printf("Error fetching messages: %v", err)
			}
			continue
		}

		for _, msg := range msgs {
			processTask(msg)
		}
	}
}

// processTask processes a single task message
func processTask(msg *nats.Msg) {
	log.Printf("Received task: %s", string(msg.Data))

	// Simulate task processing
	// In a real implementation, this would:
	// 1. Parse the task message
	// 2. Execute the task (e.g., run Agent666 on the issue)
	// 3. Update task status via API or NATS
	// 4. Handle errors and retries

	// For now, just acknowledge the message
	time.Sleep(1 * time.Second) // Simulate work

	if err := msg.Ack(); err != nil {
		log.Printf("Failed to acknowledge message: %v", err)
		return
	}

	log.Printf("Task processed successfully")
}
