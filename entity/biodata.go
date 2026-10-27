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
	FotoPath         *string    `gorm:"type:text;serializer:enkripsi" json:"foto"`
	NamaIjazah       string     `gorm:"type:varchar(255);serializer:enkripsi" json:"nama_ijazah"`
	NamaKTM          string     `gorm:"type:varchar(20);serializer:enkripsi" json:"nama_ktm"`
	UkuranAlmamater  string     `gorm:"type:varchar(20);serializer:enkripsi" json:"ukuran_almamater"`
	ProgramStudi     string     `gorm:"type:varchar(100);serializer:enkripsi" json:"program_studi"`
	JalurMasuk       string     `gorm:"type:varchar(100);serializer:enkripsi" json:"jalur_masuk"`
	PeriodeMasuk     string     `gorm:"type:varchar(100);serializer:enkripsi" json:"periode_masuk"`
	Kewarganegaraan  string     `gorm:"type:varchar(100);serializer:enkripsi" json:"kewarganegaraan"`
	NIK              string     `gorm:"type:varchar(16);serializer:enkripsi" json:"nik"`
	NoPaspor         *string    `gorm:"type:varchar(50);serializer:enkripsi" json:"no_paspor"`
	StatusPernikahan string     `gorm:"type:varchar(50);serializer:enkripsi" json:"status_pernikahan"`
	JenisKelamin     string     `gorm:"type:varchar(20);serializer:enkripsi" json:"jenis_kelamin"`
	Agama            string     `gorm:"type:varchar(50);serializer:enkripsi" json:"agama"`
	TanggalLahir     *time.Time `gorm:"type:date;serializer:enkripsi" json:"tanggal_lahir"`
	NegaraLahir      string     `gorm:"type:varchar(100);serializer:enkripsi" json:"negara_lahir"`
	ProvinsiLahir    string     `gorm:"type:varchar(100);serializer:enkripsi" json:"provinsi_lahir"`
	KotaLahir        string     `gorm:"type:varchar(100);serializer:enkripsi" json:"kota_lahir"`
	TempatLahir      string     `gorm:"type:varchar(100);serializer:enkripsi" json:"tempat_lahir"`

	AlamatKTP        string `gorm:"type:text;serializer:enkripsi" json:"alamat_ktp"`
	NegaraAsal       string `gorm:"type:varchar(100);serializer:enkripsi" json:"negara_asal"`
	ProvinsiAsal     string `gorm:"type:varchar(100);serializer:enkripsi" json:"provinsi_asal"`
	KotaAsal         string `gorm:"type:varchar(100);serializer:enkripsi" json:"kota_asal"`
	KodePos          string `gorm:"type:varchar(5);serializer:enkripsi" json:"kode_pos"`
	AlamatDomisili   string `gorm:"type:text;serializer:enkripsi" json:"alamat_domisili"`
	NegaraDomisili   string `gorm:"type:varchar(100);serializer:enkripsi" json:"negara_domisili"`
	ProvinsiDomisili string `gorm:"type:varchar(100);serializer:enkripsi" json:"provinsi_domisili"`
	KotaDomisili     string `gorm:"type:varchar(100);serializer:enkripsi" json:"kota_domisili"`
	Email            string `gorm:"type:varchar(255);serializer:enkripsi" json:"email"`
	NoHP             string `gorm:"type:varchar(30);serializer:enkripsi" json:"no_hp"`
	NoKIPK           string `gorm:"type:varchar(50);serializer:enkripsi" json:"no_kip_k"`
	NoBPJS           string `gorm:"type:varchar(50);serializer:enkripsi" json:"no_bpjs"`
	SumberInformasi  string `gorm:"type:varchar(50);serializer:enkripsi" json:"sumber_informasi"`
	DaftarAsrama     bool   `gorm:"not null;default:false;serializer:enkripsi" json:"daftar_asrama"`

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
	ProvinsiSekolah string     `gorm:"type:varchar(100);serializer:enkripsi" json:"provinsi_sekolah"`
	KotaSekolah     string     `gorm:"type:varchar(100);serializer:enkripsi" json:"kota_sekolah"`
	NamaSekolah     string     `gorm:"type:varchar(255);serializer:enkripsi" json:"nama_sekolah"`
	Jurusan         string     `gorm:"type:varchar(100);serializer:enkripsi" json:"jurusan"`
	NISN            string     `gorm:"type:varchar(10);serializer:enkripsi" json:"nisn"`
	TahunMasukSMTA  int        `gorm:"type:int;serializer:enkripsi" json:"tahun_masuk_smta"`
	TahunLulusSMTA  int        `gorm:"type:int;serializer:enkripsi" json:"tahun_lulus_smta"`
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
	MataPelajaran string    `gorm:"type:varchar(100);not null;serializer:enkripsi" json:"mata_pelajaran"`
	Nilai         float64   `gorm:"type:decimal(4,1);not null;serializer:enkripsi" json:"nilai"`
}

func (NilaiTKA) TableName() string { 
	return "nilai_tka" 
}

