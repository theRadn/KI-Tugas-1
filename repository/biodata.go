package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/theRadn/KI-Tugas-1/entity"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BiodataRepository struct {
	db *gorm.DB
}

func NewBiodataRepository(db *gorm.DB) *BiodataRepository {
	return &BiodataRepository{db: db}
}

func (r *BiodataRepository) Get(
	ctx context.Context,
	userID uuid.UUID,
) (*entity.Biodata, error) {
	var b entity.Biodata

	err := r.db.WithContext(ctx).
		Preload("Pendidikan.NilaiTKA").
		Preload("Keluarga").
		Preload("Rumah").
		Preload("CatatanKhusus").
		Preload("PendukungEkonomi").
		Where("user_id = ?", userID).
		First(&b).Error

	if err != nil {
		return nil, err
	}

	return &b, nil
}

func (r *BiodataRepository) SaveMain(
	ctx context.Context,
	b *entity.Biodata,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var old entity.Biodata

		err := tx.Where("user_id = ?", b.UserID).
			First(&old).Error

		if errors.Is(err, gorm.ErrRecordNotFound) {
			no, err := nomorPendaftaranBaru(tx)
			if err != nil {
				return err
			}
			b.NoPendaftaran = no
			return tx.Create(b).Error
		}

		if err != nil {
			return err
		}

		b.ID = old.ID

		// Select("*") supaya nilai nol (false, teks kosong) ikut tersimpan.
		// NoPendaftaran dibuat sistem, jadi tidak boleh tertimpa.
		return tx.Model(&old).
			Select("*").
			Omit(
				"ID",
				"UserID",
				"NoPendaftaran",
				"DibuatPada",
				"Pendidikan",
				"Keluarga",
				"Rumah",
				"CatatanKhusus",
				"PendukungEkonomi",
				"Berkas",
			).
			Updates(b).Error
	})
}

func (r *BiodataRepository) SaveRelation(
	ctx context.Context,
	userID uuid.UUID,
	value any,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var b entity.Biodata

		err := tx.Where("user_id = ?", userID).
			First(&b).Error

		if errors.Is(err, gorm.ErrRecordNotFound) {
			no, err := nomorPendaftaranBaru(tx)
			if err != nil {
				return err
			}
			b = entity.Biodata{UserID: userID, NoPendaftaran: no}

			if err := tx.Create(&b).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}

		switch v := value.(type) {
		case *entity.Rumah:
			v.BiodataID = b.ID
			return saveOne(tx, "rumah", v.BiodataID, v)

		case *entity.CatatanKhusus:
			v.BiodataID = b.ID
			return saveOne(tx, "catatan_khusus", v.BiodataID, v)

		case *entity.PendukungEkonomi:
			v.BiodataID = b.ID
			return saveOne(tx, "pendukung_ekonomi", v.BiodataID, v)

		case *entity.BiodataKeluarga:
			v.BiodataID = b.ID
			return saveOne(tx, "biodata_keluarga", v.BiodataID, v)

		case *entity.Pendidikan:
			v.BiodataID = b.ID

			var old entity.Pendidikan
			err := tx.Where("biodata_id = ?", b.ID).
				First(&old).Error

			if errors.Is(err, gorm.ErrRecordNotFound) {
				if err := tx.Omit("NilaiTKA").Create(v).Error; err != nil {
					return err
				}
			} else if err != nil {
				return err
			} else {
				v.ID = old.ID

				if err := tx.Model(&old).
					Select("*").
					Omit(
						"ID",
						"BiodataID",
						"DibuatPada",
						"NilaiTKA",
					).
					Updates(v).Error; err != nil {
					return err
				}
			}

			if err := tx.Where("pendidikan_id = ?", v.ID).
				Delete(&entity.NilaiTKA{}).Error; err != nil {
				return err
			}

			for i := range v.NilaiTKA {
				v.NilaiTKA[i].PendidikanID = v.ID

				if err := tx.Create(&v.NilaiTKA[i]).Error; err != nil {
					return err
				}
			}

			return nil

		default:
			return errors.New("tipe data biodata tidak didukung")
		}
	})
}

func saveOne(
	tx *gorm.DB,
	table string,
	id uuid.UUID,
	value any,
) error {
	var count int64

	if err := tx.Table(table).
		Where("biodata_id = ?", id).
		Count(&count).Error; err != nil {
		return err
	}

	if count == 0 {
		return tx.Create(value).Error
	}

	return tx.Model(value).
		Where("biodata_id = ?", id).
		Select("*").
		Omit("ID", "BiodataID", "DibuatPada").
		Updates(value).Error
}

// nomorPendaftaranBaru dipanggil di dalam transaksi tepat sebelum baris
// Biodata baru dibuat, contoh hasil: SIPMABA-2026-000001.
func nomorPendaftaranBaru(tx *gorm.DB) (string, error) {
	var jumlah int64
	if err := tx.Model(&entity.Biodata{}).Count(&jumlah).Error; err != nil {
		return "", err
	}
	return fmt.Sprintf("SIPMABA-%d-%06d", time.Now().Year(), jumlah+1), nil
}

// PastikanBiodata mengambil baris biodata pengguna, atau membuatnya
// (beserta nomor pendaftaran) bila belum ada. Dipakai saat unggah berkas.
func (r *BiodataRepository) PastikanBiodata(
	ctx context.Context,
	userID uuid.UUID,
) (*entity.Biodata, error) {
	var b entity.Biodata
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Where("user_id = ?", userID).First(&b).Error
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		no, err := nomorPendaftaranBaru(tx)
		if err != nil {
			return err
		}
		b = entity.Biodata{UserID: userID, NoPendaftaran: no}
		return tx.Create(&b).Error
	})
	if err != nil {
		return nil, err
	}
	return &b, nil
}
