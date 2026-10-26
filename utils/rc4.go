package utils

import "crypto/rc4"

func encryptRC4(key, plaintext []byte) ([]byte, error) {
	stream, err := rc4.NewCipher(key)
	if err != nil {
		return nil, err
	}

	ciphertext := make([]byte, len(plaintext))
	stream.XORKeyStream(ciphertext, plaintext)
	return ciphertext, nil
}

func decryptRC4(key, ciphertext []byte) ([]byte, error) {
	stream, err := rc4.NewCipher(key)
	if err != nil {
		return nil, err
	}

	plaintext := make([]byte, len(ciphertext))
	stream.XORKeyStream(plaintext, ciphertext)
	return plaintext, nil
}