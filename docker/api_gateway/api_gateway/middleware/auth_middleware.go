package middleware

import (
	"context"
	"net/http"
	"strings"

	"api_gateway/auth"
	"api_gateway/utils"
)

// ContextKey defines the type for context keys to avoid conflicts
type ContextKey string

const CacheEntryKey ContextKey = "cacheEntry"

// JWTMiddleware validates JWT for protected routes and passes CacheEntry to handlers
func JWTMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString := r.Header.Get("Authorization")
		if tokenString == "" {
			utils.RespondWithError(w, http.StatusUnauthorized, "Missing token")
			return
		}

		// Remove "Bearer " prefix if present
		tokenString = strings.TrimPrefix(tokenString, "Bearer ")

		// Validate the JWT and extract claims
		_, err := auth.ValidateJWT(tokenString)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, "Invalid token")
			return
		}

		hashedKey, err := auth.DecodeJWT(tokenString)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Error decoding token: "+err.Error())
			return
		}

		cacheEntry, err := auth.GetCachedAPIKeyData(hashedKey)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Error retrieving cache entry: "+err.Error())
			return
		}

		// Embed CacheEntry into the request context
		ctx := context.WithValue(r.Context(), CacheEntryKey, cacheEntry)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
