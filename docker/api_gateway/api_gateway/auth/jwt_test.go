package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// TestGenerateJWT tests JWT token generation
func TestGenerateJWT(t *testing.T) {
	tests := []struct {
		name      string
		apiKey    string
		hashedKey string
	}{
		{
			name:      "basic token generation",
			apiKey:    "test-api-key-123",
			hashedKey: "hashed-key-abc",
		},
		{
			name:      "token with different keys",
			apiKey:    "another-key-456",
			hashedKey: "hashed-key-def",
		},
		{
			name:      "token with long keys",
			apiKey:    "very-long-api-key-with-many-characters-12345678901234567890",
			hashedKey: "very-long-hashed-key-with-many-characters-12345678901234567890",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, expTime, err := GenerateJWT(tt.apiKey, tt.hashedKey)
			if err != nil {
				t.Fatalf("GenerateJWT() error = %v", err)
			}

			if token == "" {
				t.Error("Generated token should not be empty")
			}

			// Verify expiration time is in the future
			now := time.Now().Unix()
			if expTime <= now {
				t.Error("Expiration time should be in the future")
			}

			// Verify expiration is approximately 15 minutes from now
			expectedExpTime := now + 15*60
			if expTime < expectedExpTime-5 || expTime > expectedExpTime+5 {
				t.Errorf("Expiration time = %d, expected around %d", expTime, expectedExpTime)
			}

			// Verify token can be parsed
			parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
				return jwtSecret, nil
			})

			if err != nil {
				t.Fatalf("Failed to parse generated token: %v", err)
			}

			if !parsedToken.Valid {
				t.Error("Generated token should be valid")
			}

			// Verify claims
			if claims, ok := parsedToken.Claims.(jwt.MapClaims); ok {
				if sub, ok := claims["sub"].(string); !ok || sub != tt.apiKey {
					t.Errorf("Token sub = %v, want %s", claims["sub"], tt.apiKey)
				}
				if hashedKey, ok := claims["hashed_key"].(string); !ok || hashedKey != tt.hashedKey {
					t.Errorf("Token hashed_key = %v, want %s", claims["hashed_key"], tt.hashedKey)
				}
			} else {
				t.Error("Failed to extract claims from token")
			}
		})
	}
}

// TestValidateJWT tests JWT token validation
func TestValidateJWT(t *testing.T) {
	// Generate a valid token
	apiKey := "test-key"
	hashedKey := "hashed-test-key"
	validToken, _, err := GenerateJWT(apiKey, hashedKey)
	if err != nil {
		t.Fatalf("Failed to generate test token: %v", err)
	}

	tests := []struct {
		name      string
		token     string
		wantValid bool
		wantError bool
	}{
		{
			name:      "valid token",
			token:     validToken,
			wantValid: true,
			wantError: false,
		},
		{
			name:      "empty token",
			token:     "",
			wantValid: false,
			wantError: true,
		},
		{
			name:      "malformed token",
			token:     "not.a.token",
			wantValid: false,
			wantError: true,
		},
		{
			name:      "invalid signature",
			token:     "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ0ZXN0In0.invalidsignature",
			wantValid: false,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := ValidateJWT(tt.token)

			if tt.wantError {
				if err == nil {
					t.Error("ValidateJWT() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("ValidateJWT() unexpected error = %v", err)
			}

			if tt.wantValid {
				if !token.Valid {
					t.Error("Token should be valid")
				}
			}
		})
	}
}

// TestDecodeJWT tests extracting hashed key from JWT
func TestDecodeJWT(t *testing.T) {
	tests := []struct {
		name      string
		apiKey    string
		hashedKey string
	}{
		{
			name:      "decode basic token",
			apiKey:    "api-key-1",
			hashedKey: "hashed-key-1",
		},
		{
			name:      "decode with special characters",
			apiKey:    "api@key#123",
			hashedKey: "hashed$key%456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Generate token
			token, _, err := GenerateJWT(tt.apiKey, tt.hashedKey)
			if err != nil {
				t.Fatalf("Failed to generate token: %v", err)
			}

			// Decode token
			decodedHashedKey, err := DecodeJWT(token)
			if err != nil {
				t.Fatalf("DecodeJWT() error = %v", err)
			}

			if decodedHashedKey != tt.hashedKey {
				t.Errorf("DecodeJWT() = %s, want %s", decodedHashedKey, tt.hashedKey)
			}
		})
	}
}

// TestDecodeJWTInvalidToken tests DecodeJWT with invalid tokens
func TestDecodeJWTInvalidToken(t *testing.T) {
	tests := []struct {
		name  string
		token string
	}{
		{
			name:  "empty token",
			token: "",
		},
		{
			name:  "malformed token",
			token: "invalid.token.here",
		},
		{
			name:  "token without hashed_key claim",
			token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ0ZXN0In0.signature",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodeJWT(tt.token)
			if err == nil {
				t.Error("DecodeJWT() expected error for invalid token, got nil")
			}
		})
	}
}

// TestJWTTokenExpiration tests token expiration behavior
func TestJWTTokenExpiration(t *testing.T) {
	token, expTime, err := GenerateJWT("test-key", "hashed-key")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Verify expiration claim exists in token
	parsedToken, err := ValidateJWT(token)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatal("Failed to extract claims")
	}

	// Verify exp claim matches returned expTime
	expClaim, ok := claims["exp"].(float64)
	if !ok {
		t.Fatal("Token missing exp claim")
	}

	if int64(expClaim) != expTime {
		t.Errorf("Token exp claim = %d, want %d", int64(expClaim), expTime)
	}

	// Verify expiration is in the future
	now := time.Now().Unix()
	if int64(expClaim) <= now {
		t.Error("Token should not be expired")
	}
}

