package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"testing"
)

func TestEncryptDecryptPassword(t *testing.T) {
	key := NormalizeKey("my-super-secret-key-12345")
	rawPassword := "DatabaseP@ssw0rd!2026"

	encrypted, err := EncryptPassword(key, rawPassword)
	if err != nil {
		t.Fatalf("EncryptPassword failed: %v", err)
	}

	if encrypted == rawPassword {
		t.Fatalf("encrypted password should not equal raw password")
	}

	decrypted, err := DecryptPassword(key, encrypted)
	if err != nil {
		t.Fatalf("DecryptPassword failed: %v", err)
	}

	if decrypted != rawPassword {
		t.Fatalf("expected decrypted password %q, got %q", rawPassword, decrypted)
	}
}

func TestEncryptDecrypt_EmptyPassword(t *testing.T) {
	key := NormalizeKey("key")
	enc, err := EncryptPassword(key, "")
	if err != nil || enc != "" {
		t.Fatalf("expected empty result on empty password, got %q, err: %v", enc, err)
	}

	dec, err := DecryptPassword(key, "")
	if err != nil || dec != "" {
		t.Fatalf("expected empty result on empty decrypt, got %q, err: %v", dec, err)
	}
}

func TestDecryptPassword_PlaintextFallback(t *testing.T) {
	key := NormalizeKey("key")
	plain := "plain-not-base64-!@#"
	dec, err := DecryptPassword(key, plain)
	if err != nil {
		t.Fatalf("expected fallback for non-base64 plaintext, got err: %v", err)
	}
	if dec != plain {
		t.Fatalf("expected %q, got %q", plain, dec)
	}
}

func TestNormalizeKey(t *testing.T) {
	k1 := NormalizeKey("short")
	if len(k1) != 32 {
		t.Fatalf("expected key length 32, got %d", len(k1))
	}

	k2 := NormalizeKey("this-is-a-very-long-key-exceeding-thirty-two-bytes-for-aes-256")
	if len(k2) != 32 {
		t.Fatalf("expected key length 32, got %d", len(k2))
	}
}

func TestDecryptPassword_LegacyCFB(t *testing.T) {
	key := NormalizeKey("test-legacy-key")
	rawPassword := "LegacySecret123"

	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatalf("cipher err: %v", err)
	}
	iv := []byte("1234567890123456")
	ciphertext := make([]byte, len(rawPassword))
	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext, []byte(rawPassword))
	legacyPayload := base64.StdEncoding.EncodeToString(append(iv, ciphertext...))

	dec, err := DecryptPassword(key, legacyPayload)
	if err != nil {
		t.Fatalf("expected legacy CFB decrypt to succeed: %v", err)
	}
	if dec != rawPassword {
		t.Fatalf("expected %q, got %q", rawPassword, dec)
	}
}

func TestDecryptPassword_TamperedGCM(t *testing.T) {
	key := NormalizeKey("tamper-test-key-32-bytes-secure!")
	plaintext := "confidential-database-password-999"

	enc, err := EncryptPassword(key, plaintext)
	if err != nil {
		t.Fatalf("failed to encrypt: %v", err)
	}

	// Verify tampering with the ciphertext causes authentication error
	encBytes := []byte(enc)
	// Flip the last character in the base64 string
	if encBytes[len(encBytes)-1] == 'A' {
		encBytes[len(encBytes)-1] = 'B'
	} else {
		encBytes[len(encBytes)-1] = 'A'
	}
	tampered := string(encBytes)

	_, err = DecryptPassword(key, tampered)
	if err == nil {
		t.Fatalf("expected tampered ciphertext to fail decryption with AEAD error, but succeeded")
	}
}

