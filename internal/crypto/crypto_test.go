package crypto_test

import (
	"strings"
	"testing"

	"github.com/lerko/helm/internal/crypto"
)

func TestRoundtrip(t *testing.T) {
	plain := "hello world"
	ct, err := crypto.EncryptString(plain, "test-secret")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	got, err := crypto.DecryptString(ct, "test-secret")
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if got != plain {
		t.Errorf("got %q, want %q", got, plain)
	}
}

func TestWrongSecretFails(t *testing.T) {
	ct, err := crypto.EncryptString("secret data", "correct-secret")
	if err != nil {
		t.Fatal(err)
	}
	_, err = crypto.DecryptString(ct, "wrong-secret")
	if err == nil {
		t.Error("expected error decrypting with wrong secret, got nil")
	}
}

func TestCiphertextTooShort(t *testing.T) {
	// hex encoding of 3 bytes — shorter than GCM nonce size (12 bytes)
	_, err := crypto.DecryptString("aabbcc", "secret")
	if err == nil {
		t.Error("expected error for short ciphertext, got nil")
	}
}

func TestInvalidHex(t *testing.T) {
	_, err := crypto.DecryptString("not-hex!", "secret")
	if err == nil {
		t.Error("expected error for invalid hex, got nil")
	}
}

func TestEmptyPlaintext(t *testing.T) {
	ct, err := crypto.EncryptString("", "secret")
	if err != nil {
		t.Fatal(err)
	}
	got, err := crypto.DecryptString(ct, "secret")
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Errorf("got %q, want empty string", got)
	}
}

func TestNonceRandomness(t *testing.T) {
	ct1, _ := crypto.EncryptString("same", "secret")
	ct2, _ := crypto.EncryptString("same", "secret")
	if ct1 == ct2 {
		t.Error("two encrypts of same plaintext produced identical ciphertext — nonce is not random")
	}
}

func TestDeriveKeyLength(t *testing.T) {
	key := crypto.DeriveKey("any secret")
	if len(key) != 32 {
		t.Errorf("DeriveKey returned %d bytes, want 32", len(key))
	}
}

func TestLongPlaintext(t *testing.T) {
	plain := strings.Repeat("a", 1<<20) // 1 MB
	ct, err := crypto.EncryptString(plain, "secret")
	if err != nil {
		t.Fatal(err)
	}
	got, err := crypto.DecryptString(ct, "secret")
	if err != nil {
		t.Fatal(err)
	}
	if got != plain {
		t.Error("long plaintext roundtrip failed")
	}
}
