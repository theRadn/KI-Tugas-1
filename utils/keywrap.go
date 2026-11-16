package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/hkdf"
	"crypto/sha256"
)

type WrappedKey struct {
	Version    string `json:"version"`
	Nonce      []byte `json:"nonce"`
	Ciphertext []byte `json:"ciphertext"`
}

func GenerateKey(size int) ([]byte, error) {
	if size <= 0 {
		return nil, errors.New("key size must be positive")
	}
	key := make([]byte, size)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}
	return key, nil
}

// MasterKEK reads a Base64-encoded 32-byte key from DATABASE_MASTER_KEK.
func MasterKEK() ([]byte, error) {
	encoded := os.Getenv("DATABASE_MASTER_KEK")
	if encoded == "" {
		return nil, errors.New("DATABASE_MASTER_KEK belum diatur")
	}
	key, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("DATABASE_MASTER_KEK bukan Base64 yang valid: %w", err)
	}
	if len(key) != 32 {
		return nil, errors.New("DATABASE_MASTER_KEK harus berisi tepat 32 byte")
	}
	return key, nil
}

func WrapKey(kek, plaintext, aad []byte) (WrappedKey, error) {
	block, err := aes.NewCipher(kek)
	if err != nil {
		return WrappedKey{}, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return WrappedKey{}, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return WrappedKey{}, err
	}

	return WrappedKey{
		Version:    "v1",
		Nonce:      nonce,
		Ciphertext: gcm.Seal(nil, nonce, plaintext, aad),
	}, nil
}

func UnwrapKey(kek []byte, wrapped WrappedKey, aad []byte) ([]byte, error) {
	if wrapped.Version != "v1" {
		return nil, errors.New("unsupported wrapped key version")
	}
	block, err := aes.NewCipher(kek)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(wrapped.Nonce) != gcm.NonceSize() {
		return nil, errors.New("invalid nonce")
	}
	return gcm.Open(nil, wrapped.Nonce, wrapped.Ciphertext, aad)
}

func MarshalWrappedKey(wrapped WrappedKey) ([]byte, error) {
	return json.Marshal(wrapped)
}

func UnmarshalWrappedKey(data []byte) (WrappedKey, error) {
	var wrapped WrappedKey
	err := json.Unmarshal(data, &wrapped)
	return wrapped, err
}

// DeriveMACKey derives a separate HMAC key for utils.Encrypt/Decrypt.
func DeriveMACKey(dek, info []byte) ([]byte, error) {
	out := make([]byte, 32)
	if _, err := io.ReadFull(hkdf.New(sha256.New, dek, nil, info), out); err != nil {
		return nil, err
	}
	return out, nil
}
