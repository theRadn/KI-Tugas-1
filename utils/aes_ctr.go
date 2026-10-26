package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
)

func encryptAESCTR(key, plaintext []byte) (iv, ciphertext []byte, err error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}

	iv = make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, nil, err
	}

	ciphertext = make([]byte, len(plaintext))
	cipher.NewCTR(block, iv).XORKeyStream(ciphertext, plaintext)
	return iv, ciphertext, nil
}

func decryptAESCTR(key, iv, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	plaintext := make([]byte, len(ciphertext))
	cipher.NewCTR(block, iv).XORKeyStream(plaintext, ciphertext)
	return plaintext, nil
}
