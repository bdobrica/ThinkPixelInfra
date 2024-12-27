package middleware

import (
	"net/http"
	"api_gateway/auth"
	"api_gateway/utils"
)

// JWTMiddleware validates JWT for protected routes
func JWTMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString := r.Header.Get("Authorization")
		if tokenString == "" {
			utils.RespondWithError(w, http.StatusUnauthorized, "Missing token")
			return
		}

		_, err := auth.ValidateJWT(tokenString)
		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, "Invalid token")
			return
		}

		next.ServeHTTP(w, r)
	})
}
