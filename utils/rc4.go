package utils

import (
	"crypto/rand"
	"crypto/rc4"
	"crypto/sha256"
	"io"

	"golang.org/x/crypto/hkdf"
)

// change: ngubah fungsi endecryptRC4 biar pake HKDF untuk buat kunci file dari kunci utama dan IV, jadi lebih aman.
func encryptRC4(key, plaintext []byte) (iv, ciphertext []byte, err error) {
	iv = make([]byte, 16)
	rand.Read(iv)
	fileKey := make([]byte, 32)
	_, _ = io.ReadFull(hkdf.New(sha256.New, key, iv, []byte("RC4-file-key")), fileKey)
	stream, _ := rc4.NewCipher(fileKey)

	ciphertext = make([]byte, len(plaintext))
	stream.XORKeyStream(ciphertext, plaintext)
	return iv, ciphertext, nil
}

func decryptRC4(key, iv, ciphertext []byte) ([]byte, error) {
	// disini
	if len(iv) != 0 {
		fileKey := make([]byte, 32)
		_, _ = io.ReadFull(hkdf.New(sha256.New, key, iv, []byte("RC4-file-key")), fileKey)
		key = fileKey
	}
	stream, err := rc4.NewCipher(key)
	if err != nil {
		return nil, err
	}

	plaintext := make([]byte, len(ciphertext))
	stream.XORKeyStream(plaintext, ciphertext)
	return plaintext, nil
}
