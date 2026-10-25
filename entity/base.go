package entity

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BaseModel struct {
	ID uuid.UUID `gorm:"type:text;primaryKey;not null" json:"id"`
}

func (base *BaseModel) BeforeCreate(tx *gorm.DB) (err error) {
	if base.ID == uuid.Nil {
		newID, err := uuid.NewV7()
		if err != nil {
			return err
		}
		base.ID = newID
	}
	return nil
}

type Example struct {
	BaseModel
	Test string `gorm:"type:varchar(255);not null"`
}
