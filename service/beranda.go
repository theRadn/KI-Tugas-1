package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/theRadn/KI-Tugas-1/dto"
	"github.com/theRadn/KI-Tugas-1/entity"
	"gorm.io/gorm"
)

type BerandaService interface {
	Get(ctx context.Context, userID uuid.UUID) (dto.BerandaResponse, error)
}

type berandaService struct {
	db *gorm.DB
}

func NewBerandaService(db *gorm.DB) BerandaService {
	return &berandaService{db: db}
}

func (s *berandaService) Get(ctx context.Context, userID uuid.UUID) (dto.BerandaResponse, error) {
	var user entity.User
	if err := s.db.WithContext(ctx).First(&user, "id = ?", userID).Error; err != nil {
		return dto.BerandaResponse{}, err
	}

	// bagian-bagian biodata yang harus dicek kelengkapannya
	response := dto.BerandaResponse{NamaPengguna: user.NamaPengguna}
	var biodata entity.Biodata
	if err := s.db.WithContext(ctx).
		Preload("Pendidikan").Preload("Keluarga").Preload("Rumah").
		Preload("CatatanKhusus").Preload("PendukungEkonomi").
		First(&biodata, "user_id = ?", userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.KelengkapanData = daftarBagian(nil)
			return response, nil
		}
		return dto.BerandaResponse{}, err
	}

	// kalau biodata ditemukan, isi data yang relevan ke response
	response.NoPendaftaran = biodata.NoPendaftaran
	response.KelengkapanData = daftarBagian(&biodata)
	var totals struct {
		JumlahBerkas    int64
		TotalUkuranByte int64
	}
	if err := s.db.WithContext(ctx).Model(&entity.Berkas{}). // hitung jumlah berkas dan total ukuran byte
		Select("COUNT(*) AS jumlah_berkas, COALESCE(SUM(ukuran_byte), 0) AS total_ukuran_byte").
		Where("biodata_id = ?", biodata.ID).Scan(&totals).Error; err != nil {
		return dto.BerandaResponse{}, err
	}
	response.JumlahBerkas = totals.JumlahBerkas
	response.TotalUkuranByte = totals.TotalUkuranByte
	return response, nil
}

// bikin list status biodata per bagian buat halaman beranda
func daftarBagian(biodata *entity.Biodata) []dto.StatusBagian {
	names := []string{"data_diri", "pendidikan", "keluarga", "rumah", "catatan_khusus", "pendukung"}
	result := make([]dto.StatusBagian, len(names))
	for i, name := range names {
		result[i].Bagian = name
	}
	if biodata == nil {
		return result
	}
	result[0].Tersimpan = true
	result[0].DisimpanPada = &biodata.DiubahPada
	setStatus(&result[1], biodata.Pendidikan.ID, biodata.Pendidikan.DiubahPada)
	setStatus(&result[2], biodata.Keluarga.ID, biodata.Keluarga.DiubahPada)
	setStatus(&result[3], biodata.Rumah.ID, biodata.Rumah.DiubahPada)
	setStatus(&result[4], biodata.CatatanKhusus.ID, biodata.CatatanKhusus.DiubahPada)
	setStatus(&result[5], biodata.PendukungEkonomi.ID, biodata.PendukungEkonomi.DiubahPada)
	return result
}

func setStatus(bagian *dto.StatusBagian, id uuid.UUID, changed time.Time) {
	if id != uuid.Nil {
		bagian.Tersimpan = true
		bagian.DisimpanPada = &changed
	}
}
