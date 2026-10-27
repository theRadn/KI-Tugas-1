package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/hkdf"
)

const (
	SessionCryptoVersion = "v3"
	AES128KeySize        = 16
	HMACKeySize          = 32
	SessionNonceSize     = 16
	CTRIVSize            = aes.BlockSize
)

type SessionKeys struct {
	AESKey []byte
	MACKey []byte
}

type EncryptedEnvelope struct {
	Version    string `json:"version"`
	SessionID  string `json:"session_id"`
	RequestID  string `json:"request_id"`
	Direction  string `json:"direction"`
	IV         []byte `json:"iv"`
	Ciphertext []byte `json:"ciphertext"`
	Tag        []byte `json:"tag"`
}

func GenerateEphemeralX25519() (*ecdh.PrivateKey, []byte, error) {
	privateKey, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	return privateKey, privateKey.PublicKey().Bytes(), nil
}

func ComputeX25519SharedSecret(
	privateKey *ecdh.PrivateKey,
	peerPublicKey []byte,
) ([]byte, error) {
	if privateKey == nil {
		return nil, errors.New("missing private key")
	}

	publicKey, err := ecdh.X25519().NewPublicKey(peerPublicKey)
	if err != nil {
		return nil, fmt.Errorf("invalid peer public key: %w", err)
	}

	sharedSecret, err := privateKey.ECDH(publicKey)
	if err != nil {
		return nil, fmt.Errorf("key exchange failed: %w", err)
	}

	return sharedSecret, nil
}

func DeriveSessionKeys(
	sharedSecret, salt, info []byte,
) (SessionKeys, error) {
	if len(sharedSecret) == 0 {
		return SessionKeys{}, errors.New("empty shared secret")
	}

	reader := hkdf.New(sha256.New, sharedSecret, salt, info)

	material := make([]byte, AES128KeySize+HMACKeySize)
	if _, err := io.ReadFull(reader, material); err != nil {
		return SessionKeys{}, fmt.Errorf("derive session keys: %w", err)
	}

	keys := SessionKeys{
		AESKey: append([]byte(nil), material[:AES128KeySize]...),
		MACKey: append([]byte(nil), material[AES128KeySize:]...),
	}

	clear(material)

	return keys, nil
}

func EncryptSessionPayload(
	keys SessionKeys,
	sessionID, requestID, direction string,
	plaintext []byte,
) (EncryptedEnvelope, error) {
	if err := validateSessionKeys(keys); err != nil {
		return EncryptedEnvelope{}, err
	}

	if sessionID == "" || requestID == "" {
		return EncryptedEnvelope{}, errors.New("missing session or request ID")
	}

	if direction != "request" && direction != "response" {
		return EncryptedEnvelope{}, errors.New("invalid message direction")
	}

	block, err := aes.NewCipher(keys.AESKey)
	if err != nil {
		return EncryptedEnvelope{}, err
	}

	iv := make([]byte, CTRIVSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return EncryptedEnvelope{}, fmt.Errorf("generate IV: %w", err)
	}

	ciphertext := make([]byte, len(plaintext))
	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(ciphertext, plaintext)

	envelope := EncryptedEnvelope{
		Version:    SessionCryptoVersion,
		SessionID:  sessionID,
		RequestID:  requestID,
		Direction:  direction,
		IV:         iv,
		Ciphertext: ciphertext,
	}

	envelope.Tag = sessionAuthenticationTag(envelope, keys.MACKey)

	return envelope, nil
}

func DecryptSessionPayload(
	keys SessionKeys,
	expectedSessionID, expectedDirection string,
	envelope EncryptedEnvelope,
) ([]byte, error) {
	if err := validateSessionKeys(keys); err != nil {
		return nil, err
	}

	if envelope.Version != SessionCryptoVersion {
		return nil, errors.New("unsupported encryption version")
	}

	if envelope.SessionID == "" ||
		envelope.SessionID != expectedSessionID {
		return nil, errors.New("invalid session")
	}

	if envelope.RequestID == "" {
		return nil, errors.New("missing request ID")
	}

	if envelope.Direction != expectedDirection {
		return nil, errors.New("invalid message direction")
	}

	if len(envelope.IV) != CTRIVSize {
		return nil, errors.New("invalid IV length")
	}

	if envelope.Ciphertext == nil {
		return nil, errors.New("missing ciphertext")
	}

	if len(envelope.Tag) != sha256.Size {
		return nil, errors.New("invalid authentication tag")
	}

	expectedTag := sessionAuthenticationTag(envelope, keys.MACKey)
	if !hmac.Equal(envelope.Tag, expectedTag) {
		return nil, errors.New("message authentication failed")
	}

	block, err := aes.NewCipher(keys.AESKey)
	if err != nil {
		return nil, err
	}

	plaintext := make([]byte, len(envelope.Ciphertext))
	stream := cipher.NewCTR(block, envelope.IV)
	stream.XORKeyStream(plaintext, envelope.Ciphertext)

	return plaintext, nil
}

func sessionAuthenticationTag(
	envelope EncryptedEnvelope,
	macKey []byte,
) []byte {
	mac := hmac.New(sha256.New, macKey)

	fields := [][]byte{
		[]byte(envelope.Version),
		[]byte(envelope.SessionID),
		[]byte(envelope.RequestID),
		[]byte(envelope.Direction),
		envelope.IV,
		envelope.Ciphertext,
	}

	var length [8]byte

	for _, field := range fields {
		binary.BigEndian.PutUint64(length[:], uint64(len(field)))
		_, _ = mac.Write(length[:])
		_, _ = mac.Write(field)
	}

	return mac.Sum(nil)
}

func validateSessionKeys(keys SessionKeys) error {
	if len(keys.AESKey) != AES128KeySize {
		return errors.New("AES-128 key must be 16 bytes")
	}

	if len(keys.MACKey) != HMACKeySize {
		return errors.New("HMAC key must be 32 bytes")
	}

	return nil
}

func GenerateSessionID() (string, error) {
	id := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, id); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", id), nil
}

func GenerateRequestID() (string, error) {
	id := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, id); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", id), nil
}

func SessionDerivationInfo(sessionID string) []byte {
	return []byte("my-go-api/" + SessionCryptoVersion + "/" + sessionID)
}

func ValidateSessionID(sessionID string) bool {
	return len(sessionID) == 64 && strings.Trim(sessionID, "0123456789abcdef") == ""
}
