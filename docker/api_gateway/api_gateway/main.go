package main

import (
	"net/http"

	"github.com/gorilla/mux"
	"api_gateway/auth"
	"api_gateway/handlers"
	"api_gateway/logger"
	"api_gateway/middleware"
	"api_gateway/ping"
)

func main() {
	// Initialize Router
	r := mux.NewRouter()

	// Ping Route
	r.HandleFunc("/ping", ping.PingHandler).Methods("GET")

	// Auth Routes
	r.HandleFunc("/auth/token", auth.AuthHandler).Methods("POST")

	// Protected Routes
	r.Handle("/store", middleware.JWTMiddleware(http.HandlerFunc(handlers.StoreHandler))).Methods("POST")
	r.Handle("/search", middleware.JWTMiddleware(http.HandlerFunc(handlers.SearchHandler))).Methods("POST")

	// Start Server
	logger.Infof("API Gateway is running on port 8080")
	logger.Fatalf("API Gateway failed to start: %v", http.ListenAndServe(":8080", r))
}
