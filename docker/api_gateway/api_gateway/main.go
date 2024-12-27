package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"api_gateway/auth"
	"api_gateway/handlers"
	"api_gateway/middleware"
)

func main() {
	// Initialize Router
	r := mux.NewRouter()

	// Auth Routes
	r.HandleFunc("/auth/token", auth.AuthHandler).Methods("POST")

	// Protected Routes
	r.Handle("/store", middleware.JWTMiddleware(http.HandlerFunc(handlers.StoreHandler))).Methods("POST")
	r.Handle("/search", middleware.JWTMiddleware(http.HandlerFunc(handlers.SearchHandler))).Methods("POST")

	// Start Server
	log.Println("API Gateway is running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
