package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/theRadn/KI-Tugas-1/entity"
	"github.com/theRadn/KI-Tugas-1/utils"

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

	materials, err := r.loadKeyMaterials(
		r.db,
		ctx,
		userID,
		false,
		"biodata",
		"biodata_keluarga",
		"pendidikan",
		"nilai_tka",
		"rumah",
		"catatan_khusus",
		"pendukung_ekonomi",
	)
	if err != nil {
		return nil, err
	}
	ctx = entity.WithKeyMaterial(ctx, materials)

	err = r.db.WithContext(ctx).
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
		materials, err := r.loadKeyMaterials(tx, ctx, b.UserID, true, "biodata")
		if err != nil {
			return err
		}
		tx = tx.WithContext(entity.WithKeyMaterial(ctx, materials))

		var old entity.Biodata

		err = tx.Where("user_id = ?", b.UserID).First(&old).Error

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
		// Loading the parent row below also scans encrypted Biodata fields,
		// so its key material is required in every relation transaction.
		scopes := append([]string{"biodata"}, scopesForValue(value)...)
		materials, err := r.loadKeyMaterials(tx, ctx, userID, true, scopes...)
		if err != nil {
			return err
		}
		tx = tx.WithContext(entity.WithKeyMaterial(ctx, materials))

		var b entity.Biodata

		err = tx.Where("user_id = ?", userID).
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

func scopesForValue(value any) []string {
	switch value.(type) {
	case *entity.BiodataKeluarga:
		return []string{"biodata_keluarga"}
	case *entity.Pendidikan:
		return []string{"pendidikan", "nilai_tka"}
	case *entity.Rumah:
		return []string{"rumah"}
	case *entity.CatatanKhusus:
		return []string{"catatan_khusus"}
	case *entity.PendukungEkonomi:
		return []string{"pendukung_ekonomi"}
	default:
		return nil
	}
}

func (r *BiodataRepository) loadKeyMaterials(
	db *gorm.DB,
	ctx context.Context,
	userID uuid.UUID,
	create bool,
	scopes ...string,
) (map[string]entity.KeyMaterial, error) {
	if !create {
		var count int64
		if err := db.WithContext(ctx).
			Model(&entity.EncryptionKeySet{}).
			Where("user_id = ? AND scope IN ?", userID, scopes).
			Count(&count).Error; err != nil {
			return nil, err
		}
		if count == 0 {
			return map[string]entity.KeyMaterial{}, nil
		}
	}

	var user entity.User
	if err := db.WithContext(ctx).First(&user, "id = ?", userID).Error; err != nil {
		return nil, err
	}

	masterKEK, err := utils.MasterKEK()
	if err != nil {
		return nil, err
	}
	userKEKWrapped, err := utils.UnmarshalWrappedKey(user.KEKWrapped)
	if err != nil {
		return nil, err
	}
	userKEK, err := utils.UnwrapKey(
		masterKEK,
		userKEKWrapped,
		[]byte("user:"+userID.String()+":kek"),
	)
	if err != nil {
		return nil, err
	}

	result := make(map[string]entity.KeyMaterial, len(scopes))
	for _, scope := range scopes {
		var row entity.EncryptionKeySet
		err := db.WithContext(ctx).
			Where("user_id = ? AND scope = ?", userID, scope).
			First(&row).Error
		if errors.Is(err, gorm.ErrRecordNotFound) && create {
			row, err = newKeySet(userID, scope, userKEK)
			if err == nil {
				err = db.WithContext(ctx).Create(&row).Error
			}
		}
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) && !create {
				continue
			}
			return nil, err
		}

		keys := make(map[string][]byte, 3)
		for algorithm, wrappedBytes := range map[string][]byte{
			"AES": row.WrappedAES,
			"DES": row.WrappedDES,
			"RC4": row.WrappedRC4,
		} {
			wrapped, err := utils.UnmarshalWrappedKey(wrappedBytes)
			if err != nil {
				return nil, err
			}
			key, err := utils.UnwrapKey(
				userKEK,
				wrapped,
				keySetAAD(userID, scope, algorithm),
			)
			if err != nil {
				return nil, err
			}
			keys[algorithm] = key
		}
		result[scope] = entity.KeyMaterial{UserID: userID, Scope: scope, Keys: keys}
	}
	return result, nil
}

func newKeySet(userID uuid.UUID, scope string, userKEK []byte) (entity.EncryptionKeySet, error) {
	keys := map[string]int{"AES": 32, "DES": 8, "RC4": 32}
	wrapped := make(map[string][]byte, len(keys))
	for algorithm, size := range keys {
		dek, err := utils.GenerateKey(size)
		if err != nil {
			return entity.EncryptionKeySet{}, err
		}
		value, err := utils.WrapKey(userKEK, dek, keySetAAD(userID, scope, algorithm))
		if err != nil {
			return entity.EncryptionKeySet{}, err
		}
		wrapped[algorithm], err = utils.MarshalWrappedKey(value)
		if err != nil {
			return entity.EncryptionKeySet{}, err
		}
	}
	return entity.EncryptionKeySet{
		UserID:     userID,
		Scope:      scope,
		WrappedAES: wrapped["AES"],
		WrappedDES: wrapped["DES"],
		WrappedRC4: wrapped["RC4"],
		KeyVersion: "v1",
	}, nil
}

func keySetAAD(userID uuid.UUID, scope, algorithm string) []byte {
	return []byte(fmt.Sprintf(
		"user:%s:scope:%s:%s-dek",
		userID,
		scope,
		algorithm,
	))
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
		materials, err := r.loadKeyMaterials(tx, ctx, userID, true, "biodata")
		if err != nil {
			return err
		}
		tx = tx.WithContext(entity.WithKeyMaterial(ctx, materials))

		err = tx.Where("user_id = ?", userID).First(&b).Error
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
