package service

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"os"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/theRadn/KI-Tugas-1/dto"
	"github.com/theRadn/KI-Tugas-1/entity"
	"github.com/theRadn/KI-Tugas-1/utils"
	"gorm.io/gorm"
)

var BerkasUnready = errors.New("not encrypted yet")

type AnalisisService interface {
	DaftarBerkas(ctx context.Context, userID uuid.UUID) ([]dto.BerkasResponse, error)
	UjiDekripsi(ctx context.Context, userID uuid.UUID, req dto.UjiDekripsiRequest) (dto.UjiDekripsiResponse, error)
}

type analisisService struct {
	db *gorm.DB
}

func NewAnalisisService(db *gorm.DB) AnalisisService {
	return &analisisService{db: db}
}

func (s *analisisService) DaftarBerkas(ctx context.Context, userID uuid.UUID) ([]dto.BerkasResponse, error) {
	var berkas []entity.Berkas
	err := s.db.WithContext(ctx).Model(&entity.Berkas{}).
		Joins("JOIN biodata ON biodata.id = berkas.biodata_id").
		Where("biodata.user_id = ?", userID).Preload("Enkripsi").
		Order("berkas.dibuat_pada DESC").Find(&berkas).Error
	if err != nil {
		return nil, err
	}
	result := make([]dto.BerkasResponse, 0, len(berkas))
	for _, item := range berkas {
		if len(item.Enkripsi) == 3 {
			result = append(result, dto.BerkasResponse{
				ID: item.ID.String(), NamaAsli: item.NamaAsli,
				UkuranByte: item.UkuranByte, KodeBerkas: item.KodeBerkas,
				TipeMIME: item.TipeMIME, Status: item.Status,
				DiunggahPada: item.DiunggahPada,
			})
		}
	}
	return result, nil
}

func (s *analisisService) UjiDekripsi(ctx context.Context, userID uuid.UUID, req dto.UjiDekripsiRequest) (dto.UjiDekripsiResponse, error) {
	berkasID, err := uuid.Parse(req.BerkasID)
	if err != nil || req.JumlahPercobaan < 1 || req.JumlahPercobaan > 100 {
		return dto.UjiDekripsiResponse{}, errors.New("permintaan uji tidak valid")
	}
	var berkas entity.Berkas
	err = s.db.WithContext(ctx).Joins("JOIN biodata ON biodata.id = berkas.biodata_id").
		Where("biodata.user_id = ? AND berkas.id = ?", userID, berkasID).
		Preload("Enkripsi").First(&berkas).Error
	if err != nil {
		return dto.UjiDekripsiResponse{}, err
	}
	if len(berkas.Enkripsi) != 3 {
		return dto.UjiDekripsiResponse{}, BerkasUnready
	}

	response := dto.UjiDekripsiResponse{BerkasID: req.BerkasID, Hasil: make([]dto.HasilAnalisisResponse, 0, 3)}
	stored := entity.HasilAnalisis{UserID: userID, BerkasID: berkasID, JumlahPercobaan: req.JumlahPercobaan}
	seen := make(map[utils.Algorithm]bool)
	for _, item := range berkas.Enkripsi {
		algorithm, secretName := algoritmaDanSecret(item.Algoritma)
		if algorithm == "" || seen[algorithm] {
			return dto.UjiDekripsiResponse{}, BerkasUnready
		}
		seen[algorithm] = true
		secret := []byte(os.Getenv(secretName))
		file, err := os.ReadFile(item.CiphertextPath)
		if err != nil { // general error
			return dto.UjiDekripsiResponse{}, err
		}
		var encrypted utils.EncryptedData
		if err := json.Unmarshal(file, &encrypted); err != nil { // JSON unmarshaling error
			return dto.UjiDekripsiResponse{}, err
		}
		if encrypted.Algorithm != algorithm || int64(len(encrypted.Ciphertext)) != item.UkuranCiphertextByte { // metadata mismatch
			return dto.UjiDekripsiResponse{}, errors.New("metadata ciphertext tidak sesuai")
		}
		plaintext, err := utils.Decrypt(encrypted, secret, secret)
		if err != nil || int64(len(plaintext)) != berkas.UkuranByte { // decryption error or size mismatch
			return dto.UjiDekripsiResponse{}, errors.New("ciphertext tidak dapat didekripsi")
		}

		samples := make([]float64, req.JumlahPercobaan)
		for i := range samples {
			start := time.Now()
			utils.Decrypt(encrypted, secret, secret)
			samples[i] = float64(time.Since(start).Nanoseconds()) / 1e6
		}
		stat := statistikDekripsi(samples, berkas.UkuranByte, item.UkuranCiphertextByte)
		stat.Algoritma = item.Algoritma
		stat.Mode = item.Mode
		response.Hasil = append(response.Hasil, stat)
		stored.Hasil = append(stored.Hasil, entity.HasilAnalisisAlgoritma{
			Algoritma: stat.Algoritma, Mode: stat.Mode,
			RataRataMS: stat.RataRataMs, MedianMS: stat.MedianMs,
			SimpanganBakuMS: stat.SimpanganBakuMs, MinMS: stat.MinMs, MaksMS: stat.MaksMs,
			UkuranCiphertextByte: stat.UkuranCiphertextByte,
			SelisihUkuranByte: stat.SelisihUkuranByte, MBPerDetik: stat.Mbps,
		})
	}
	if err := s.db.WithContext(ctx).Create(&stored).Error; err != nil {
		return dto.UjiDekripsiResponse{}, err
	}
	return response, nil
}

func algoritmaDanSecret(name string) (utils.Algorithm, string) {
	switch name {
	case "AES", string(utils.AlgorithmAES256CTR):
		return utils.AlgorithmAES256CTR, "DATABASE_SECRET_AES256"
	case "DES", string(utils.AlgorithmDESCTR):
		return utils.AlgorithmDESCTR, "DATABASE_SECRET_DES"
	case "RC4":
		return utils.AlgorithmRC4, "DATABASE_SECRET_RC4"
	}
	return "", ""
}

func statistikDekripsi(samples []float64, originalSize, ciphertextSize int64) dto.HasilAnalisisResponse {
	sorted := append([]float64(nil), samples...)
	sort.Float64s(sorted)
	sum := 0.0
	for _, sample := range samples {
		sum += sample
	}
	mean := sum / float64(len(samples))
	variance := 0.0
	for _, sample := range samples {
		variance += (sample - mean) * (sample - mean)
	}
	median := sorted[len(sorted)/2]
	if len(sorted)%2 == 0 {
		median = (sorted[len(sorted)/2-1] + median) / 2
	}
	mbPerSecond := 0.0
	if mean > 0 {
		mbPerSecond = float64(originalSize) / (mean / 1000) / 1e6
	}
	return dto.HasilAnalisisResponse{
		RataRataMs: mean, MedianMs: median,
		SimpanganBakuMs: math.Sqrt(variance / float64(len(samples))),
		MinMs: sorted[0], MaksMs: sorted[len(sorted)-1],
		UkuranCiphertextByte: ciphertextSize,
		SelisihUkuranByte: ciphertextSize - originalSize,
		Mbps: mbPerSecond,
	}
}
