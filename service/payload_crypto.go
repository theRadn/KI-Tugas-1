package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/theRadn/KI-Tugas-1/utils"
	"gorm.io/gorm"
)

func packEncryptedPayload(
	db *gorm.DB,
	plaintext []byte,
	userID, fileID uuid.UUID,
	purpose string,
) ([]byte, error) {
	if db == nil {
		return nil, errors.New("encryption database is not configured")
	}

	userKEK, err := unwrapUserKEK(db, userID)
	if err != nil {
		return nil, err
	}

	versions := make(map[string]storedVideoCipher)
	for _, c := range videoCiphers() {
		dek, err := utils.GenerateKey(c.size)
		if err != nil {
			return nil, err
		}

		aad := dekAAD(fileID, purpose, c.name)
		macKey, err := deriveFileMACKey(dek, aad)
		if err != nil {
			return nil, err
		}

		encrypted, err := utils.Encrypt(c.algorithm, dek, macKey, plaintext)
		if err != nil {
			return nil, err
		}

		wrapped, err := utils.WrapKey(userKEK, dek, aad)
		if err != nil {
			return nil, err
		}
		wrappedBytes, err := utils.MarshalWrappedKey(wrapped)
		if err != nil {
			return nil, err
		}

		versions[c.name] = storedVideoCipher{
			EncryptedData: encrypted,
			WrappedDEK:    wrappedBytes,
		}
	}

	return json.Marshal(versions)
}

func unpackEncryptedPayload(
	db *gorm.DB,
	data []byte,
	userID, fileID uuid.UUID,
	purpose, algorithm string,
) ([]byte, error) {
	var versions map[string]storedVideoCipher
	if err := json.Unmarshal(data, &versions); err != nil {
		return nil, err
	}

	encrypted, ok := versions[algorithm]
	if !ok {
		return nil, ErrInvalidVideoChunk
	}

	var selected *videoCipher
	for _, c := range videoCiphers() {
		if c.name == algorithm {
			selected = &c
			break
		}
	}
	if selected == nil || encrypted.Algorithm != selected.algorithm {
		return nil, ErrInvalidVideoChunk
	}

	if len(encrypted.WrappedDEK) == 0 {
		// Legacy data was encrypted directly with the environment key.
		envName := map[string]string{
			"AES": "DATABASE_SECRET_AES256",
			"DES": "DATABASE_SECRET_DES",
			"RC4": "DATABASE_SECRET_RC4",
		}[algorithm]
		legacyKey := []byte(os.Getenv(envName))
		if len(legacyKey) == 0 {
			return nil, errors.New("legacy encryption key is not configured")
		}
		return utils.Decrypt(encrypted.EncryptedData, legacyKey, legacyKey)
	}

	if db == nil {
		return nil, errors.New("encryption database is not configured")
	}

	userKEK, err := unwrapUserKEK(db, userID)
	if err != nil {
		return nil, err
	}

	aad := dekAAD(fileID, purpose, algorithm)
	wrapped, err := utils.UnmarshalWrappedKey(encrypted.WrappedDEK)
	if err != nil {
		return nil, err
	}
	dek, err := utils.UnwrapKey(userKEK, wrapped, aad)
	if err != nil {
		return nil, err
	}

	macKey, err := deriveFileMACKey(dek, aad)
	if err != nil {
		return nil, err
	}

	return utils.Decrypt(encrypted.EncryptedData, dek, macKey)
}

// Keep the video service methods as wrappers so existing video call sites work.
func (s *VideoService) packVideoData(
	plaintext []byte,
	userID, fileID uuid.UUID,
	purpose string,
) ([]byte, error) {
	return packEncryptedPayload(s.db, plaintext, userID, fileID, purpose)
}

func (s *VideoService) unpackVideoData(
	data []byte,
	userID, fileID uuid.UUID,
	purpose, algorithm string,
) ([]byte, error) {
	return unpackEncryptedPayload(
		s.db, data, userID, fileID, purpose, algorithm,
	)
}

// Add these wrappers so DocumentService can use the same helpers.
func (s *DocumentService) packVideoData(
	plaintext []byte,
	userID, fileID uuid.UUID,
	purpose string,
) ([]byte, error) {
	return packEncryptedPayload(s.db, plaintext, userID, fileID, purpose)
}

func (s *DocumentService) unpackVideoData(
	data []byte,
	userID, fileID uuid.UUID,
	purpose, algorithm string,
) ([]byte, error) {
	return unpackEncryptedPayload(
		s.db, data, userID, fileID, purpose, algorithm,
	)
}

var _ = fmt.Sprintf
