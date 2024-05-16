package token

import (
	"crypto/rand"
	"encoding/base64"
)

const tokenLength = 32

// GenerateToken generates a random token
func GenerateToken() (string, error) {
	token := make([]byte, tokenLength)
	_, err := rand.Read(token)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(token), nil
}
