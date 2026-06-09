package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
)

// GetMasterKey reads the master key from environment variable BBC_MCP_KEY.
func GetMasterKey() (string, error) {
	key := os.Getenv("BBC_MCP_KEY")
	if key == "" {
		return "", errors.New("crypto: 环境变量 BBC_MCP_KEY 未设置")
	}
	return key, nil
}

// deriveKey derives a 32-byte AES-256 key from a passphrase using SHA-256.
func deriveKey(masterKey string) []byte {
	h := sha256.Sum256([]byte(masterKey))
	return h[:]
}

// Encrypt encrypts plaintext using AES-256-GCM and returns a base64-encoded
// cipher string (nonce + ciphertext + tag).
func Encrypt(plaintext, masterKey string) (string, error) {
	key := deriveKey(masterKey)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("crypto: 创建 AES cipher 失败: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("crypto: 创建 GCM 失败: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("crypto: 生成 nonce 失败: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), nil)
	combined := append(nonce, ciphertext...)
	return base64.StdEncoding.EncodeToString(combined), nil
}

// Decrypt decrypts a base64-encoded cipher string produced by Encrypt.
func Decrypt(encoded, masterKey string) (string, error) {
	combined, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("crypto: base64 解码失败: %w", err)
	}

	key := deriveKey(masterKey)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("crypto: 创建 AES cipher 失败: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("crypto: 创建 GCM 失败: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(combined) < nonceSize {
		return "", errors.New("crypto: 密文数据太短")
	}

	nonce, ciphertext := combined[:nonceSize], combined[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("crypto: 解密失败（密钥错误或密文损坏）: %w", err)
	}

	return string(plaintext), nil
}

// DecryptIfNeeded attempts to decrypt the given string. If decryption fails
// (base64 decode or GCM auth), the original string is returned as-is,
// supporting gradual migration from plaintext to encrypted passwords.
// Empty strings are returned unchanged.
func DecryptIfNeeded(encoded string) (string, error) {
	if encoded == "" {
		return encoded, nil
	}

	// Try base64 decode as a fast filter: plaintext passwords are unlikely
	// to be valid base64. Even if they happen to be, GCM auth will reject them.
	combined, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return encoded, nil
	}

	key, err := GetMasterKey()
	if err != nil {
		return "", err
	}

	dek := deriveKey(key)
	block, err := aes.NewCipher(dek)
	if err != nil {
		return encoded, nil
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return encoded, nil
	}

	nonceSize := gcm.NonceSize()
	if len(combined) < nonceSize {
		return encoded, nil
	}

	nonce, ciphertext := combined[:nonceSize], combined[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return encoded, nil
	}

	return string(plaintext), nil
}

// GenerateKey generates a random 32-byte key and returns its hex encoding.
func GenerateKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("crypto: 生成密钥失败: %w", err)
	}
	return hex.EncodeToString(b), nil
}
