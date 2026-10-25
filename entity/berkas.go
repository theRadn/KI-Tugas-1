
package entity

import (
	"time"
	"github.com/google/uuid"
)

type Berkas struct {
	BaseModel
	BiodataID uuid.UUID `gorm:"type:text;index;not null" json:"biodata_id"`

	KodeBerkas string `gorm:"type:varchar(80);not null;uniqueIndex:idx_biodata_kode_berkas" json:"kode_berkas"`
	NamaAsli   string `gorm:"type:varchar(255);not null" json:"nama_asli"`
	UkuranByte int64  `gorm:"type:bigint;not null" json:"ukuran_byte"`
	TipeMIME   string `gorm:"type:varchar(100);not null" json:"tipe_mime"`
	Status     string `gorm:"type:varchar(20);not null;default:mengunggah" json:"status"`
	FilePath   string `gorm:"type:text" json:"-"`

	Enkripsi []EnkripsiBerkas `gorm:"foreignKey:BerkasID;references:ID" json:"enkripsi,omitempty"`

	DiunggahPada *time.Time `gorm:"type:timestamp" json:"diunggah_pada"`
	DibuatPada   time.Time  `gorm:"autoCreateTime" json:"dibuat_pada"`
	DiubahPada   time.Time  `gorm:"autoUpdateTime" json:"diubah_pada"`
}

func (Berkas) TableName() string { 
	return "berkas" 
}

type EnkripsiBerkas struct {
	BaseModel
	BerkasID             uuid.UUID `gorm:"type:text;index;not null" json:"berkas_id"`
	Algoritma            string    `gorm:"type:varchar(20);not null" json:"algoritma"`
	Mode                 string    `gorm:"type:varchar(50);not null" json:"mode"`
	IV                   *string   `gorm:"type:text" json:"iv,omitempty"`
	Nonce                *string   `gorm:"type:text" json:"nonce,omitempty"`
	CiphertextPath       string    `gorm:"type:text;not null" json:"ciphertext_path"`
	UkuranCiphertextByte int64     `gorm:"type:bigint;not null" json:"ukuran_ciphertext_byte"`
}

func (EnkripsiBerkas) TableName() string { 
	return "enkripsi_berkas" 
}
