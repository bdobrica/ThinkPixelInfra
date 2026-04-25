# Document Queue with Dead Letter Queue Support

This implementation provides a production-ready dead letter queue (DLQ) pattern for NATS core, without requiring JetStream. This is useful when you need reliable message processing with retry logic but want to avoid the complexity of JetStream.

## Features

- **Configurable Retry Logic**: Set maximum retry attempts per message
- **Automatic Dead Letter Queue**: Failed messages are moved to `<subject>.dlq` after exceeding retry limit
- **Message Status Tracking**: Track message processing state (PENDING, PROCESSING, COMPLETED, FAILED, DEAD_LETTER)
- **Error Tracking**: Store error messages and timestamps for debugging
- **Production Ready**: Handles edge cases and provides proper logging

## Message Structure

The `DocumentQueuePayload` now includes:

```protobuf
message DocumentQueuePayload {
  int32 site_id = 1;
  int32 id = 2;
  string text = 3;
  map<string, string> extra = 4;
  int64 timestamp_millis = 5;          // NEW: Consistent snake_case naming
  int32 retry_count = 6;               // NEW: Current retry count
  int32 max_retries = 7;               // NEW: Maximum retry attempts
  MessageStatus status = 8;            // NEW: Message processing status
  string error_message = 9;            // NEW: Last error message
  int64 last_retry_timestamp = 10;     // NEW: Timestamp of last retry
  string language = 11;                // NEW: Explicit text language for embeddings
}
```

## Message Status Enum

```protobuf
enum MessageStatus {
  PENDING = 0;      // Initial state
  PROCESSING = 1;   // Currently being processed
  COMPLETED = 2;    // Successfully processed
  FAILED = 3;       // Failed, will be retried
  DEAD_LETTER = 4;  // Moved to dead letter queue
}
```

## Usage

### 1. Creating Messages with Retry Settings

```go
// Create a new message with 3 retry attempts
payload := document_queue.NewDocumentQueuePayload(
    1,                          // site_id
    123,                        // document_id
    "Content to process",       // text
    "en",                       // language (or "auto")
    map[string]string{          // metadata
        "source": "api",
        "type": "article",
    },
    3,                          // max_retries (default: 3 if <= 0)
)
```

### 2. Publishing Messages

```go
dq := document_queue.NewDocumentQueue()
// Use the API_GATEWAY_DOCUMENT_QUEUE_SUBJECT env var to set the subject

if err := dq.Publish(payload); err != nil {
    log.Printf("Failed to publish: %v", err)
}
```

### 3. Subscribing with Dead Letter Queue

#### Option A: Using SubscriptionManager (Recommended)

The `SubscriptionManager` automatically handles both main queue and DLQ subscriptions:

```go
sm := document_queue.NewSubscriptionManager(dq)
go sm.Monitor() // Starts automatic subscription management
```

#### Option B: Manual Subscription

For more control, subscribe manually:

```go
mainHandler := func(payload *document_queue.DocumentQueuePayload) error {
    // Process the message
    // Return nil for success, error for failure
    if err := processMessage(payload); err != nil {
        return err // Will trigger retry logic
    }
    return nil
}

dlqHandler := func(payload *document_queue.DocumentQueuePayload) error {
    // Handle dead letter messages
    log.Printf("Dead letter: site=%d, id=%d, error=%s",
        payload.SiteId, payload.Id, payload.ErrorMessage)

    // Implement alerting, logging, or manual inspection
    return nil
}

mainSub, dlqSub, err := dq.SubscribeWithDLQ(mainHandler, dlqHandler)
if err != nil {
    log.Fatal("Subscription failed:", err)
}
```

## Dead Letter Queue Pattern

### How It Works

1. **Initial Processing**: Messages start with `status = PENDING`
2. **Processing**: Status changes to `PROCESSING` when handler is called
3. **Success**: Status changes to `COMPLETED` on successful processing
4. **Failure**:
   - Status changes to `FAILED`
   - `retry_count` is incremented
   - `error_message` and `last_retry_timestamp` are updated
   - Message is republished to original subject if retries < max_retries
5. **Dead Letter**:
   - If `retry_count >= max_retries`, status changes to `DEAD_LETTER`
   - Message is published to `<subject>.dlq`

### Subject Naming

- Main queue: `store.jobs`
- Dead letter queue: `store.jobs.dlq`

## Production Considerations

### 1. Dead Letter Queue Monitoring

Implement monitoring for DLQ messages:

```go
dlqHandler := func(payload *document_queue.DocumentQueuePayload) error {
    // Send alerts to monitoring system
    sendAlert("DLQ Message", map[string]interface{}{
        "site_id": payload.SiteId,
        "document_id": payload.Id,
        "error": payload.ErrorMessage,
        "retries": payload.RetryCount,
    })

    // Store in database for manual inspection
    storeDLQMessage(payload)

    return nil
}
```

### 2. Environment Configuration

Key environment variables:

```bash
API_GATEWAY_DOCUMENT_QUEUE_SUBJECT="store.jobs"  # Main queue subject
API_GATEWAY_NATS_URL="nats://nats:4222"          # NATS server URL
# ... other NATS configuration
```

### 3. Retry Strategy

Consider your retry strategy based on failure types:

- **Transient failures** (network, temporary service unavailable): Suitable for retry
- **Permanent failures** (invalid data, authorization): Consider lower retry counts
- **Critical failures**: May need immediate human intervention

### 4. Dead Letter Queue Processing

Options for handling DLQ messages:

1. **Manual Inspection**: Store in database for admin review
2. **Alerting**: Send notifications to operations team
3. **Reprocessing**: Implement manual requeue functionality
4. **Analysis**: Track failure patterns for system improvements

## Advantages Over JetStream

1. **Simplicity**: No JetStream configuration required
2. **Lightweight**: Lower resource usage
3. **Immediate Availability**: Works with any NATS core setup
4. **Control**: Full control over retry logic and DLQ behavior
5. **Flexibility**: Easy to customize retry strategies per message type

## Migration from Simple Queue

If you're migrating from a simple queue implementation:

1. Update message publishers to use `NewDocumentQueuePayload()`
2. Update subscribers to return errors instead of just logging
3. Add DLQ monitoring and handling
4. Test retry behavior with different failure scenarios
5. Monitor DLQ message volume in production

This implementation provides a robust foundation for reliable message processing in production environments while maintaining the simplicity of NATS core.
