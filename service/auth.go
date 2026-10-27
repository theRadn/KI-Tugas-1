package service

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/theRadn/KI-Tugas-1/dto"
	"github.com/theRadn/KI-Tugas-1/entity"
	"github.com/theRadn/KI-Tugas-1/repository"
	"github.com/theRadn/KI-Tugas-1/utils"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrUserAlreadyExists  = errors.New("username atau email sudah terdaftar")
	ErrInvalidCredentials = errors.New("username atau kata sandi salah")
	ErrPasswordMismatch   = errors.New("konfirmasi kata sandi tidak cocok")
	ErrEmailNotFound      = errors.New("email tidak ditemukan")
)

type AuthService struct {
	repo *repository.AuthRepository
}

func NewAuthService(repo *repository.AuthRepository) *AuthService {
	return &AuthService{repo: repo}
}

func (s *AuthService) Daftar(
	ctx context.Context,
	req dto.DaftarRequest,
) (*entity.User, error) {
	if req.KataSandi != req.KonfirmasiKataSandi {
		return nil, ErrPasswordMismatch
	}

	req.NamaPengguna = strings.TrimSpace(req.NamaPengguna)
	req.Email = strings.TrimSpace(req.Email)

	exists, err := s.repo.ExistsByUsername(ctx, req.NamaPengguna)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrUserAlreadyExists
	}

	exists, err = s.repo.ExistsByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrUserAlreadyExists
	}

	hash, err := bcrypt.GenerateFromPassword(
		[]byte(req.KataSandi),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return nil, err
	}

	user := &entity.User{
		NamaPengguna:  req.NamaPengguna,
		Email:         req.Email,
		KataSandiHash: string(hash),
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *AuthService) Masuk(
	ctx context.Context,
	req dto.MasukRequest,
) (*entity.User, error) {
	user, err := s.repo.FindByUsername(
		ctx,
		strings.TrimSpace(req.NamaPengguna),
	)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	err = bcrypt.CompareHashAndPassword(
		[]byte(user.KataSandiHash),
		[]byte(req.KataSandi),
	)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	return user, nil
}

func (s *AuthService) LupaKataSandi(
	ctx context.Context,
	req dto.LupaKataSandiRequest,
) error {
	_, err := s.repo.FindByEmail(
		ctx,
		strings.TrimSpace(req.Email),
	)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrEmailNotFound
		}
		return err
	}

	// Email ditemukan, tetapi proses pengiriman reset
	// belum diimplementasikan.
	return nil
}

func (s *AuthService) GantiKataSandi(
	ctx context.Context,
	userID uuid.UUID,
	req dto.GantiKataSandiRequest,
) error {
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return err
	}

	err = bcrypt.CompareHashAndPassword(
		[]byte(user.KataSandiHash),
		[]byte(req.KataSandiLama),
	)
	if err != nil {
		return errors.New("kata sandi lama tidak sesuai")
	}

	passwordHash, err := bcrypt.GenerateFromPassword(
		[]byte(req.KataSandiBaru),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return err
	}

	return s.repo.UpdatePassword(ctx, userID, string(passwordHash))
}

func (s *AuthService) RefreshToken(
	ctx context.Context,
	req dto.RefreshTokenRequest,
) (*dto.RefreshTokenResponse, error) {
	claims, err := utils.VerifyToken(
		req.RefreshToken,
		os.Getenv("JWT_REFRESH_SECRET"),
	)
	if err != nil {
		return nil, err
	}

	tokenUse, ok := claims["token_use"].(string)
	if !ok || tokenUse != "refresh" {
		return nil, errors.New("invalid token type")
	}

	sub, ok := claims["sub"].(string)
	if !ok {
		return nil, errors.New("invalid user ID")
	}

	userID, err := uuid.Parse(sub)
	if err != nil {
		return nil, err
	}

	// Confirm that the user still exists.
	user, err := s.repo.FindByID(
		ctx,
		userID,
	)
	if err != nil {
		return nil, err
	}

	accessToken, err := utils.GenerateToken(user.ID, "user")
	if err != nil {
		return nil, err
	}

	refreshToken, err := utils.GenerateRefreshToken(user.ID, "user")
	if err != nil {
		return nil, err
	}

	return &dto.RefreshTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    900,
	}, nil
}