type Rumah struct {
	BaseModel
	BiodataID        uuid.UUID `gorm:"type:text;uniqueIndex;not null" json:"biodata_id"`
	Kepemilikan      string    `gorm:"type:varchar(100);serializer:enkripsi" json:"kepemilikan"`
	TahunPerolehan   int       `gorm:"type:int;serializer:enkripsi" json:"tahun_perolehan"`
	SumberListrik    string    `gorm:"type:varchar(100);serializer:enkripsi" json:"sumber_listrik"`
	LuasTanah        int       `gorm:"type:int;serializer:enkripsi" json:"luas_tanah"`
	LuasBangunan     int       `gorm:"type:int;serializer:enkripsi" json:"luas_bangunan"`
	NJOP             int64     `gorm:"type:bigint;serializer:enkripsi" json:"njop"`
	MCK              string    `gorm:"type:varchar(100);serializer:enkripsi" json:"mck"`
	SumberAirMinum   string    `gorm:"type:varchar(100);serializer:enkripsi" json:"sumber_air_minum"`
	JarakPusatKotaKM int       `gorm:"type:int;serializer:enkripsi" json:"jarak_pusat_kota_km"`
	JumlahPenghuni   int       `gorm:"type:int;serializer:enkripsi" json:"jumlah_penghuni"`
	DibuatPada       time.Time `gorm:"autoCreateTime" json:"dibuat_pada"`
	DiubahPada       time.Time `gorm:"autoUpdateTime" json:"diubah_pada"`
}

func (Rumah) TableName() string { 
	return "rumah" 
}

type CatatanKhusus struct {
	BaseModel
	BiodataID            uuid.UUID `gorm:"type:text;uniqueIndex;not null" json:"biodata_id"`
	RencanaTempatTinggal string    `gorm:"type:varchar(100);serializer:enkripsi" json:"rencana_tempat_tinggal"`
	AdaDukunganKeluarga  bool      `gorm:"not null;default:false;serializer:enkripsi" json:"ada_dukungan_keluarga"`
	TransportasiDariAsal string    `gorm:"type:varchar(100);serializer:enkripsi" json:"transportasi_dari_asal"`
	TransportasiHarian   string    `gorm:"type:varchar(100);serializer:enkripsi" json:"transportasi_harian"`
	GolonganDarah        string    `gorm:"type:varchar(2);serializer:enkripsi" json:"golongan_darah"`
	Rhesus               string    `gorm:"type:varchar(10);serializer:enkripsi" json:"rhesus"`
	KondisiKesehatan     string    `gorm:"type:text;serializer:enkripsi" json:"kondisi_kesehatan"`
	Disabilitas          bool      `gorm:"not null;default:false;serializer:enkripsi" json:"disabilitas"`
	DibuatPada           time.Time `gorm:"autoCreateTime" json:"dibuat_pada"`
	DiubahPada           time.Time `gorm:"autoUpdateTime" json:"diubah_pada"`
}

func (CatatanKhusus) TableName() string { 
	return "catatan_khusus" 
}

type PendukungEkonomi struct {
	BaseModel
	BiodataID       uuid.UUID `gorm:"type:text;uniqueIndex;not null" json:"biodata_id"`
	PekerjaanAyah   string    `gorm:"type:varchar(255);serializer:enkripsi" json:"pekerjaan_ayah"`
	PekerjaanIbu    string    `gorm:"type:varchar(255);serializer:enkripsi" json:"pekerjaan_ibu"`
	PenghasilanAyah int64     `gorm:"type:bigint;not null;default:0;serializer:enkripsi" json:"penghasilan_ayah"`
	PenghasilanIbu  int64     `gorm:"type:bigint;not null;default:0;serializer:enkripsi" json:"penghasilan_ibu"`
	TanggunganAnak  int       `gorm:"type:int;not null;default:0;serializer:enkripsi" json:"tanggungan_anak"`
	TagihanListrik  int64     `gorm:"type:bigint;not null;default:0;serializer:enkripsi" json:"tagihan_listrik"`
	DayaListrik     string    `gorm:"type:varchar(50);serializer:enkripsi" json:"daya_listrik"`
	PajakRodaDua    int64     `gorm:"type:bigint;not null;default:0;serializer:enkripsi" json:"pajak_roda_dua"`
	PajakRodaEmpat  int64     `gorm:"type:bigint;not null;default:0;serializer:enkripsi" json:"pajak_roda_empat"`
	TotalPBB        int64     `gorm:"type:bigint;not null;default:0;serializer:enkripsi" json:"total_pbb"`
	DibuatPada      time.Time `gorm:"autoCreateTime" json:"dibuat_pada"`
	DiubahPada      time.Time `gorm:"autoUpdateTime" json:"diubah_pada"`
}

func (PendukungEkonomi) TableName() string { 
	return "pendukung_ekonomi" 
}

type Dokumen struct {
	BaseModel
	BiodataID  uuid.UUID `gorm:"type:text;not null;uniqueIndex:idx_dokumen_biodata_kode,priority:1" json:"biodata_id"`
	Kode       string    `gorm:"type:varchar(50);not null;uniqueIndex:idx_dokumen_biodata_kode,priority:2" json:"kode"`
	FilePath   string    `gorm:"type:text;not null" json:"file_path"`
	DibuatPada time.Time `gorm:"autoCreateTime" json:"dibuat_pada"`
}

func (Dokumen) TableName() string { 
	return "dokumen" 
}
