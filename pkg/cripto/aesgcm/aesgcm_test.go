package aesgcm

import (
	"encoding/hex"
	"testing"
)

func TestCryptAesgcm_EncryptDecrypt(t *testing.T) {
	t.Parallel()

	crypt := NewCrypt(Config{SecretKey: "12345678901234567890123456789012"})
	ciphertextHex, nonceHex, err := crypt.Encrypt([]byte("payload"))
	if err != nil {
		t.Fatalf("Encrypt returned error: %v", err)
	}

	ciphertext, err := hex.DecodeString(ciphertextHex)
	if err != nil {
		t.Fatalf("decode ciphertext: %v", err)
	}
	nonce, err := hex.DecodeString(nonceHex)
	if err != nil {
		t.Fatalf("decode nonce: %v", err)
	}

	plaintext, err := crypt.Decrypt(nonce, ciphertext)
	if err != nil {
		t.Fatalf("Decrypt returned error: %v", err)
	}
	if string(plaintext) != "payload" {
		t.Fatalf("expected payload, got %q", plaintext)
	}
}

func TestCryptAesgcm_InvalidKey(t *testing.T) {
	t.Parallel()

	crypt := NewCrypt(Config{SecretKey: "short"})
	if _, _, err := crypt.Encrypt([]byte("payload")); err == nil {
		t.Fatal("expected invalid key error")
	}
}
