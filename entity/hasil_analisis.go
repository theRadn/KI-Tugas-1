package entity

import (
	"time"
	"github.com/google/uuid"
)

type HasilAnalisis struct {
	BaseModel
	UserID          uuid.UUID                `gorm:"type:text;index;not null" json:"user_id"`
	BerkasID        uuid.UUID                `gorm:"type:text;index;not null" json:"berkas_id"`
	JumlahPercobaan int                      `gorm:"type:int;not null" json:"jumlah_percobaan"`
	Hasil           []HasilAnalisisAlgoritma `gorm:"foreignKey:HasilAnalisisID;references:ID" json:"hasil"`
	DibuatPada      time.Time                `gorm:"autoCreateTime" json:"dibuat_pada"`
}

func (HasilAnalisis) TableName() string { 
	return "hasil_analisis" 
}

type HasilAnalisisAlgoritma struct {
	BaseModel
	HasilAnalisisID      uuid.UUID `gorm:"type:text;index;not null" json:"hasil_analisis_id"`
	Algoritma            string    `gorm:"type:varchar(20);not null" json:"algoritma"`
	Mode                 string    `gorm:"type:varchar(50);not null" json:"mode"`
	RataRataMS           float64   `gorm:"type:decimal(12,4);not null" json:"rata_rata_ms"`
	MedianMS             float64   `gorm:"type:decimal(12,4);not null" json:"median_ms"`
	SimpanganBakuMS      float64   `gorm:"type:decimal(12,4);not null" json:"simpangan_baku_ms"`
	MinMS                float64   `gorm:"type:decimal(12,4);not null" json:"min_ms"`
	MaksMS               float64   `gorm:"type:decimal(12,4);not null" json:"maks_ms"`
	UkuranCiphertextByte int64     `gorm:"type:bigint;not null" json:"ukuran_ciphertext_byte"`
	SelisihUkuranByte    int64     `gorm:"type:bigint;not null" json:"selisih_ukuran_byte"`
	MBPerDetik           float64   `gorm:"type:decimal(12,4);not null" json:"mb_per_detik"`
}

func (HasilAnalisisAlgoritma) TableName() string { 
	return "hasil_analisis_algoritma" 
}
