package dto

type UjiDekripsiRequest struct {
	BerkasID        string `json:"berkas_id" binding:"required"`
	JumlahPercobaan int    `json:"jumlah_percobaan" binding:"required,min=1"`
}

type HasilAnalisisResponse struct {
	Algoritma            string  `json:"algoritma"`
	Mode                 string  `json:"mode"`
	RataRataMs            float64 `json:"rata_rata_ms"`
	MedianMs              float64 `json:"median_ms"`
	SimpanganBakuMs       float64 `json:"simpangan_baku_ms"`
	MinMs                 float64 `json:"min_ms"`
	MaksMs                float64 `json:"maks_ms"`
	UkuranCiphertextByte  int64   `json:"ukuran_ciphertext_byte"`
	SelisihUkuranByte     int64   `json:"selisih_ukuran_byte"`
	Mbps                  float64 `json:"mb_per_detik"`
}

type UjiDekripsiResponse struct {
	BerkasID string                    `json:"berkas_id"`
	Hasil    []HasilAnalisisResponse   `json:"hasil"`
}