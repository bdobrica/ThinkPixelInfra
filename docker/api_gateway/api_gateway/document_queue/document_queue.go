package document_queue

import (
	"api_gateway/config"
	"fmt"

	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto" // import generated pb.go
)

// natsConn is the global NATS connection used for publishing and subscribing to document queue messages.
var natsConn *nats.Conn

// Init sets up the document queue system using NATS.
// It reads environment variables for connection details, authentication, and TLS configuration.
// Returns an error if connection or authentication fails.
func Init() error {
	natsURL := config.GetEnv("API_GATEWAY_NATS_URL", "nats://localhost:4222")
	nkeyPath := config.GetEnv("API_GATEWAY_NATS_NKEY_PATH", "/etc/nats/nkeys/default.nk")
	tlsCertPath := config.GetEnv("API_GATEWAY_NATS_TLS_CERT", "/etc/nats/tls/cert.pem")
	tlsKeyPath := config.GetEnv("API_GATEWAY_NATS_TLS_KEY", "/etc/nats/tls/key.pem")
	tlsCAPath := config.GetEnv("API_GATEWAY_NATS_TLS_CA", "/etc/nats/tls/ca.pem")

	var opts []nats.Option

	// NKey authentication
	if nkeyPath != "" {
		opt, err := nats.NkeyOptionFromSeed(nkeyPath)
		if err != nil {
			return fmt.Errorf("failed to create NKey option: %w", err)
		}
		opts = append(opts, opt)
	}

	// TLS configuration
	if tlsCertPath != "" && tlsKeyPath != "" {
		opts = append(opts, nats.ClientCert(tlsCertPath, tlsKeyPath))
		opts = append(opts, nats.RootCAs(tlsCAPath))
	}

	conn, err := nats.Connect(natsURL, opts...)
	if err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}
	natsConn = conn
	return nil
}

// Publish sends a protobuf-encoded DocumentQueuePayload to the specified NATS subject.
// Returns an error if the connection is not initialized or if marshaling fails.
func Publish(subject string, payload *DocumentQueuePayload) error {
	if natsConn == nil {
		return fmt.Errorf("NATS connection not initialized")
	}
	data, err := proto.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal protobuf payload: %w", err)
	}
	return natsConn.Publish(subject, data)
}

// Subscribe registers a handler function for messages on the given NATS subject.
// The handler receives each message as a decoded DocumentQueuePayload.
// Returns the subscription object and any error encountered during setup.
func Subscribe(subject string, handler func(payload *DocumentQueuePayload)) (*nats.Subscription, error) {
	if natsConn == nil {
		return nil, fmt.Errorf("NATS connection not initialized")
	}
	sub, err := natsConn.Subscribe(subject, func(msg *nats.Msg) {
		payload := &DocumentQueuePayload{}
		if err := proto.Unmarshal(msg.Data, payload); err != nil {
			// Handle unmarshal error
			return
		}
		handler(payload)
	})
	if err != nil {
		return nil, err
	}
	return sub, nil
}
