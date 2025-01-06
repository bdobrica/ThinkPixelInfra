package register

import (
	"crypto/sha256"
	"fmt"
)

func saltMessage(message string, salt string) string {
	hash := sha256.Sum256([]byte(message + salt))
	return fmt.Sprintf("%x", hash) // Hexadecimal encoding
}
