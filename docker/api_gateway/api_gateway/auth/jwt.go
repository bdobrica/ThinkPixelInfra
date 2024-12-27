package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v4"
	"api_gateway/config"
)

// Use a secure method for managing secrets
var jwtSecret = []byte(config.GetEnv("API_GATEWAY_JWT_SECRET", "supersecretkey"))

// GenerateJWT creates a short-lived token for stateless authentication
func GenerateJWT(apiKey string) (string, error) {
	claims := jwt.MapClaims{
		"sub": apiKey,
		"exp": time.Now().Add(15 * time.Minute).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ValidateJWT verifies the provided JWT
func ValidateJWT(tokenString string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
}
