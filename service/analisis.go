package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
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
		potongan := item.KodeBerkas == KodeVideo || item.KodeBerkas == KodeDokumen
		if item.Status == "terunggah" && (len(item.Enkripsi) == 3 || (potongan && item.FilePath != "")) {
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
	potongan := berkas.KodeBerkas == KodeVideo || berkas.KodeBerkas == KodeDokumen
	if berkas.Status != "terunggah" || (!potongan && len(berkas.Enkripsi) != 3) || (potongan && berkas.FilePath == "") {
		return dto.UjiDekripsiResponse{}, BerkasUnready
	}

	response := dto.UjiDekripsiResponse{BerkasID: req.BerkasID, Hasil: make([]dto.HasilAnalisisResponse, 0, 3)}
	stored := entity.HasilAnalisis{UserID: userID, BerkasID: berkasID, JumlahPercobaan: req.JumlahPercobaan}
	for _, name := range algoritmaBerkas {
		algorithm, secret, err := kunciBerkas(name)
		if err != nil {
			return dto.UjiDekripsiResponse{}, err
		}
		var encrypted []utils.EncryptedData
		var ciphertextSize int64
		if potongan {
			encrypted, ciphertextSize, err = muatPotonganAnalisis(berkas, name, algorithm)
		} else {
			encrypted, ciphertextSize, err = muatBerkasAnalisis(berkas, name, algorithm)
		}
		if err != nil {
			return dto.UjiDekripsiResponse{}, err
		}
		if err := dekripsiAnalisis(encrypted, secret, berkas.UkuranByte); err != nil {
			return dto.UjiDekripsiResponse{}, err
		}

		samples := make([]float64, req.JumlahPercobaan)
		for i := range samples {
			start := time.Now()
			if err := dekripsiAnalisis(encrypted, secret, berkas.UkuranByte); err != nil {
				return dto.UjiDekripsiResponse{}, err
			}
			samples[i] = float64(time.Since(start).Nanoseconds()) / 1e6
		}
		stat := statistikDekripsi(samples, berkas.UkuranByte, ciphertextSize)
		stat.Algoritma = name
		stat.Mode = string(algorithm)
		response.Hasil = append(response.Hasil, stat)
		stored.Hasil = append(stored.Hasil, entity.HasilAnalisisAlgoritma{
			Algoritma: stat.Algoritma, Mode: stat.Mode,
			RataRataMS: stat.RataRataMs, MedianMS: stat.MedianMs,
			SimpanganBakuMS: stat.SimpanganBakuMs, MinMS: stat.MinMs, MaksMS: stat.MaksMs,
			UkuranCiphertextByte: stat.UkuranCiphertextByte,
			SelisihUkuranByte:    stat.SelisihUkuranByte, MBPerDetik: stat.Mbps,
		})
	}
	if err := s.db.WithContext(ctx).Create(&stored).Error; err != nil {
		return dto.UjiDekripsiResponse{}, err
	}
	return response, nil
}

func muatBerkasAnalisis(berkas entity.Berkas, name string, algorithm utils.Algorithm) ([]utils.EncryptedData, int64, error) {
	for _, item := range berkas.Enkripsi {
		if item.Algoritma != name {
			continue
		}
		file, err := os.ReadFile(item.CiphertextPath)
		if err != nil {
			return nil, 0, err
		}
		var encrypted utils.EncryptedData
		if err := json.Unmarshal(file, &encrypted); err != nil {
			return nil, 0, err
		}
		if encrypted.Algorithm != algorithm || int64(len(encrypted.Ciphertext)) != item.UkuranCiphertextByte {
			return nil, 0, errors.New("metadata ciphertext tidak sesuai")
		}
		return []utils.EncryptedData{encrypted}, item.UkuranCiphertextByte, nil
	}
	return nil, 0, BerkasUnready
}

func muatPotonganAnalisis(berkas entity.Berkas, name string, algorithm utils.Algorithm) ([]utils.EncryptedData, int64, error) {
	manifestFile, err := os.ReadFile(filepath.Join(berkas.FilePath, "manifest.json"))
	if err != nil {
		return nil, 0, err
	}
	manifestPlain, err := unpackVideoData(manifestFile, "AES")
	if err != nil {
		return nil, 0, err
	}
	var count int
	if berkas.KodeBerkas == KodeVideo {
		var manifest videoManifest
		err = json.Unmarshal(manifestPlain, &manifest)
		count = manifest.ChunkCount
		if count < 1 || count > maxVideoChunks || manifest.SizeBytes != berkas.UkuranByte {
			return nil, 0, BerkasUnready
		}
	} else {
		var manifest documentManifest
		err = json.Unmarshal(manifestPlain, &manifest)
		count = manifest.ChunkCount
		if count < 1 || count > maxDocumentChunks || manifest.SizeBytes != berkas.UkuranByte {
			return nil, 0, BerkasUnready
		}
	}
	if err != nil || count != int((berkas.UkuranByte+MaxVideoChunkBytes-1)/MaxVideoChunkBytes) {
		return nil, 0, BerkasUnready
	}

	encrypted := make([]utils.EncryptedData, 0, count)
	var ciphertextSize int64
	for i := 0; i < count; i++ {
		file, err := os.ReadFile(filepath.Join(berkas.FilePath, fmt.Sprintf("%06d.json", i)))
		if err != nil {
			return nil, 0, err
		}
		var versions map[string]utils.EncryptedData
		if err := json.Unmarshal(file, &versions); err != nil {
			return nil, 0, err
		}
		data, ok := versions[name]
		expected := MaxVideoChunkBytes
		if i == count-1 {
			expected = int(berkas.UkuranByte - int64(i)*MaxVideoChunkBytes)
		}
		if !ok || len(versions) != 3 || data.Algorithm != algorithm || len(data.Ciphertext) != expected {
			return nil, 0, BerkasUnready
		}
		encrypted = append(encrypted, data)
		ciphertextSize += int64(len(data.Ciphertext))
	}
	return encrypted, ciphertextSize, nil
}

func dekripsiAnalisis(encrypted []utils.EncryptedData, secret []byte, expectedSize int64) error {
	var total int64
	for _, data := range encrypted {
		plaintext, err := utils.Decrypt(data, secret, secret)
		if err != nil {
			return err
		}
		total += int64(len(plaintext))
	}
	if total != expectedSize {
		return errors.New("ukuran hasil dekripsi tidak sesuai")
	}
	return nil
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
		MinMs:           sorted[0], MaksMs: sorted[len(sorted)-1],
		UkuranCiphertextByte: ciphertextSize,
		SelisihUkuranByte:    ciphertextSize - originalSize,
		Mbps:                 mbPerSecond,
	}
}
