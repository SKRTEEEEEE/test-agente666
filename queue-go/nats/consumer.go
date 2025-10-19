package nats

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/nats-io/nats.go"
)

// TaskHandler is a function that processes a task message
type TaskHandler func(task *TaskMessage) error

// Subscribe creates a durable subscription to process tasks
func (c *Client) Subscribe(handler TaskHandler) (*nats.Subscription, error) {
	// Ensure consumer exists
	if err := c.CreateConsumer(); err != nil {
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	// Subscribe to the durable consumer
	sub, err := c.js.PullSubscribe(SubjectTaskNew, ConsumerName)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe: %w", err)
	}

	log.Printf("Successfully subscribed to %s with consumer %s", SubjectTaskNew, ConsumerName)
	return sub, nil
}

// ProcessMessages continuously processes messages from the subscription
func (c *Client) ProcessMessages(sub *nats.Subscription, handler TaskHandler) {
	log.Println("Starting message processing loop...")

	for {
		// Fetch a batch of messages (max 10 at a time)
		msgs, err := sub.Fetch(10, nats.MaxWait(5))
		if err != nil {
			// Timeout is expected when no messages available
			if err != nats.ErrTimeout {
				log.Printf("Error fetching messages: %v", err)
			}
			continue
		}

		for _, msg := range msgs {
			if err := c.processMessage(msg, handler); err != nil {
				log.Printf("Error processing message: %v", err)
			}
		}
	}
}

// processMessage processes a single message
func (c *Client) processMessage(msg *nats.Msg, handler TaskHandler) error {
	// Parse the task message
	var task TaskMessage
	if err := json.Unmarshal(msg.Data, &task); err != nil {
		log.Printf("Failed to unmarshal task message: %v", err)
		// Acknowledge to remove bad message from queue
		msg.Nak()
		return fmt.Errorf("failed to unmarshal: %w", err)
	}

	log.Printf("Processing task: ID=%s, IssueID=%s, Status=%s", task.ID, task.IssueID, task.Status)

	// Process the task
	if err := handler(&task); err != nil {
		log.Printf("Handler failed for task %s: %v", task.ID, err)
		// Negative acknowledgment - message will be redelivered
		msg.Nak()
		return err
	}

	// Acknowledge successful processing
	if err := msg.Ack(); err != nil {
		log.Printf("Failed to acknowledge message: %v", err)
		return err
	}

	log.Printf("Successfully processed task: ID=%s", task.ID)
	return nil
}

// SubscribeToStatusUpdates subscribes to task status updates
func (c *Client) SubscribeToStatusUpdates(handler func(*StatusUpdateMessage) error) (*nats.Subscription, error) {
	subject := fmt.Sprintf("%s.*", SubjectTaskStatus)

	sub, err := c.js.Subscribe(subject, func(msg *nats.Msg) {
		var update StatusUpdateMessage
		if err := json.Unmarshal(msg.Data, &update); err != nil {
			log.Printf("Failed to unmarshal status update: %v", err)
			msg.Nak()
			return
		}

		if err := handler(&update); err != nil {
			log.Printf("Handler failed for status update: %v", err)
			msg.Nak()
			return
		}

		msg.Ack()
	}, nats.Durable("status-updates"), nats.ManualAck())

	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to status updates: %w", err)
	}

	return sub, nil
}

// SubscribeToDeletes subscribes to task deletion messages
func (c *Client) SubscribeToDeletes(handler func(*DeleteMessage) error) (*nats.Subscription, error) {
	sub, err := c.js.Subscribe(SubjectTaskDelete, func(msg *nats.Msg) {
		var delMsg DeleteMessage
		if err := json.Unmarshal(msg.Data, &delMsg); err != nil {
			log.Printf("Failed to unmarshal delete message: %v", err)
			msg.Nak()
			return
		}

		if err := handler(&delMsg); err != nil {
			log.Printf("Handler failed for delete message: %v", err)
			msg.Nak()
			return
		}

		msg.Ack()
	}, nats.Durable("task-deletes"), nats.ManualAck())

	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to deletes: %w", err)
	}

	return sub, nil
}
