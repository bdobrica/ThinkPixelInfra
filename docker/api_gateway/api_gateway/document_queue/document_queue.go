package document_queue

import (
	"api_gateway/config"
	"api_gateway/logger"
	"api_gateway/metrics"
	"fmt"
	"os"
	"time"

	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto" // import generated pb.go
)

// DocumentQueue is a stateful object for managing NATS connection and document queue operations.
type DocumentQueue struct {
	natsConn *nats.Conn
	subject  string
}

// NewDocumentQueue initializes and returns a new DocumentQueue object.
// It reads environment variables for connection details, authentication, and TLS configuration.
// Returns the DocumentQueue and any error encountered.
func NewDocumentQueue() *DocumentQueue {
	natsURL := config.GetEnv("API_GATEWAY_NATS_URL", "nats://nats:4222")
	nkeyPath := config.GetEnv("API_GATEWAY_NATS_NKEY_PATH", "/etc/nats/nkeys/default.nk")
	tlsCertPath := config.GetEnv("API_GATEWAY_NATS_TLS_CERT", "/etc/nats/tls/cert.pem")
	tlsKeyPath := config.GetEnv("API_GATEWAY_NATS_TLS_KEY", "/etc/nats/tls/key.pem")
	tlsCAPath := config.GetEnv("API_GATEWAY_NATS_TLS_CA", "/etc/nats/tls/ca.pem")
	subject := config.GetEnv("API_GATEWAY_DOCUMENT_QUEUE_SUBJECT", "store.jobs")

	var opts []nats.Option

	// NKey authentication
	if nkeyPath != "" {
		if _, err := os.Stat(nkeyPath); err == nil {
			opt, err := nats.NkeyOptionFromSeed(nkeyPath)
			if err != nil {
				logger.Warningf("Failed to create NKey option: %v", err)
				return nil
			}
			opts = append(opts, opt)
		} else {
			logger.Warningf("NKey file not found: %s. Skipping NKey authentication.", nkeyPath)
		}
	}

	// TLS configuration
	if tlsCertPath != "" && tlsKeyPath != "" {
		_, certErr := os.Stat(tlsCertPath)
		_, keyErr := os.Stat(tlsKeyPath)
		if certErr == nil && keyErr == nil {
			opts = append(opts, nats.ClientCert(tlsCertPath, tlsKeyPath))
		} else {
			logger.Warningf("TLS certificate or key file not found: %s, %s. Skipping TLS authentication.", tlsCertPath, tlsKeyPath)
		}
	}
	if tlsCAPath != "" {
		_, caErr := os.Stat(tlsCAPath)
		if caErr == nil {
			opts = append(opts, nats.RootCAs(tlsCAPath))
		} else {
			logger.Warningf("TLS CA file not found: %s. Using system CA certificates.", tlsCAPath)
		}
	}

	// Connect to NATS server
	conn, err := nats.Connect(natsURL, opts...)
	if err != nil {
		logger.Errorf("Failed to connect to NATS: %v", err)
		return nil
	}
	return &DocumentQueue{natsConn: conn, subject: subject}
}

// publish sends a protobuf-encoded DocumentQueuePayload to the specified NATS subject.
// Returns an error if the connection is not initialized or if marshaling fails.
func (dq *DocumentQueue) publish(subject string, payload *DocumentQueuePayload) error {
	if dq.natsConn == nil {
		metrics.NATSMessagesPublished.WithLabelValues("failure").Inc()
		return fmt.Errorf("NATS connection not initialized")
	}
	data, err := proto.Marshal(payload)
	if err != nil {
		metrics.NATSMessagesPublished.WithLabelValues("failure").Inc()
		return fmt.Errorf("failed to marshal protobuf payload: %w", err)
	}
	err = dq.natsConn.Publish(subject, data)
	if err != nil {
		metrics.NATSMessagesPublished.WithLabelValues("failure").Inc()
	} else {
		metrics.NATSMessagesPublished.WithLabelValues("success").Inc()
	}
	return err
}

