package register

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"
)

// GenerateVerificationToken generates an alphanumeric verification token with a prefix
func generateVerificationToken() string {
	prefix := "verify-"
	timestamp := time.Now().UnixNano()
	hash := sha256.Sum256([]byte(fmt.Sprintf("%d", timestamp)))
	token := base64.URLEncoding.EncodeToString(hash[:])
	return prefix + token[:16] // Use the first 16 characters for brevity
}

// GenerateAPIKey generates an alphanumeric API key with a prefix
func generateAPIKey() string {
	prefix := "api-key-"
	timestamp := time.Now().UnixNano()
	hash := sha256.Sum256([]byte(fmt.Sprintf("%d", timestamp)))
	key := base64.URLEncoding.EncodeToString(hash[:])
	return prefix + key[:24] // Use the first 24 characters for a longer key
}
