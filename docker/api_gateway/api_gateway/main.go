package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"api_gateway/auth"
	"api_gateway/db"
	"api_gateway/document_queue"
	"api_gateway/handlers"
	"api_gateway/logger"
	"api_gateway/middleware"
	"api_gateway/ping"
	"api_gateway/register"

	"github.com/gorilla/mux"
)

func main() {
	// Create root context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize Router
	r := mux.NewRouter()

	// Initialize the Document Queue
	dq := document_queue.NewDocumentQueue()
	if dq == nil {
		logger.Fatalf("Failed to initialize DocumentQueue")
	}

	// Ping Route
	r.HandleFunc("/ping", ping.PingHandler).Methods("GET")

	// Auth Routes
	r.HandleFunc("/auth/token", auth.AuthHandler).Methods("POST")

	// Protected Routes
	r.Handle("/store", middleware.JWTMiddleware(middleware.LoggingMiddleware(middleware.DocumentQueueMiddleware(http.HandlerFunc(handlers.StoreHandler), dq)))).Methods("POST")
	r.Handle("/store/max_batch_text_size", middleware.JWTMiddleware(http.HandlerFunc(handlers.MaxBatchTextSizeHandler))).Methods("POST")
	r.Handle("/search", middleware.JWTMiddleware(http.HandlerFunc(handlers.SearchHandler))).Methods("POST")
	r.Handle("/remove/embedding", middleware.JWTMiddleware(middleware.LoggingMiddleware(http.HandlerFunc(handlers.RemoveEmbeddingsHandler)))).Methods("POST")
	r.Handle("/remove/embeddings", middleware.JWTMiddleware(middleware.LoggingMiddleware(http.HandlerFunc(handlers.RemoveMultipleEmbeddingsHandler)))).Methods("POST")
	r.Handle("/remove/embedding_by_offset", middleware.JWTMiddleware(http.HandlerFunc(handlers.RemoveEmbeddingByOffsetHandler))).Methods("POST")
	r.Handle("/remove/embeddings_by_offset", middleware.JWTMiddleware(http.HandlerFunc(handlers.RemoveMultipleEmbeddingsByOffsetHandler))).Methods("POST")
	r.Handle("/refresh/apikey", middleware.JWTMiddleware(http.HandlerFunc(handlers.RefreshAPIKeyHandler))).Methods("POST")

	// Register Routes
	r.HandleFunc("/register", register.RegisterHandler).Methods("POST")

	// Document Queue Subscription
	sm := document_queue.NewSubscriptionManager(dq)
	go sm.Monitor(ctx)

	// Configure HTTP server with timeouts
	srv := &http.Server{
		Addr:         ":8080",
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Channel to listen for errors from the HTTP server
	serverErrors := make(chan error, 1)

	// Start HTTP server in a goroutine
	go func() {
		logger.Infof("API Gateway is running on port 8080")
		serverErrors <- srv.ListenAndServe()
	}()

	// Channel to listen for interrupt or terminate signals
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	// Block until we receive a signal or server error
	select {
	case err := <-serverErrors:
		logger.Fatalf("Server error: %v", err)
	case sig := <-shutdown:
		logger.Infof("Received shutdown signal: %v. Starting graceful shutdown...", sig)

		// Cancel context to stop background goroutines
		cancel()

		// Give outstanding requests a deadline for completion
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		// Shutdown HTTP server gracefully
		if err := srv.Shutdown(shutdownCtx); err != nil {
			logger.Errorf("HTTP server shutdown error: %v", err)
			// Force close if graceful shutdown fails
			if err := srv.Close(); err != nil {
				logger.Errorf("HTTP server close error: %v", err)
			}
		} else {
			logger.Infof("HTTP server shutdown complete")
		}

		// Stop subscription manager
		if err := sm.Stop(); err != nil {
			logger.Errorf("Failed to stop subscription manager: %v", err)
		} else {
			logger.Infof("Subscription manager stopped")
		}

		// Close NATS connection
		if err := dq.Close(); err != nil {
			logger.Errorf("Failed to close NATS connection: %v", err)
		} else {
			logger.Infof("NATS connection closed")
		}

		// Close database connection
		if err := db.CloseDBConnection(); err != nil {
			logger.Errorf("Failed to close database connection: %v", err)
		} else {
			logger.Infof("Database connection closed")
		}

		logger.Infof("Graceful shutdown complete")
	}
}
