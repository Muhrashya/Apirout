package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
)

// NormalizeKey pads or truncates any key string to exactly 32 bytes for AES-256
func NormalizeKey(rawKey string) []byte {
	key := make([]byte, 32)
	copy(key, []byte(rawKey))
	return key
}

// EncryptPassword encrypts plain text password using AES-256-GCM with a random nonce
func EncryptPassword(key []byte, password string) (string, error) {
	if password == "" {
		return "", nil
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// Seal appends the ciphertext and 16-byte authentication tag to nonce
	ciphertext := gcm.Seal(nonce, nonce, []byte(password), nil)
	return "gcm:" + base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptPassword decrypts base64 encoded AES-256 ciphertext (supports modern GCM with fallback to legacy CFB)
func DecryptPassword(key []byte, encrypted string) (string, error) {
	if encrypted == "" {
		return "", nil
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	// 1. Explicit GCM prefix: strictly verify authenticity and fail on tampering
	if strings.HasPrefix(encrypted, "gcm:") {
		payload := strings.TrimPrefix(encrypted, "gcm:")
		data, err := base64.StdEncoding.DecodeString(payload)
		if err != nil {
			return "", fmt.Errorf("invalid gcm base64: %w", err)
		}
		gcm, err := cipher.NewGCM(block)
		if err != nil {
			return "", err
		}
		if len(data) < gcm.NonceSize()+gcm.Overhead() {
			return "", errors.New("gcm ciphertext too short")
		}
		nonce := data[:gcm.NonceSize()]
		ciphertext := data[gcm.NonceSize():]
		plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
		if err != nil {
			return "", fmt.Errorf("gcm authentication failed (tampered ciphertext): %w", err)
		}
		return string(plaintext), nil
	}

	// 2. Unprefixed data: try standard base64 decode
	data, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		// Fallback for plaintext (e.g. initial migration)
		return encrypted, nil
	}

	// 2a. Try modern AES-GCM decryption
	gcm, err := cipher.NewGCM(block)
	if err == nil && len(data) >= gcm.NonceSize()+gcm.Overhead() {
		nonce := data[:gcm.NonceSize()]
		ciphertext := data[gcm.NonceSize():]
		if plaintext, err := gcm.Open(nil, nonce, ciphertext, nil); err == nil {
			return string(plaintext), nil
		}
	}

	// 2b. Fallback to legacy AES-CFB decryption for backwards compatibility
	if len(data) >= aes.BlockSize {
		iv := data[:aes.BlockSize]
		ciphertext := make([]byte, len(data)-aes.BlockSize)
		copy(ciphertext, data[aes.BlockSize:])

		stream := cipher.NewCFBDecrypter(block, iv)
		stream.XORKeyStream(ciphertext, ciphertext)
		return string(ciphertext), nil
	}

	return encrypted, errors.New("ciphertext too short")
}
