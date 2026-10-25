package dto

type OrangTuaRequest struct {
	NIK                string `json:"nik" binding:"omitempty,len=16,numeric"`
	Nama               string `json:"nama"`
	TempatLahir        string `json:"tempat_lahir"`
	TanggalLahir       string `json:"tanggal_lahir" binding:"omitempty,datetime=2006-01-02"`
	Pekerjaan          string `json:"pekerjaan"`
	Instansi           string `json:"instansi"`
	PendidikanTerakhir string `json:"pendidikan_terakhir"`
	NoHP               string `json:"no_hp"`
	Status             string `json:"status" binding:"omitempty,oneof=masih_hidup meninggal"`
}

type AlamatOrtuRequest struct {
	Alamat         string `json:"alamat"`
	RT             string `json:"rt" binding:"omitempty,len=3,numeric"`
	RW             string `json:"rw" binding:"omitempty,len=3,numeric"`
	Kelurahan      string `json:"kelurahan"`
	Kecamatan      string `json:"kecamatan"`
	Provinsi       string `json:"provinsi"`
	Kota           string `json:"kota"`
	KodePos        string `json:"kode_pos" binding:"omitempty,len=5,numeric"`
	NoTeleponRumah string `json:"no_telepon_rumah"`
}

type KeluargaRequest struct {
	Ayah                OrangTuaRequest   `json:"ayah"`
	Ibu                 OrangTuaRequest   `json:"ibu"`
	RangePendapatanOrtu string            `json:"range_pendapatan_ortu"`
	NomorKK             string            `json:"nomor_kk" binding:"omitempty,len=16,numeric"`
	JumlahKakak         int               `json:"jumlah_kakak" binding:"omitempty,gte=0"`
	JumlahAdik          int               `json:"jumlah_adik" binding:"omitempty,gte=0"`
	AlamatOrtu          AlamatOrtuRequest `json:"alamat_ortu"`
}

type BiodataKeluargaResponse struct {
	ID string `json:"id"`
	KeluargaRequest
}
