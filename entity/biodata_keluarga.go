package entity

import (
	"time"
	"github.com/google/uuid"
)

type BiodataKeluarga struct {
	BaseModel
	BiodataID uuid.UUID `gorm:"type:text;uniqueIndex;not null" json:"biodata_id"`

	// Data ayah atau wali
	AyahNIK                string     `gorm:"type:varchar(16)" json:"ayah_nik"`
	AyahNama               string     `gorm:"type:varchar(255)" json:"ayah_nama"`
	AyahTempatLahir        string     `gorm:"type:varchar(100)" json:"ayah_tempat_lahir"`
	AyahTanggalLahir       *time.Time `gorm:"type:date" json:"ayah_tanggal_lahir"`
	AyahPekerjaan          string     `gorm:"type:varchar(255)" json:"ayah_pekerjaan"`
	AyahInstansi           string     `gorm:"type:varchar(255)" json:"ayah_instansi"`
	AyahPendidikanTerakhir string     `gorm:"type:varchar(50)" json:"ayah_pendidikan_terakhir"`
	AyahNoHP               string     `gorm:"type:varchar(30)" json:"ayah_no_hp"`
	AyahStatus             string     `gorm:"type:varchar(20)" json:"ayah_status"`

	// Data ibu kandung
	IbuNIK                string     `gorm:"type:varchar(16)" json:"ibu_nik"`
	IbuNama               string     `gorm:"type:varchar(255)" json:"ibu_nama"`
	IbuTempatLahir        string     `gorm:"type:varchar(100)" json:"ibu_tempat_lahir"`
	IbuTanggalLahir       *time.Time `gorm:"type:date" json:"ibu_tanggal_lahir"`
	IbuPekerjaan          string     `gorm:"type:varchar(255)" json:"ibu_pekerjaan"`
	IbuInstansi           string     `gorm:"type:varchar(255)" json:"ibu_instansi"`
	IbuPendidikanTerakhir string     `gorm:"type:varchar(50)" json:"ibu_pendidikan_terakhir"`
	IbuNoHP               string     `gorm:"type:varchar(30)" json:"ibu_no_hp"`
	IbuStatus             string     `gorm:"type:varchar(20)" json:"ibu_status"`

	RangePendapatanOrtu string `gorm:"type:varchar(100)" json:"range_pendapatan_ortu"`
	NomorKK             string `gorm:"type:varchar(16)" json:"nomor_kk"`
	JumlahKakak         int    `gorm:"type:int;not null;default:0" json:"jumlah_kakak"`
	JumlahAdik          int    `gorm:"type:int;not null;default:0" json:"jumlah_adik"`

	AlamatOrtuAlamat         string `gorm:"type:text" json:"alamat_ortu_alamat"`
	AlamatOrtuRT             string `gorm:"type:varchar(3)" json:"alamat_ortu_rt"`
	AlamatOrtuRW             string `gorm:"type:varchar(3)" json:"alamat_ortu_rw"`
	AlamatOrtuKelurahan      string `gorm:"type:varchar(100)" json:"alamat_ortu_kelurahan"`
	AlamatOrtuKecamatan      string `gorm:"type:varchar(100)" json:"alamat_ortu_kecamatan"`
	AlamatOrtuProvinsi       string `gorm:"type:varchar(100)" json:"alamat_ortu_provinsi"`
	AlamatOrtuKota           string `gorm:"type:varchar(100)" json:"alamat_ortu_kota"`
	AlamatOrtuKodePos        string `gorm:"type:varchar(5)" json:"alamat_ortu_kode_pos"`
	AlamatOrtuNoTeleponRumah string `gorm:"type:varchar(30)" json:"alamat_ortu_no_telepon_rumah"`

	DibuatPada time.Time `gorm:"autoCreateTime" json:"dibuat_pada"`
	DiubahPada time.Time `gorm:"autoUpdateTime" json:"diubah_pada"`
}

func (BiodataKeluarga) TableName() string { 
	return "biodata_keluarga" 
}
