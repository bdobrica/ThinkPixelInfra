package main

import (
	"net/http"

	"api_gateway/auth"
	"api_gateway/document_queue"
	"api_gateway/handlers"
	"api_gateway/logger"
	"api_gateway/middleware"
	"api_gateway/ping"
	"api_gateway/register"

	"github.com/gorilla/mux"
)

func main() {
	// Initialize Router
	r := mux.NewRouter()
	// Initialize the Document Queue
	dq := document_queue.NewDocumentQueue()

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
	go sm.Monitor()

	// Start Server
	logger.Infof("API Gateway is running on port 8080")
	logger.Fatalf("API Gateway failed to start: %v", http.ListenAndServe(":8080", r))
}
