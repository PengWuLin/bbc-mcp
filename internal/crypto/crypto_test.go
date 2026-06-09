package crypto

import (
	"encoding/base64"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	key := "my-secret-key-123"
	plaintext := "YkRPTXf6EIChMOvz"

	encrypted, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt() failed: %v", err)
	}
	if encrypted == plaintext {
		t.Fatal("Encrypt() returned plaintext unchanged")
	}

	decrypted, err := Decrypt(encrypted, key)
	if err != nil {
		t.Fatalf("Decrypt() failed: %v", err)
	}
	if decrypted != plaintext {
		t.Fatalf("Decrypt() = %q, want %q", decrypted, plaintext)
	}
}

func TestDecryptIfNeededPlaintextPassthrough(t *testing.T) {
	t.Setenv("BBC_MCP_KEY", "some-key")
	plaintext := ".pwl123456"
	result, err := DecryptIfNeeded(plaintext)
	if err != nil {
		t.Fatalf("DecryptIfNeeded(%q) failed: %v", plaintext, err)
	}
	if result != plaintext {
		t.Fatalf("DecryptIfNeeded(%q) = %q, want %q", plaintext, result, plaintext)
	}
}

func TestDecryptWrongKey(t *testing.T) {
	encrypted, err := Encrypt("secret-password", "correct-key")
	if err != nil {
		t.Fatalf("Encrypt() failed: %v", err)
	}

	_, err = Decrypt(encrypted, "wrong-key")
	if err == nil {
		t.Fatal("Decrypt() with wrong key should return error")
	}
}

func TestDecryptTampered(t *testing.T) {
	encrypted, err := Encrypt("secret-password", "test-key")
	if err != nil {
		t.Fatalf("Encrypt() failed: %v", err)
	}

	// Decode, tamper a byte, re-encode
	raw, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		t.Fatalf("base64 decode failed: %v", err)
	}
	raw[len(raw)-5] ^= 0x01
	tampered := base64.StdEncoding.EncodeToString(raw)

	_, err = Decrypt(tampered, "test-key")
	if err == nil {
		t.Fatal("Decrypt() of tampered ciphertext should fail (GCM auth)")
	}
}

func TestEncryptSameInputDifferentOutput(t *testing.T) {
	key := "test-key"
	plaintext := "same-password"

	result1, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("first Encrypt() failed: %v", err)
	}
	result2, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("second Encrypt() failed: %v", err)
	}
	if result1 == result2 {
		t.Fatal("two Encrypt() calls should produce different ciphertexts (random nonce)")
	}
}

func TestDeriveKey(t *testing.T) {
	k1 := deriveKey("master-secret")
	k2 := deriveKey("master-secret")
	if string(k1) != string(k2) {
		t.Fatal("deriveKey() should be deterministic")
	}
	if len(k1) != 32 {
		t.Fatalf("deriveKey() length = %d, want 32", len(k1))
	}
}

func TestEmptyPassword(t *testing.T) {
	result, err := DecryptIfNeeded("")
	if err != nil {
		t.Fatalf("DecryptIfNeeded('') failed: %v", err)
	}
	if result != "" {
		t.Fatalf("DecryptIfNeeded('') = %q, want ''", result)
	}
}

func TestDecryptIfNeededWithEncryptedValue(t *testing.T) {
	key := "bbc-mcp-test-key"
	t.Setenv("BBC_MCP_KEY", key)
	plaintext := "my-db-password"

	encrypted, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt() failed: %v", err)
	}

	result, err := DecryptIfNeeded(encrypted)
	if err != nil {
		t.Fatalf("DecryptIfNeeded() failed: %v", err)
	}
	if result != plaintext {
		t.Fatalf("DecryptIfNeeded() = %q, want %q", result, plaintext)
	}
}

func TestDecryptIfNeededWrongKeyFallsBack(t *testing.T) {
	encrypted, err := Encrypt("secret", "key-a")
	if err != nil {
		t.Fatalf("Encrypt() failed: %v", err)
	}
	t.Setenv("BBC_MCP_KEY", "key-b")

	result, err := DecryptIfNeeded(encrypted)
	if err != nil {
		t.Fatalf("DecryptIfNeeded() should not error on wrong key: %v", err)
	}
	// With wrong key, GCM auth fails, so original (encrypted) string is returned
	if result != encrypted {
		t.Fatalf("DecryptIfNeeded() with wrong key should return original: got %q", result)
	}
}

func TestGenerateKey(t *testing.T) {
	k1, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey() failed: %v", err)
	}
	k2, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey() failed: %v", err)
	}
	if k1 == k2 {
		t.Fatal("two GenerateKey() calls should produce different keys")
	}
	if len(k1) != 64 { // 32 bytes hex-encoded = 64 chars
		t.Fatalf("GenerateKey() hex length = %d, want 64", len(k1))
	}
}
