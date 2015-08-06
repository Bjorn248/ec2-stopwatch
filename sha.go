package main

import (
	// "encoding/json"
	"crypto/sha256"
	"fmt"
)

func generateSha256String(plaintext string) string {
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(plaintext)))
	return hash
}
