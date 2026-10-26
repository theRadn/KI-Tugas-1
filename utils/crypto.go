// referensi: https://pkg.go.dev/crypto/cipher#NewCTR

package utils

import (
	"crypto/hmac" "crypto/sha256" "encoding/binary" "fmt"
)

const FormatVersion = "v2"

type Algorithm string

const (
	AlgorithmAES256CTR Algorithm = "AES-256-CTR"
	AlgorithmDESCTR Algorithm = "DES-CTR"
	AlgorithmRC4 Algorithm = "RC4"
)

type EncryptedData struct {
	Version    string    `json:"version"`
	Algorithm  Algorithm `json:"algorithm"`
	IV         []byte    `json:"iv,omitempty"`
	Ciphertext []byte    `json:"ciphertext"`
	Tag        []byte    `json:"tag"`
}

func Encrypt(algorithm Algorithm, key, macKey, plaintext []byte) (EncryptedData, error) {
	var iv, ciphertext []byte
	var err error
	switch algorithm {
	case AlgorithmAES256CTR:
		iv, ciphertext, err = encryptAESCTR(key, plaintext)
	case AlgorithmDESCTR, AlgorithmRC4:
		return EncryptedData{}, fmt.Errorf("encryption algorithm %q is insecure for new data", algorithm)
	default:
		return EncryptedData{}, fmt.Errorf("unsupported encryption algorithm %q", algorithm)
	}
	if err != nil {
		return EncryptedData{}, err
	}
	data := EncryptedData{
		Version:    FormatVersion,
		Algorithm:  algorithm,
		IV:         iv,
		Ciphertext: ciphertext,
	}
	data.Tag = authenticationTag(data, macKey)
	return data, nil
}

func Decrypt(data EncryptedData, key, macKey []byte) ([]byte, error) {
	if data.Version != FormatVersion {
		return nil, fmt.Errorf("different version")
	}
	if data.Ciphertext == nil {
		return nil, fmt.Errorf("no ciphertext to decrypt")
	}
	if len(data.Tag) != sha256.Size || !hmac.Equal(data.Tag, authenticationTag(data, macKey)) {
		return nil, fmt.Errorf("encrypted data failed authentication")
	}

	switch data.Algorithm {
	case AlgorithmAES256CTR:
		return decryptAESCTR(key, data.IV, data.Ciphertext)
	case AlgorithmDESCTR:
		return decryptDESCTR(key, data.IV, data.Ciphertext)
	case AlgorithmRC4:
		return decryptRC4(key, data.Ciphertext)
	}
	return nil, fmt.Errorf("unsupported encryption algorithm %q", data.Algorithm)
}

func authenticationTag(data EncryptedData, key []byte) []byte {
	mac := hmac.New(sha256.New, key)
	fields := [][]byte{[]byte(data.Version), []byte(data.Algorithm), data.IV, data.Ciphertext}
	var length [8]byte
	for _, field := range fields {
		binary.BigEndian.PutUint64(length[:], uint64(len(field)))
		_, _ = mac.Write(length[:])
		_, _ = mac.Write(field)
	}
	return mac.Sum(nil)
}
