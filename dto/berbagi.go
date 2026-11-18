package dto

import "time"

type BuatPermintaanBerbagiRequest struct {
	NamaPenggunaPemilik string   `json:"nama_pengguna_pemilik" binding:"required"`
	Cakupan             []string `json:"cakupan" binding:"required,min=1,dive,required"`
	BerkasIDs           []string `json:"berkas_ids"`
	Pesan               string   `json:"pesan"`
	DurasiHari          int      `json:"durasi_hari" binding:"required,min=1,max=30"`
}

type JawabPermintaanBerbagiRequest struct {
	Setuju     bool     `json:"setuju"`
	Cakupan    []string `json:"cakupan"`
	BerkasIDs  []string `json:"berkas_ids"`
	DurasiHari int      `json:"durasi_hari" binding:"min=1,max=30"`
	Pesan      string   `json:"pesan"`
}

type PermintaanBerbagiResponse struct {
	ID               string     `json:"id"`
	PemintaID        string     `json:"peminta_id"`
	PemintaNama      string     `json:"peminta_nama"`
	PemilikID        string     `json:"pemilik_id"`
	PemilikNama      string     `json:"pemilik_nama"`
	Status           string     `json:"status"`
	CakupanDiminta   []string   `json:"cakupan_diminta"`
	CakupanDisetujui []string   `json:"cakupan_disetujui"`
	BerkasDiminta    []string   `json:"berkas_diminta"`
	BerkasDisetujui  []string   `json:"berkas_disetujui"`
	PesanPeminta     string     `json:"pesan_peminta"`
	PesanPemilik     string     `json:"pesan_pemilik"`
	BerlakuSampai    *time.Time `json:"berlaku_sampai"`
	DibuatPada       time.Time  `json:"dibuat_pada"`
	DiubahPada       time.Time  `json:"diubah_pada"`
	Aktif            bool       `json:"aktif"`
}

type BerbagiChunkResponse struct {
	BerkasID    string `json:"berkas_id"`
	Index       int    `json:"index"`
	TipeMIME    string `json:"tipe_mime"`
	UkuranByte  int    `json:"ukuran_byte"`
	IsiBase64   string `json:"isi_base64"`
}
