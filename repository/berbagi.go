package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/theRadn/KI-Tugas-1/entity"
	"gorm.io/gorm"
)

type BerbagiRepository struct {
	db *gorm.DB
}

func NewBerbagiRepository(db *gorm.DB) *BerbagiRepository {
	return &BerbagiRepository{db: db}
}

func (r *BerbagiRepository) Create(ctx context.Context, p *entity.PermintaanBerbagi) error {
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *BerbagiRepository) Get(ctx context.Context, id uuid.UUID) (*entity.PermintaanBerbagi, error) {
	var p entity.PermintaanBerbagi
	if err := r.db.WithContext(ctx).First(&p, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *BerbagiRepository) DaftarMasuk(ctx context.Context, pemilikID uuid.UUID) ([]entity.PermintaanBerbagi, error) {
	var result []entity.PermintaanBerbagi
	err := r.db.WithContext(ctx).
		Where("pemilik_id = ?", pemilikID).
		Order("dibuat_pada DESC").
		Find(&result).Error
	return result, err
}

func (r *BerbagiRepository) DaftarKeluar(ctx context.Context, pemintaID uuid.UUID) ([]entity.PermintaanBerbagi, error) {
	var result []entity.PermintaanBerbagi
	err := r.db.WithContext(ctx).
		Where("peminta_id = ?", pemintaID).
		Order("dibuat_pada DESC").
		Find(&result).Error
	return result, err
}

func (r *BerbagiRepository) UpdateStatus(
	ctx context.Context,
	id uuid.UUID,
	dari string,
	perubahan map[string]any,
) (int64, error) {
	res := r.db.WithContext(ctx).Model(&entity.PermintaanBerbagi{}).
		Where("id = ? AND status = ?", id, dari).
		Updates(perubahan)
	return res.RowsAffected, res.Error
}

func (r *BerbagiRepository) SimpanKunci(ctx context.Context, keys []entity.KunciBerbagi) error {
	if len(keys) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&keys).Error
}

func (r *BerbagiRepository) HapusKunci(ctx context.Context, permintaanID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("permintaan_id = ?", permintaanID).
		Delete(&entity.KunciBerbagi{}).Error
}

func (r *BerbagiRepository) KunciScope(
	ctx context.Context, permintaanID uuid.UUID, scope string,
) ([]entity.KunciBerbagi, error) {
	var out []entity.KunciBerbagi
	err := r.db.WithContext(ctx).
		Where("permintaan_id = ? AND jenis = ? AND ref_id = ?",
			permintaanID, "scope", scope).
		Find(&out).Error
	return out, err
}

func (r *BerbagiRepository) KunciRef(
	ctx context.Context, permintaanID uuid.UUID, jenis, refID string,
) ([]entity.KunciBerbagi, error) {
	var out []entity.KunciBerbagi
	err := r.db.WithContext(ctx).
		Where("permintaan_id = ? AND jenis = ? AND ref_id = ?",
			permintaanID, jenis, refID).
		Find(&out).Error
	return out, err
}

func (r *BerbagiRepository) ByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	var u entity.User
	if err := r.db.WithContext(ctx).First(&u, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *BerbagiRepository) ByUsername(ctx context.Context, nama string) (*entity.User, error) {
	var u entity.User
	if err := r.db.WithContext(ctx).First(&u, "nama_pengguna = ?", nama).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// DB mengembalikan handle agar service bisa menjalankan query tambahan
// (mis. biodata dengan key material) tanpa mengimpor gorm langsung.
func (r *BerbagiRepository) DB() *gorm.DB { return r.db }
