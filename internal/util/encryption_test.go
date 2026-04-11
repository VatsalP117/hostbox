package util

import (
	"encoding/hex"
	"testing"
)

// testKey is a valid 32-byte (64 hex char) key for tests.
var testKey = hex.EncodeToString([]byte("01234567890123456789012345678901"))

func TestEncryptDecryptRoundTrip(t *testing.T) {
	plaintext := "my-secret-value"
	encrypted, err := Encrypt(plaintext, testKey)
	if err != nil {
		t.Fatalf("Encrypt() error: %v", err)
	}
	decrypted, err := Decrypt(encrypted, testKey)
	if err != nil {
		t.Fatalf("Decrypt() error: %v", err)
	}
	if decrypted != plaintext {
		t.Errorf("Decrypt() = %q, want %q", decrypted, plaintext)
	}
}

func TestEncryptDecryptEmpty(t *testing.T) {
	encrypted, err := Encrypt("", testKey)
	if err != nil {
		t.Fatalf("Encrypt() error: %v", err)
	}
	decrypted, err := Decrypt(encrypted, testKey)
	if err != nil {
		t.Fatalf("Decrypt() error: %v", err)
	}
	if decrypted != "" {
		t.Errorf("Decrypt() = %q, want empty string", decrypted)
	}
}

func TestEncryptDecryptUnicode(t *testing.T) {
	plaintext := "こんにちは世界 🌍"
	encrypted, err := Encrypt(plaintext, testKey)
	if err != nil {
		t.Fatalf("Encrypt() error: %v", err)
	}
	decrypted, err := Decrypt(encrypted, testKey)
	if err != nil {
		t.Fatalf("Decrypt() error: %v", err)
	}
	if decrypted != plaintext {
		t.Errorf("Decrypt() = %q, want %q", decrypted, plaintext)
	}
}

func TestDecryptWrongKey(t *testing.T) {
	encrypted, err := Encrypt("secret", testKey)
	if err != nil {
		t.Fatalf("Encrypt() error: %v", err)
	}
	wrongKey := hex.EncodeToString([]byte("different-key-different-key-1234"))
	_, err = Decrypt(encrypted, wrongKey)
	if err == nil {
		t.Error("Decrypt() with wrong key should return error")
	}
}

func TestDecryptCorruptedCiphertext(t *testing.T) {
	encrypted, err := Encrypt("secret", testKey)
	if err != nil {
		t.Fatalf("Encrypt() error: %v", err)
	}
	// Corrupt by changing a character
	corrupted := encrypted[:len(encrypted)-2] + "ff"
	_, err = Decrypt(corrupted, testKey)
	if err == nil {
		t.Error("Decrypt() with corrupted ciphertext should return error")
	}
}

func TestDecryptTruncated(t *testing.T) {
	_, err := Decrypt("aabb", testKey)
	if err == nil {
		t.Error("Decrypt() with truncated input should return error")
	}
}

func TestEncryptInvalidKey(t *testing.T) {
	_, err := Encrypt("test", "short")
	if err == nil {
		t.Error("Encrypt() with short key should return error")
	}
}

func TestEncryptKeyNot64Hex(t *testing.T) {
	_, err := Encrypt("test", "abcd")
	if err == nil {
		t.Error("Encrypt() with 4 hex char key should return error")
	}
}

func TestTwoEncryptionsDiffer(t *testing.T) {
	enc1, err := Encrypt("same-value", testKey)
	if err != nil {
		t.Fatal(err)
	}
	enc2, err := Encrypt("same-value", testKey)
	if err != nil {
		t.Fatal(err)
	}
	if enc1 == enc2 {
		t.Error("two encryptions of same plaintext should produce different ciphertexts")
	}
}
