package entity

import "github.com/google/uuid"

type EncryptionKeySet struct {
	BaseModel
	UserID       uuid.UUID `gorm:"type:text;not null;uniqueIndex:idx_encryption_key_set_scope"`
	Scope        string    `gorm:"type:varchar(50);not null;uniqueIndex:idx_encryption_key_set_scope"`
	WrappedAES   []byte    `gorm:"type:blob;not null"`
	WrappedDES   []byte    `gorm:"type:blob;not null"`
	WrappedRC4   []byte    `gorm:"type:blob;not null"`
	KeyVersion   string    `gorm:"type:varchar(20);not null;default:v1"`
}

func (EncryptionKeySet) TableName() string {
	return "encryption_key_sets"
}