// Publish sends a protobuf-encoded DocumentQueuePayload to the default NATS subject.
// Returns an error if the connection is not initialized or if marshaling fails.
func (dq *DocumentQueue) Publish(payload *DocumentQueuePayload) error {
	return dq.publish(dq.subject, payload)
}

// subscribe registers a handler function for messages on the given NATS subject.
// The handler receives each message as a decoded DocumentQueuePayload.
// Returns the subscription object and any error encountered during setup.
func (dq *DocumentQueue) subscribe(subject string, handler func(payload *DocumentQueuePayload)) (*nats.Subscription, error) {
	if dq.natsConn == nil {
		return nil, fmt.Errorf("NATS connection not initialized")
	}
	sub, err := dq.natsConn.Subscribe(subject, func(msg *nats.Msg) {
		payload := &DocumentQueuePayload{}
		if err := proto.Unmarshal(msg.Data, payload); err != nil {
			// Record unmarshal error metric
			metrics.NATSUnmarshalErrors.Inc()

			// Log unmarshal error with message details
			logger.Errorf("Failed to unmarshal NATS message on subject %s: %v (message size: %d bytes, first 100 bytes: %x)",
				subject, err, len(msg.Data), truncateBytes(msg.Data, 100))

			// Create error payload for poison message
			dlqSubject := subject + ".dlq"
			poisonPayload := &DocumentQueuePayload{
				Status:       MessageStatus_FAILED,
				ErrorMessage: fmt.Sprintf("Unmarshal failed on %s: %v", subject, err),
				RetryCount:   0,
				MaxRetries:   0,
			}

			// Try to publish to DLQ (best effort, don't block)
			go func() {
				// Record DLQ message metric
				metrics.NATSDLQMessages.Inc()

				if pubErr := dq.publish(dlqSubject, poisonPayload); pubErr != nil {
					logger.Errorf("Failed to publish poison message to DLQ %s: %v", dlqSubject, pubErr)
				} else {
					logger.Infof("Poison message sent to DLQ: %s", dlqSubject)
				}
			}()

			// Note: For NATS Core (non-JetStream), messages are auto-acked
			// No explicit Ack() needed - just return to drop the message
			metrics.NATSMessagesProcessed.WithLabelValues("failure").Inc()
			return
		}

		// Process the message
		handler(payload)
		metrics.NATSMessagesProcessed.WithLabelValues("success").Inc()
	})
	if err != nil {
		return nil, err
	}
	return sub, nil
}

// Subscribe registers a handler function for messages on the default NATS subject.
// The handler receives each message as a decoded DocumentQueuePayload.
// Returns the subscription object and any error encountered during setup.
func (dq *DocumentQueue) Subscribe(handler func(payload *DocumentQueuePayload)) (*nats.Subscription, error) {
	return dq.subscribe(dq.subject, handler)
}

// handleRetry manages the retry logic and dead letter queue functionality
func (dq *DocumentQueue) handleRetry(originalSubject string, payload *DocumentQueuePayload, errorMsg string) {
	payload.RetryCount++
	payload.ErrorMessage = errorMsg
	payload.LastRetryTimestamp = getCurrentTimestampMillis()

	if payload.RetryCount >= payload.MaxRetries {
		// Send to dead letter queue
		payload.Status = MessageStatus_DEAD_LETTER
		dlqSubject := originalSubject + ".dlq"

		if err := dq.publish(dlqSubject, payload); err != nil {
			logger.Errorf("Failed to publish to dead letter queue %s: %v", dlqSubject, err)
		} else {
			logger.Warningf("Message moved to dead letter queue %s after %d retries: %s",
				dlqSubject, payload.RetryCount, errorMsg)
		}
	} else {
		// Retry on original subject
		payload.Status = MessageStatus_FAILED

		if err := dq.publish(originalSubject, payload); err != nil {
			logger.Errorf("Failed to republish message for retry: %v", err)
		} else {
			logger.Infof("Message republished for retry %d/%d on subject %s",
				payload.RetryCount, payload.MaxRetries, originalSubject)
		}
	}
}

