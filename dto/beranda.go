package dto

import "time"

type StatusBagian struct { // status biodata per bagian
	Bagian       string     `json:"bagian"`
	Tersimpan    bool       `json:"tersimpan"`
	DisimpanPada *time.Time `json:"disimpan_pada,omitempty"`
}

type BerandaResponse struct { // ini yang dikirim ke FE untuk halaman beranda
	NamaPengguna    string         `json:"nama_pengguna"`
	NoPendaftaran   string         `json:"no_pendaftaran,omitempty"`
	KelengkapanData []StatusBagian `json:"kelengkapan_data"`
	JumlahBerkas    int64          `json:"jumlah_berkas"`
	TotalUkuranByte int64          `json:"total_ukuran_byte"`
}