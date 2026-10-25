package dto

import "time"

type BerkasResponse struct {
	ID           string `json:"id"`
	KodeBerkas   string `json:"kode_berkas"`
	NamaAsli     string `json:"nama_asli"`
	UkuranByte   int64  `json:"ukuran_byte"`
	TipeMIME     string `json:"tipe_mime"`
	Status       string `json:"status"`
	DiunggahPada *time.Time `json:"diunggah_pada"`
}

type EnkripsiBerkasResponse struct {
	Algoritma            string `json:"algoritma"`
	Mode                 string `json:"mode"`
	UkuranCiphertextByte int64  `json:"ukuran_ciphertext_byte"`
}

type BerkasListResponse struct {
	Berkas          []BerkasResponse `json:"berkas"`
	Jumlah          int              `json:"jumlah"`
	TotalUkuranByte int64            `json:"total_ukuran_byte"`
}

type UnggahBerkasResponse struct {
	BerkasResponse
	Enkripsi []EnkripsiBerkasResponse `json:"enkripsi,omitempty"`
}

type UnduhBerkasRequest struct {
	Algoritma string `form:"algoritma" json:"algoritma" binding:"required,oneof=AES DES RC4"`
}

type CiphertextResponse struct {
	BerkasID   string `json:"berkas_id"`
	Algoritma  string `json:"algoritma"`
	Ciphertext string `json:"ciphertext"`
	Encoding   string `json:"encoding"`
}
