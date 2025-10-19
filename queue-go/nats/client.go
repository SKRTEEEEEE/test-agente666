package nats

import (
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

const (
	// Stream and subject names
	StreamName        = "TASKS"
	SubjectTaskNew    = "tasks.new"
	SubjectTaskUpdate = "tasks.update"
	SubjectTaskDelete = "tasks.delete"
	SubjectTaskStatus = "tasks.status"

	// Consumer name
	ConsumerName = "task-workers"
)

// Client wraps NATS JetStream connection
type Client struct {
	nc *nats.Conn
	js nats.JetStreamContext
}

// NewClient creates a new NATS client and establishes connection
func NewClient(url string) (*Client, error) {
	// Connect to NATS server with retry logic
	nc, err := nats.Connect(
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
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Get JetStream context
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to get JetStream context: %w", err)
	}

	client := &Client{
		nc: nc,
		js: js,
	}

	// Initialize streams
	if err := client.initializeStreams(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to initialize streams: %w", err)
	}

	log.Printf("Successfully connected to NATS at %s", url)
	return client, nil
}

// initializeStreams creates or updates the required JetStream streams
func (c *Client) initializeStreams() error {
	// Define the TASKS stream configuration
	streamConfig := &nats.StreamConfig{
		Name:        StreamName,
		Description: "Task queue for Agent666",
		Subjects:    []string{"tasks.*"},
		Retention:   nats.WorkQueuePolicy, // Messages deleted after acknowledgment
		MaxAge:      7 * 24 * time.Hour,   // Keep messages for 7 days max
		Storage:     nats.FileStorage,     // Persist to disk
		Replicas:    1,                    // Single replica for now
		Discard:     nats.DiscardOld,      // Discard old messages when limits reached
	}

	// Try to get existing stream info
	_, err := c.js.StreamInfo(StreamName)
	if err != nil {
		// Stream doesn't exist, create it
		if err == nats.ErrStreamNotFound {
			log.Printf("Creating stream: %s", StreamName)
			_, err = c.js.AddStream(streamConfig)
			if err != nil {
				return fmt.Errorf("failed to create stream: %w", err)
			}
			log.Printf("Stream %s created successfully", StreamName)
		} else {
			return fmt.Errorf("failed to get stream info: %w", err)
		}
	} else {
		// Stream exists, update it
		log.Printf("Stream %s already exists, updating configuration", StreamName)
		_, err = c.js.UpdateStream(streamConfig)
		if err != nil {
			log.Printf("Warning: failed to update stream: %v", err)
		}
	}

	return nil
}

// CreateConsumer creates a durable consumer for processing tasks
func (c *Client) CreateConsumer() error {
	consumerConfig := &nats.ConsumerConfig{
		Durable:       ConsumerName,
		Description:   "Durable consumer for processing tasks",
		AckPolicy:     nats.AckExplicitPolicy, // Require explicit acknowledgment
		MaxDeliver:    3,                      // Maximum delivery attempts
		AckWait:       30 * time.Second,       // Wait 30s for acknowledgment
		DeliverPolicy: nats.DeliverNewPolicy,  // Only deliver new messages
		FilterSubject: SubjectTaskNew,         // Only subscribe to new tasks
		MaxAckPending: 100,                    // Max unacknowledged messages
		ReplayPolicy:  nats.ReplayInstantPolicy,
	}

	// Try to get existing consumer
	_, err := c.js.ConsumerInfo(StreamName, ConsumerName)
	if err != nil {
		if err == nats.ErrConsumerNotFound {
			log.Printf("Creating consumer: %s", ConsumerName)
			_, err = c.js.AddConsumer(StreamName, consumerConfig)
			if err != nil {
				return fmt.Errorf("failed to create consumer: %w", err)
			}
			log.Printf("Consumer %s created successfully", ConsumerName)
		} else {
			return fmt.Errorf("failed to get consumer info: %w", err)
		}
	} else {
		log.Printf("Consumer %s already exists", ConsumerName)
	}

	return nil
}

// Close closes the NATS connection
func (c *Client) Close() {
	if c.nc != nil {
		c.nc.Close()
		log.Println("NATS connection closed")
	}
}

// GetJetStream returns the JetStream context
func (c *Client) GetJetStream() nats.JetStreamContext {
	return c.js
}

// GetConn returns the underlying NATS connection
func (c *Client) GetConn() *nats.Conn {
	return c.nc
}

// IsConnected checks if the NATS connection is active
func (c *Client) IsConnected() bool {
	return c.nc != nil && c.nc.IsConnected()
}
