package entity

import (
	"time"

	"github.com/google/uuid"
)

type Biodata struct {
	BaseModel
	UserID uuid.UUID `gorm:"type:text;uniqueIndex;not null" json:"user_id"`

	// Data diri
	NoPendaftaran    string     `gorm:"type:varchar(50);uniqueIndex" json:"no_pendaftaran"`
	FotoPath         *string    `gorm:"type:text" json:"foto"`
	NamaIjazah       string     `gorm:"type:varchar(255)" json:"nama_ijazah"`
	NamaKTM          string     `gorm:"type:varchar(20)" json:"nama_ktm"`
	UkuranAlmamater  string     `gorm:"type:varchar(20)" json:"ukuran_almamater"`
	ProgramStudi     string     `gorm:"type:varchar(100)" json:"program_studi"`
	JalurMasuk       string     `gorm:"type:varchar(100)" json:"jalur_masuk"`
	PeriodeMasuk     string     `gorm:"type:varchar(100)" json:"periode_masuk"`
	Kewarganegaraan  string     `gorm:"type:varchar(100)" json:"kewarganegaraan"`
	NIK              string     `gorm:"type:varchar(16)" json:"nik"`
	NoPaspor         *string    `gorm:"type:varchar(50)" json:"no_paspor"`
	StatusPernikahan string     `gorm:"type:varchar(50)" json:"status_pernikahan"`
	JenisKelamin     string     `gorm:"type:varchar(20)" json:"jenis_kelamin"`
	Agama            string     `gorm:"type:varchar(50)" json:"agama"`
	TanggalLahir     *time.Time `gorm:"type:date" json:"tanggal_lahir"`
	NegaraLahir      string     `gorm:"type:varchar(100)" json:"negara_lahir"`
	ProvinsiLahir    string     `gorm:"type:varchar(100)" json:"provinsi_lahir"`
	KotaLahir        string     `gorm:"type:varchar(100)" json:"kota_lahir"`
	TempatLahir      string     `gorm:"type:varchar(100)" json:"tempat_lahir"`

	AlamatKTP        string `gorm:"type:text" json:"alamat_ktp"`
	NegaraAsal       string `gorm:"type:varchar(100)" json:"negara_asal"`
	ProvinsiAsal     string `gorm:"type:varchar(100)" json:"provinsi_asal"`
	KotaAsal         string `gorm:"type:varchar(100)" json:"kota_asal"`
	KodePos          string `gorm:"type:varchar(5)" json:"kode_pos"`
	AlamatDomisili   string `gorm:"type:text" json:"alamat_domisili"`
	NegaraDomisili   string `gorm:"type:varchar(100)" json:"negara_domisili"`
	ProvinsiDomisili string `gorm:"type:varchar(100)" json:"provinsi_domisili"`
	KotaDomisili     string `gorm:"type:varchar(100)" json:"kota_domisili"`
	Email            string `gorm:"type:varchar(255)" json:"email"`
	NoHP             string `gorm:"type:varchar(30)" json:"no_hp"`
	NoKIPK           string `gorm:"type:varchar(50)" json:"no_kip_k"`
	NoBPJS           string `gorm:"type:varchar(50)" json:"no_bpjs"`
	SumberInformasi  string `gorm:"type:varchar(50)" json:"sumber_informasi"`
	DaftarAsrama     bool   `gorm:"not null;default:false" json:"daftar_asrama"`

	Pendidikan       Pendidikan       `gorm:"foreignKey:BiodataID;references:ID" json:"pendidikan"`
	Keluarga         BiodataKeluarga  `gorm:"foreignKey:BiodataID;references:ID" json:"keluarga"`
	Rumah            Rumah            `gorm:"foreignKey:BiodataID;references:ID" json:"rumah"`
	CatatanKhusus    CatatanKhusus    `gorm:"foreignKey:BiodataID;references:ID" json:"catatan_khusus"`
	PendukungEkonomi PendukungEkonomi `gorm:"foreignKey:BiodataID;references:ID" json:"pendukung"`
	Dokumen          []Dokumen        `gorm:"foreignKey:BiodataID;references:ID" json:"dokumen,omitempty"`
	Berkas           []Berkas         `gorm:"foreignKey:BiodataID;references:ID" json:"berkas,omitempty"`

	DibuatPada time.Time `gorm:"autoCreateTime" json:"dibuat_pada"`
	DiubahPada time.Time `gorm:"autoUpdateTime" json:"diubah_pada"`
}

func (Biodata) TableName() string { 
	return "biodata" 
}

type Pendidikan struct {
	BaseModel
	BiodataID       uuid.UUID  `gorm:"type:text;uniqueIndex;not null" json:"biodata_id"`
	ProvinsiSekolah string     `gorm:"type:varchar(100)" json:"provinsi_sekolah"`
	KotaSekolah     string     `gorm:"type:varchar(100)" json:"kota_sekolah"`
	NamaSekolah     string     `gorm:"type:varchar(255)" json:"nama_sekolah"`
	Jurusan         string     `gorm:"type:varchar(100)" json:"jurusan"`
	NISN            string     `gorm:"type:varchar(10)" json:"nisn"`
	TahunMasukSMTA  int        `gorm:"type:int" json:"tahun_masuk_smta"`
	TahunLulusSMTA  int        `gorm:"type:int" json:"tahun_lulus_smta"`
	NilaiTKA        []NilaiTKA `gorm:"foreignKey:PendidikanID;references:ID" json:"nilai_tka,omitempty"`
	DibuatPada      time.Time  `gorm:"autoCreateTime" json:"dibuat_pada"`
	DiubahPada      time.Time  `gorm:"autoUpdateTime" json:"diubah_pada"`
}

