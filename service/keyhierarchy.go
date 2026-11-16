package service

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/theRadn/KI-Tugas-1/entity"
	"github.com/theRadn/KI-Tugas-1/utils"
	"gorm.io/gorm"
)

func unwrapUserKEK(db *gorm.DB, userID uuid.UUID) ([]byte, error) {
	var user entity.User
	if err := db.First(&user, "id = ?", userID).Error; err != nil {
		return nil, err
	}
	if len(user.KEKWrapped) == 0 {
		return nil, errors.New("user KEK belum tersedia")
	}

	masterKEK, err := utils.MasterKEK()
	if err != nil {
		return nil, err
	}
	wrapped, err := utils.UnmarshalWrappedKey(user.KEKWrapped)
	if err != nil {
		return nil, err
	}
	return utils.UnwrapKey(
		masterKEK,
		wrapped,
		[]byte("user:"+userID.String()+":kek"),
	)
}

func dekSize(algorithm string) (int, error) {
	switch algorithm {
	case "AES":
		return 32, nil
	case "DES":
		return 8, nil
	case "RC4":
		return 32, nil
	default:
		return 0, fmt.Errorf("unsupported encryption algorithm %q", algorithm)
	}
}

func dekAAD(fileID uuid.UUID, purpose, algorithm string) []byte {
	return []byte(fmt.Sprintf(
		"file:%s:%s:%s-dek",
		fileID,
		purpose,
		algorithm,
	))
}

func deriveFileMACKey(dek []byte, aad []byte) ([]byte, error) {
	return utils.DeriveMACKey(dek, append([]byte("file-hmac:"), aad...))
}
