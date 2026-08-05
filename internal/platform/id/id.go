package id

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

func New(prefix string, bytes int) (string, error) {
	if bytes < 16 {
		return "", fmt.Errorf("id entropy must be at least 16 bytes")
	}
	buffer := make([]byte, bytes)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	return prefix + "_" + base64.RawURLEncoding.EncodeToString(buffer), nil
}
