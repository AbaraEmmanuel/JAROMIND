package utils

import (
	"crypto/rand"
	"encoding/hex"
)

func GenerateVerificationCode() (string, error) {
	bytes := make([]byte, 10)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
