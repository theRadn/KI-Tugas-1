package entity

import (
	"time"

	"github.com/google/uuid"
)

const (
	StatusBerbagiMenunggu    = "menunggu"
	StatusBerbagiDisetujui   = "disetujui"
	StatusBerbagiDitolak     = "ditolak"
	StatusBerbagiDicabut     = "dicabut"
	StatusBerbagiKedaluwarsa = "kedaluwarsa"
)

// PermintaanBerbagi merekam satu permintaan akses data lintas pengguna.
// PemintaID yang mengajukan, PemilikID yang memiliki data & berhak menjawab.
type PermintaanBerbagi struct {
	BaseModel
	PemintaID uuid.UUID `gorm:"type:text;index;not null" json:"peminta_id"`
	PemilikID uuid.UUID `gorm:"type:text;index;not null" json:"pemilik_id"`

	Status string `gorm:"type:varchar(20);not null;default:menunggu" json:"status"`

	// Cakupan berupa JSON array nama tab biodata (data-diri, pendidikan, dst).
	CakupanDiminta   string `gorm:"type:text;not null" json:"cakupan_diminta"`
	CakupanDisetujui string `gorm:"type:text" json:"cakupan_disetujui"`

	// Berkas berupa JSON array UUID berkas yang diminta/disetujui.
	BerkasDiminta   string `gorm:"type:text" json:"berkas_diminta"`
	BerkasDisetujui string `gorm:"type:text" json:"berkas_disetujui"`

	PesanPeminta  string `gorm:"type:text" json:"pesan_peminta"`
	PesanPemilik  string `gorm:"type:text" json:"pesan_pemilik"`
	BerlakuSampai *time.Time `gorm:"type:timestamp" json:"berlaku_sampai"`

	DibuatPada time.Time `gorm:"autoCreateTime" json:"dibuat_pada"`
	DiubahPada time.Time `gorm:"autoUpdateTime" json:"diubah_pada"`

	Kunci []KunciBerbagi `gorm:"foreignKey:PermintaanID;references:ID" json:"-"`
}

func (PermintaanBerbagi) TableName() string { return "permintaan_berbagi" }

// KunciBerbagi menyimpan kunci DEK milik pemilik yang sudah dibungkus ulang
// memakai KEK peminta. Plaintext kunci tidak pernah ditulis ke DB.
//
// Jenis:
//   - "scope"  : RefID = nama scope EncryptionKeySet (biodata, pendidikan, dst)
//   - "berkas" : RefID = "<berkasID>"
//   - "chunk"  : RefID = "<berkasID>:<index>"
type KunciBerbagi struct {
	BaseModel
	PermintaanID uuid.UUID `gorm:"type:text;index;not null" json:"permintaan_id"`
	Jenis        string    `gorm:"type:varchar(20);not null" json:"jenis"`
	RefID        string    `gorm:"type:varchar(100);not null" json:"ref_id"`
	Algoritma    string    `gorm:"type:varchar(20);not null" json:"algoritma"`

	KunciTerbungkus []byte    `gorm:"type:blob;not null" json:"-"`
	WaktuDibuat     time.Time `gorm:"autoCreateTime" json:"waktu_dibuat"`
}

func (KunciBerbagi) TableName() string { return "kunci_berbagi" }