// NewDocumentQueuePayload creates a new DocumentQueuePayload with default retry settings
func NewDocumentQueuePayload(siteID, id int32, text string, extra map[string]string, maxRetries int32) *DocumentQueuePayload {
	if maxRetries <= 0 {
		maxRetries = int32(config.GetEnvInt("API_GATEWAY_DOCUMENT_QUEUE_MAX_RETRIES", 3))
	}

	return &DocumentQueuePayload{
		SiteId:             siteID,
		Id:                 id,
		Text:               text,
		Extra:              extra,
		TimestampMillis:    getCurrentTimestampMillis(),
		RetryCount:         0,
		MaxRetries:         maxRetries,
		Status:             MessageStatus_PENDING,
		ErrorMessage:       "",
		LastRetryTimestamp: 0,
	}
}

// SubscribeWithDLQ subscribes to both the main subject and its dead letter queue
// Returns subscriptions for both main and DLQ subjects
func (dq *DocumentQueue) SubscribeWithDLQ(handler func(payload *DocumentQueuePayload) error, dlqHandler func(payload *DocumentQueuePayload) error) (*nats.Subscription, *nats.Subscription, error) {
	if dq.natsConn == nil {
		return nil, nil, fmt.Errorf("NATS connection not initialized")
	}

	// Subscribe to main subject with retry logic
	mainSub, err := dq.natsConn.Subscribe(dq.subject, func(msg *nats.Msg) {
		payload := &DocumentQueuePayload{}
		if err := proto.Unmarshal(msg.Data, payload); err != nil {
			logger.Errorf("Failed to unmarshal message: %v", err)
			return
		}

		// Set status to processing
		payload.Status = MessageStatus_PROCESSING

		if err := handler(payload); err != nil {
			// Handle retry logic
			dq.handleRetry(dq.subject, payload, err.Error())
		} else {
			// Mark as completed on success
			payload.Status = MessageStatus_COMPLETED
		}
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to subscribe to main subject: %w", err)
	}

	// Subscribe to dead letter queue
	dlqSubject := dq.subject + config.GetEnv("API_GATEWAY_DOCUMENT_QUEUE_DLQ_SUFFIX", ".dlq")
	dlqSub, err := dq.natsConn.Subscribe(dlqSubject, func(msg *nats.Msg) {
		payload := &DocumentQueuePayload{}
		if err := proto.Unmarshal(msg.Data, payload); err != nil {
			logger.Errorf("Failed to unmarshal DLQ message: %v", err)
			return
		}

		if dlqHandler != nil {
			if err := dlqHandler(payload); err != nil {
				logger.Errorf("DLQ handler failed for message: %v", err)
			}
		} else {
			// Default DLQ handling - just log the message
			logger.Errorf("Dead letter message received for site %d, ID %d: %s",
				payload.SiteId, payload.Id, payload.ErrorMessage)
		}
	})

	if err != nil {
		mainSub.Unsubscribe() // Clean up main subscription
		return nil, nil, fmt.Errorf("failed to subscribe to DLQ subject: %w", err)
	}

	return mainSub, dlqSub, nil
}

// getCurrentTimestampMillis returns the current timestamp in milliseconds
func getCurrentTimestampMillis() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

// truncateBytes returns the first n bytes of data, or all of it if shorter
func truncateBytes(data []byte, n int) []byte {
	if len(data) <= n {
		return data
	}
	return data[:n]
}

// Close gracefully closes the NATS connection
// It drains the connection first to ensure all buffered messages are sent
func (dq *DocumentQueue) Close() error {
	if dq.natsConn == nil {
		return nil
	}

	// Drain the connection to flush any pending messages
	if err := dq.natsConn.Drain(); err != nil {
		logger.Warningf("Error draining NATS connection: %v", err)
		// Continue with close even if drain fails
	}

	// Close the connection
	dq.natsConn.Close()
	logger.Infof("NATS connection closed successfully")
	return nil
}
