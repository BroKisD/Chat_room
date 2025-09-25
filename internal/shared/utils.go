package shared

import (
	"crypto/rand"
	"encoding/hex"
)

// GenerateID generates a random ID string
func GenerateID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// IsValidUsername checks if a username is valid
func IsValidUsername(username string) bool {
	if len(username) < 3 || len(username) > 20 {
		return false
	}
	// Add more validation rules as needed
	return true
}