// TestMultipleJWTGeneration tests generating multiple tokens
func TestMultipleJWTGeneration(t *testing.T) {
	tokens := make(map[string]bool)

	// Generate 10 tokens
	for i := 0; i < 10; i++ {
		token, _, err := GenerateJWT("api-key", "hashed-key")
		if err != nil {
			t.Fatalf("Failed to generate token %d: %v", i, err)
		}

		// Note: Tokens will have slightly different exp times, so they will be unique
		// We just verify each token is valid
		_, err = ValidateJWT(token)
		if err != nil {
			t.Errorf("Token %d failed validation: %v", i, err)
		}

		tokens[token] = true
	}

	// Verify we got some tokens (they may not all be unique due to same second)
	if len(tokens) == 0 {
		t.Error("Should have generated at least one token")
	}
}

// TestJWTWithEmptyKeys tests JWT with empty strings
func TestJWTWithEmptyKeys(t *testing.T) {
	tests := []struct {
		name      string
		apiKey    string
		hashedKey string
	}{
		{
			name:      "empty api key",
			apiKey:    "",
			hashedKey: "hashed-key",
		},
		{
			name:      "empty hashed key",
			apiKey:    "api-key",
			hashedKey: "",
		},
		{
			name:      "both empty",
			apiKey:    "",
			hashedKey: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should still be able to generate token
			token, _, err := GenerateJWT(tt.apiKey, tt.hashedKey)
			if err != nil {
				t.Fatalf("GenerateJWT() error = %v", err)
			}

			// Token should be valid
			_, err = ValidateJWT(token)
			if err != nil {
				t.Errorf("ValidateJWT() error = %v", err)
			}

			// Decode should return the empty string if hashedKey was empty
			decodedHashedKey, err := DecodeJWT(token)
			if err != nil {
				t.Fatalf("DecodeJWT() error = %v", err)
			}

			if decodedHashedKey != tt.hashedKey {
				t.Errorf("DecodeJWT() = %s, want %s", decodedHashedKey, tt.hashedKey)
			}
		})
	}
}

// TestConcurrentJWTGeneration tests thread-safety of JWT generation
func TestConcurrentJWTGeneration(t *testing.T) {
	const numGoroutines = 100
	results := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			token, _, err := GenerateJWT("api-key", "hashed-key")
			if err != nil {
				results <- err
				return
			}

			_, err = ValidateJWT(token)
			results <- err
		}(i)
	}

	// Collect results
	for i := 0; i < numGoroutines; i++ {
		if err := <-results; err != nil {
			t.Errorf("Concurrent operation %d failed: %v", i, err)
		}
	}
}

// TestJWTClaims tests that all expected claims are present
func TestJWTClaims(t *testing.T) {
	apiKey := "test-api-key"
	hashedKey := "test-hashed-key"

	token, expTime, err := GenerateJWT(apiKey, hashedKey)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	parsedToken, err := ValidateJWT(token)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatal("Failed to extract claims")
	}

	// Check sub claim
	if sub, ok := claims["sub"].(string); !ok {
		t.Error("Token missing sub claim")
	} else if sub != apiKey {
		t.Errorf("sub = %s, want %s", sub, apiKey)
	}

	// Check hashed_key claim
	if hk, ok := claims["hashed_key"].(string); !ok {
		t.Error("Token missing hashed_key claim")
	} else if hk != hashedKey {
		t.Errorf("hashed_key = %s, want %s", hk, hashedKey)
	}

	// Check exp claim
	if exp, ok := claims["exp"].(float64); !ok {
		t.Error("Token missing exp claim")
	} else if int64(exp) != expTime {
		t.Errorf("exp = %d, want %d", int64(exp), expTime)
	}
}

// TestJWTSpecialCharacters tests JWT with special characters
func TestJWTSpecialCharacters(t *testing.T) {
	tests := []struct {
		name      string
		apiKey    string
		hashedKey string
	}{
		{
			name:      "unicode characters",
			apiKey:    "日本語-key",
			hashedKey: "hashed-日本語",
		},
		{
			name:      "special symbols",
			apiKey:    "key!@#$%^&*()",
			hashedKey: "hash!@#$%^&*()",
		},
		{
			name:      "spaces",
			apiKey:    "api key with spaces",
			hashedKey: "hashed key with spaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, _, err := GenerateJWT(tt.apiKey, tt.hashedKey)
			if err != nil {
				t.Fatalf("GenerateJWT() error = %v", err)
			}

			// Verify token can be validated
			_, err = ValidateJWT(token)
			if err != nil {
				t.Errorf("ValidateJWT() error = %v", err)
			}

			// Verify decoding preserves special characters
			decodedHashedKey, err := DecodeJWT(token)
			if err != nil {
				t.Fatalf("DecodeJWT() error = %v", err)
			}

			if decodedHashedKey != tt.hashedKey {
				t.Errorf("DecodeJWT() = %s, want %s", decodedHashedKey, tt.hashedKey)
			}
		})
	}
}
