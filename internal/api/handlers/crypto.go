package handlers

import (
	"github.com/lerko/helm/internal/crypto"
)

// encryptString encrypts plaintext with AES-256-GCM using the given secret.
func encryptString(plaintext, secret string) (string, error) {
	return crypto.EncryptString(plaintext, secret)
}

// decryptString decrypts a hex-encoded nonce+ciphertext produced by encryptString.
func decryptString(cipherHex, secret string) (string, error) {
	return crypto.DecryptString(cipherHex, secret)
}
