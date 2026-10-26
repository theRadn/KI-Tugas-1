package utils

import (
	"crypto/cipher"
	"crypto/des"
	"crypto/rand"
	"io"
)

func encryptDESCTR(key, plaintext []byte) (iv, ciphertext []byte, err error) {
	block, err := des.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}

	iv = make([]byte, des.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, nil, err
	}

	ciphertext = make([]byte, len(plaintext))
	cipher.NewCTR(block, iv).XORKeyStream(ciphertext, plaintext)
	return iv, ciphertext, nil
}

func decryptDESCTR(key, iv, ciphertext []byte) ([]byte, error) {
	block, err := des.NewCipher(key)
	if err != nil {
		return nil, err
	}

	plaintext := make([]byte, len(ciphertext))
	cipher.NewCTR(block, iv).XORKeyStream(plaintext, ciphertext)
	return plaintext, nil
}
