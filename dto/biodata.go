package dto

type NilaiTKAInput struct {
	MataPelajaran string  `json:"mata_pelajaran" binding:"required"`
	Nilai         float64 `json:"nilai" binding:"gte=0,lte=100"`
}

type PendidikanRequest struct {
	ProvinsiSekolah string          `json:"provinsi_sekolah"`
	KotaSekolah     string          `json:"kota_sekolah"`
	NamaSekolah     string          `json:"nama_sekolah"`
	Jurusan         string          `json:"jurusan"`
	NISN            string          `json:"nisn" binding:"omitempty,len=10,numeric"`
	TahunMasukSMTA  int             `json:"tahun_masuk_smta" binding:"omitempty,gte=1900,lte=3000"`
	TahunLulusSMTA  int             `json:"tahun_lulus_smta" binding:"omitempty,gte=1900,lte=3000"`
	NilaiTKA        []NilaiTKAInput `json:"nilai_tka" binding:"dive"`
}

type DataDiriRequest struct {
	Foto 			 string `json:"foto,omitempty"`
	NamaIjazah       string `json:"nama_ijazah"`
	NamaKTM          string `json:"nama_ktm" binding:"omitempty,max=20"`
	UkuranAlmamater  string `json:"ukuran_almamater"`
	ProgramStudi     string `json:"program_studi"`
	JalurMasuk       string `json:"jalur_masuk"`
	PeriodeMasuk     string `json:"periode_masuk"`
	Kewarganegaraan  string `json:"kewarganegaraan"`
	NIK              string `json:"nik" binding:"omitempty,len=16,numeric"`
	NoPaspor         string `json:"no_paspor,omitempty"`
	StatusPernikahan string `json:"status_pernikahan"`
	JenisKelamin     string `json:"jenis_kelamin" binding:"omitempty,oneof=laki_laki perempuan"`
	Agama            string `json:"agama"`
	TanggalLahir     string `json:"tanggal_lahir" binding:"omitempty,datetime=2006-01-02"`
	NegaraLahir      string `json:"negara_lahir"`
	ProvinsiLahir    string `json:"provinsi_lahir"`
	KotaLahir        string `json:"kota_lahir"`
	TempatLahir      string `json:"tempat_lahir"`
	AlamatKTP        string `json:"alamat_ktp"`
	NegaraAsal       string `json:"negara_asal"`
	ProvinsiAsal     string `json:"provinsi_asal"`
	KotaAsal         string `json:"kota_asal"`
	KodePos          string `json:"kode_pos" binding:"omitempty,len=5,numeric"`
	AlamatDomisili   string `json:"alamat_domisili"`
	NegaraDomisili   string `json:"negara_domisili"`
	ProvinsiDomisili string `json:"provinsi_domisili"`
	KotaDomisili     string `json:"kota_domisili"`
	Email            string `json:"email" binding:"omitempty,email"`
	NoHP             string `json:"no_hp"`
	NoKIPK           string `json:"no_kip_k"`
	NoBPJS           string `json:"no_bpjs"`
	SumberInformasi  string `json:"sumber_informasi" binding:"omitempty,oneof=website media_sosial teman keluarga sekolah"`
	DaftarAsrama     bool   `json:"daftar_asrama"`
}

type RumahRequest struct {
	Kepemilikan      string `json:"kepemilikan"`
	TahunPerolehan   int    `json:"tahun_perolehan" binding:"omitempty,gte=1000,lte=9999"`
	SumberListrik    string `json:"sumber_listrik"`
	LuasTanah        int    `json:"luas_tanah" binding:"omitempty,gte=0"`
	LuasBangunan     int    `json:"luas_bangunan" binding:"omitempty,gte=0"`
	NJOP             int64  `json:"njop" binding:"omitempty,gte=0"`
	MCK              string `json:"mck"`
	SumberAirMinum   string `json:"sumber_air_minum"`
	JarakPusatKotaKM int    `json:"jarak_pusat_kota_km" binding:"omitempty,gte=0"`
	JumlahPenghuni   int    `json:"jumlah_penghuni" binding:"gte=1"`
}

type CatatanKhususRequest struct {
	RencanaTempatTinggal string `json:"rencana_tempat_tinggal"`
	AdaDukunganKeluarga  bool   `json:"ada_dukungan_keluarga"`
	TransportasiDariAsal string `json:"transportasi_dari_asal"`
	TransportasiHarian   string `json:"transportasi_harian"`
	GolonganDarah        string `json:"golongan_darah" binding:"omitempty,oneof=A B AB O"`
	Rhesus               string `json:"rhesus" binding:"omitempty,oneof=positif negatif"`
	KondisiKesehatan     string `json:"kondisi_kesehatan"`
	Disabilitas          bool   `json:"disabilitas"`
}

type PendukungRequest struct {
	PekerjaanAyah   string `json:"pekerjaan_ayah"`
	PekerjaanIbu    string `json:"pekerjaan_ibu"`
	PenghasilanAyah int64  `json:"penghasilan_ayah" binding:"omitempty,gte=0"`
	PenghasilanIbu  int64  `json:"penghasilan_ibu" binding:"omitempty,gte=0"`
	TanggunganAnak  int    `json:"tanggungan_anak" binding:"omitempty,gte=0"`
	TagihanListrik  int64  `json:"tagihan_listrik" binding:"omitempty,gte=0"`
	DayaListrik     string `json:"daya_listrik"`
	PajakRodaDua    int64  `json:"pajak_roda_dua" binding:"omitempty,gte=0"`
	PajakRodaEmpat  int64  `json:"pajak_roda_empat" binding:"omitempty,gte=0"`
	TotalPBB        int64  `json:"total_pbb" binding:"omitempty,gte=0"`
}

type BiodataRequest struct {
	Pendidikan    PendidikanRequest    `json:"pendidikan"`
	DataDiri      DataDiriRequest      `json:"data_diri"`
	Keluarga      KeluargaRequest      `json:"keluarga,omitempty"`
	Rumah         RumahRequest         `json:"rumah"`
	CatatanKhusus CatatanKhususRequest `json:"catatan_khusus"`
	Pendukung     PendukungRequest     `json:"pendukung"`
}

type BiodataResponse struct {
	ID string `json:"id"`
	BiodataRequest
	NoPendaftaran string            `json:"no_pendaftaran,omitempty"`
}
