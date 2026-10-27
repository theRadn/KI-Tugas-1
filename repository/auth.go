
package repository

import (
	"context"
    "github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/theRadn/KI-Tugas-1/entity"
)

type AuthRepository struct {
	db *gorm.DB
}

func NewAuthRepository(db *gorm.DB) *AuthRepository {
	return &AuthRepository{db: db}
}

func (r *AuthRepository) ExistsByUsername(
	ctx context.Context,
	username string,
) (bool, error) {
	var count int64

	err := r.db.WithContext(ctx).
		Model(&entity.User{}).
		Where("nama_pengguna = ?", username).
		Count(&count).Error

	return count > 0, err
}

func (r *AuthRepository) ExistsByEmail(
	ctx context.Context,
	email string,
) (bool, error) {
	var count int64

	err := r.db.WithContext(ctx).
		Model(&entity.User{}).
		Where("email = ?", email).
		Count(&count).Error

	return count > 0, err
}

func (r *AuthRepository) Create(
	ctx context.Context,
	user *entity.User,
) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *AuthRepository) FindByUsername(
	ctx context.Context,
	username string,
) (*entity.User, error) {
	var user entity.User

	err := r.db.WithContext(ctx).
		Where("nama_pengguna = ?", username).
		First(&user).Error

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *AuthRepository) FindByEmail(
	ctx context.Context,
	email string,
) (*entity.User, error) {
	var user entity.User

	err := r.db.WithContext(ctx).
		Where("email = ?", email).
		First(&user).Error

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *AuthRepository) FindByID(
	ctx context.Context,
	id uuid.UUID,
) (*entity.User, error) {
	var user entity.User

	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&user).Error

	if err != nil {
		return nil, err
	}

	return &user, nil
}


func (r *AuthRepository) UpdatePassword(
    ctx context.Context,
    userID uuid.UUID,
    passwordHash string,
) error {
    result := r.db.WithContext(ctx).
        Model(&entity.User{}).
        Where("id = ?", userID).
        Update("kata_sandi_hash", passwordHash)

    if result.Error != nil {
        return result.Error
    }

    if result.RowsAffected == 0 {
        return gorm.ErrRecordNotFound
    }

    return nil
}