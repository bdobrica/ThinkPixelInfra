# Changelog

## [0.6.0] - 2025-12-29

### Added - Hybrid Dead Letter Queue & Enhanced Observability

**1. Redis-Based Hybrid DLQ Solution**
- Replaced database-based DLQ with scalable Redis + S3 hybrid architecture
- Redis stores failed messages ephemerally with configurable TTL
- Periodic S3 export for long-term archival and compliance
- CLI tool (`dlq-tool`) for operational management and message reinjection
- New HTTP endpoints:
  - `/dlq/stats`: Real-time DLQ statistics
  - `/dlq/messages`: Query and retrieve failed messages
- Deployment updated with DLQ log volume mounting
- Full configuration via environment variables

**2. Request ID Tracking & Context-Aware Logging**
- Added `middleware/request_id.go` for X-Request-ID generation and propagation
- Enhanced logger with context-aware functions: `InfofCtx()`, `ErrorfCtx()`, `WarningfCtx()`
- All handlers and middleware updated to include request IDs in logs
- Improved distributed tracing across system components
- Better debugging and troubleshooting capabilities

**3. Store Handler Soft-Fail Behavior**
- Partial success handling: returns 202 Accepted if at least one document queued successfully
- Returns 500 only if all documents fail
- Updated `StoreResponse` struct with detailed failure information
- Improved resilience for batch operations

**4. Enhanced Health Checks**
- Added `/ready` endpoint for Kubernetes readiness probes
- Database health validation with 2-second timeout
- Returns detailed health status with timestamps
- JSON response includes individual component checks

**5. Prometheus Metrics & Observability**
- Comprehensive metrics for HTTP requests, NATS messaging, model APIs, and database operations
- `/metrics` endpoint for Prometheus scraping
- Automatic instrumentation via middleware
- Periodic connection pool metrics collection
- Key metrics include:
  - `api_gateway_requests_total`, `api_gateway_request_duration_seconds`
  - `api_gateway_nats_messages_published_total`, `api_gateway_nats_dlq_messages_total`
  - `api_gateway_db_queries_total`, `api_gateway_db_connections`

**6. NATS Error Handling Improvements**
- Enhanced unmarshal error logging with message details and hex dumps
- Automatic poison message routing to DLQ (subject + ".dlq")
- Prevents infinite redelivery of malformed messages
- Better debugging with truncated byte inspection

### Changed

**1. Cache Management**
- Background goroutines in `qdrantconn.go` and `redisconn.go` for automatic cleanup
- Expired client cache entries removed periodically (half of TTL interval, max 1 hour)
- Prevents memory leaks from stale connections
- INFO-level logging for monitoring

**2. Context Key Management**
- Introduced `ctxkeys` package for centralized context key definitions
- Refactored all handlers and middleware to use `ctxkeys.CacheEntryKey` and `ctxkeys.DocumentQueueKey`
- Eliminates context key conflicts
- Improved code maintainability

### Testing

**Unit Tests Added:**
- `auth/jwt_test.go`: JWT generation, validation, expiration (23.1% coverage)
- `config/config_test.go`: Environment variable parsing, edge cases (100% coverage)
- `document_queue/document_queue_test.go`: Protobuf marshaling, state transitions (6.8% coverage)
- `logger/logger_test.go`: Log levels, formatting, concurrency (34.6% coverage)
- Total coverage: 9.0% of statements

**Integration Tests Added:**
- `tests/test-graceful-shutdown.sh`: Validates shutdown sequence
- `tests/test-store-handler.sh`: Tests soft-fail behavior
- `tests/test-database-nats.sh`: Database health and NATS error handling
- `tests/test-metrics.sh`: Prometheus metrics validation
- Makefile targets: `test-store`, `test-db-nats`, `test-metrics`

### Environment Variables Added

- Redis DLQ configuration (see DLQ_README.md for full list)
- S3 export settings for DLQ archival
- Health check timeout configuration

---

## [0.5.1] - 2025-12-26

### Added - Graceful Shutdown & Context Propagation

#### Critical Fixes (Issues #1, #3, #4)

**1. Graceful Shutdown in main.go**
- Added signal handling for SIGTERM, SIGINT, and os.Interrupt
- Implemented proper shutdown sequence:
  1. Stop accepting new HTTP requests (30s timeout)
  2. Cancel root context to signal all goroutines
  3. Stop SubscriptionManager
  4. Drain and close NATS connection
  5. Close database connections
- Added HTTP server timeouts:
  - ReadTimeout: 15s
  - WriteTimeout: 30s
  - IdleTimeout: 120s

**2. Context Propagation**
- Added root context that propagates to all long-running operations
- `SubscriptionManager.Monitor()` now accepts `context.Context`
- Uses `select` with `ctx.Done()` for graceful cancellation
- Replaced `time.Sleep()` with `time.Ticker` and `select` for better control

**3. NATS Connection Lifecycle**
- Added `DocumentQueue.Close()` method
- Properly drains NATS connection before closing
- Prevents message loss during shutdown
- Logs all shutdown steps for debugging

**4. SubscriptionManager Lifecycle**
- Added `SubscriptionManager.Stop()` method
- Gracefully unsubscribes from NATS subjects
- Prevents goroutine leaks

**5. Database Connection Pooling**
- Configured connection pool limits:
  - `SetMaxOpenConns(25)` - maximum concurrent connections
  - `SetMaxIdleConns(5)` - idle connection pool size
  - `SetConnMaxLifetime(5m)` - maximum connection lifetime
- Added environment variable overrides:
  - `API_GATEWAY_DB_MAX_OPEN_CONNS`
  - `API_GATEWAY_DB_MAX_IDLE_CONNS`
  - `API_GATEWAY_DB_CONN_MAX_LIFETIME`
- Added logging for connection pool configuration

### Testing

To test graceful shutdown in Kubernetes:
```bash
# Deploy updated version
kubectl apply -k k8s/apps/thinkpixel/dev

# Watch logs while deleting pod
kubectl logs -f deployment/api-gateway -c api-gateway &
kubectl delete pod -l app.kubernetes.io/name=api-gateway

# Expected log output:
# Received shutdown signal: terminated. Starting graceful shutdown...
# HTTP server shutdown complete
# Subscription manager stopped
# NATS connection closed successfully
# Database connection closed
# Graceful shutdown complete
```

To test locally:
```bash
# Run API Gateway
go run main.go

# In another terminal, send SIGTERM
pkill -TERM api_gateway

# Or use Ctrl+C (SIGINT)
```

### Environment Variables Added

- `API_GATEWAY_DB_MAX_OPEN_CONNS` (default: 25)
- `API_GATEWAY_DB_MAX_IDLE_CONNS` (default: 5)
- `API_GATEWAY_DB_CONN_MAX_LIFETIME` (default: 5m)

### Breaking Changes

- `SubscriptionManager.Monitor()` now requires `context.Context` parameter
- This only affects internal usage in `main.go`, no external API changes

### Files Modified

1. `api_gateway/main.go` - Complete rewrite with graceful shutdown
2. `document_queue/document_queue.go` - Added `Close()` method
3. `document_queue/subscription_manager.go` - Added context support and `Stop()` method
4. `db/db.go` - Added connection pooling configuration

### Next Steps

✅ Issues #1, #3, #4 are now **RESOLVED**

Ready to proceed with:
- Issue #6: Fix StoreHandler payload creation
- Issue #2: Implement soft-fail in StoreHandler
- Issue #8: Add Prometheus metrics