func (Pendidikan) TableName() string { 
	return "pendidikan" 
}

type NilaiTKA struct {
	BaseModel
	PendidikanID  uuid.UUID `gorm:"type:text;index;not null" json:"pendidikan_id"`
	MataPelajaran string    `gorm:"type:varchar(100);not null" json:"mata_pelajaran"`
	Nilai         float64   `gorm:"type:decimal(4,1);not null" json:"nilai"`
}

func (NilaiTKA) TableName() string { 
	return "nilai_tka" 
}

type Rumah struct {
	BaseModel
	BiodataID        uuid.UUID `gorm:"type:text;uniqueIndex;not null" json:"biodata_id"`
	Kepemilikan      string    `gorm:"type:varchar(100)" json:"kepemilikan"`
	TahunPerolehan   int       `gorm:"type:int" json:"tahun_perolehan"`
	SumberListrik    string    `gorm:"type:varchar(100)" json:"sumber_listrik"`
	LuasTanah        int       `gorm:"type:int" json:"luas_tanah"`
	LuasBangunan     int       `gorm:"type:int" json:"luas_bangunan"`
	NJOP             int64     `gorm:"type:bigint" json:"njop"`
	MCK              string    `gorm:"type:varchar(100)" json:"mck"`
	SumberAirMinum   string    `gorm:"type:varchar(100)" json:"sumber_air_minum"`
	JarakPusatKotaKM int       `gorm:"type:int" json:"jarak_pusat_kota_km"`
	JumlahPenghuni   int       `gorm:"type:int" json:"jumlah_penghuni"`
	DibuatPada       time.Time `gorm:"autoCreateTime" json:"dibuat_pada"`
	DiubahPada       time.Time `gorm:"autoUpdateTime" json:"diubah_pada"`
}

func (Rumah) TableName() string { 
	return "rumah" 
}

type CatatanKhusus struct {
	BaseModel
	BiodataID            uuid.UUID `gorm:"type:text;uniqueIndex;not null" json:"biodata_id"`
	RencanaTempatTinggal string    `gorm:"type:varchar(100)" json:"rencana_tempat_tinggal"`
	AdaDukunganKeluarga  bool      `gorm:"not null;default:false" json:"ada_dukungan_keluarga"`
	TransportasiDariAsal string    `gorm:"type:varchar(100)" json:"transportasi_dari_asal"`
	TransportasiHarian   string    `gorm:"type:varchar(100)" json:"transportasi_harian"`
	GolonganDarah        string    `gorm:"type:varchar(2)" json:"golongan_darah"`
	Rhesus               string    `gorm:"type:varchar(10)" json:"rhesus"`
	KondisiKesehatan     string    `gorm:"type:text" json:"kondisi_kesehatan"`
	Disabilitas          bool      `gorm:"not null;default:false" json:"disabilitas"`
	DibuatPada           time.Time `gorm:"autoCreateTime" json:"dibuat_pada"`
	DiubahPada           time.Time `gorm:"autoUpdateTime" json:"diubah_pada"`
}

func (CatatanKhusus) TableName() string { 
	return "catatan_khusus" 
}

type PendukungEkonomi struct {
	BaseModel
	BiodataID       uuid.UUID `gorm:"type:text;uniqueIndex;not null" json:"biodata_id"`
	PekerjaanAyah   string    `gorm:"type:varchar(255)" json:"pekerjaan_ayah"`
	PekerjaanIbu    string    `gorm:"type:varchar(255)" json:"pekerjaan_ibu"`
	PenghasilanAyah int64     `gorm:"type:bigint;not null;default:0" json:"penghasilan_ayah"`
	PenghasilanIbu  int64     `gorm:"type:bigint;not null;default:0" json:"penghasilan_ibu"`
	TanggunganAnak  int       `gorm:"type:int;not null;default:0" json:"tanggungan_anak"`
	TagihanListrik  int64     `gorm:"type:bigint;not null;default:0" json:"tagihan_listrik"`
	DayaListrik     string    `gorm:"type:varchar(50)" json:"daya_listrik"`
	PajakRodaDua    int64     `gorm:"type:bigint;not null;default:0" json:"pajak_roda_dua"`
	PajakRodaEmpat  int64     `gorm:"type:bigint;not null;default:0" json:"pajak_roda_empat"`
	TotalPBB        int64     `gorm:"type:bigint;not null;default:0" json:"total_pbb"`
	DibuatPada      time.Time `gorm:"autoCreateTime" json:"dibuat_pada"`
	DiubahPada      time.Time `gorm:"autoUpdateTime" json:"diubah_pada"`
}

func (PendukungEkonomi) TableName() string { 
	return "pendukung_ekonomi" 
}

type Dokumen struct {
	BaseModel
	BiodataID  uuid.UUID `gorm:"type:text;index;not null" json:"biodata_id"`
	Kode       string    `gorm:"type:varchar(50);not null;uniqueIndex:idx_biodata_dokumen" json:"kode"`
	FilePath   string    `gorm:"type:text;not null" json:"file_path"`
	DibuatPada time.Time `gorm:"autoCreateTime" json:"dibuat_pada"`
}

func (Dokumen) TableName() string { 
	return "dokumen" 
}
