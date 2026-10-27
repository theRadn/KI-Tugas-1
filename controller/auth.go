package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"errors"
	"net/http"

	// DB
	"github.com/theRadn/KI-Tugas-1/dto"
	"github.com/theRadn/KI-Tugas-1/service"
	"github.com/theRadn/KI-Tugas-1/utils"
)

type AuthController struct {
	service *service.AuthService
}

func NewAuthController(s *service.AuthService) *AuthController {
	return &AuthController{service: s}
}

func (c *AuthController) Daftar(ctx *gin.Context) {
	var req dto.DaftarRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": "Request tidak valid",
			"error":   err.Error(),
		})
		return
	}

	user, err := c.service.Daftar(ctx.Request.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrPasswordMismatch):
			ctx.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		case errors.Is(err, service.ErrUserAlreadyExists):
			ctx.JSON(http.StatusConflict, gin.H{"message": err.Error()})
		default:
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"message": "Gagal membuat akun",
			})
		}
		return
	}

	response := dto.AuthResponse{
		ID:           user.ID.String(),
		NamaPengguna: user.NamaPengguna,
		Email:        user.Email,
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"message": "Akun berhasil dibuat",
		"data":    response,
	})
}

func (c *AuthController) Masuk(ctx *gin.Context) {
	var req dto.MasukRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": "Request tidak valid",
			"error":   err.Error(),
		})
		return
	}

	user, err := c.service.Masuk(ctx.Request.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			ctx.JSON(http.StatusUnauthorized, gin.H{
				"message": "Username atau kata sandi salah",
			})
			return
		}

		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": "Login gagal",
		})
		return
	}

	token, err := utils.GenerateToken(user.ID, "user")
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Login gagal"})
		return
	}

	response := dto.AuthResponse{
		ID:           user.ID.String(),
		NamaPengguna: user.NamaPengguna,
		Email:        user.Email,
		Token:        token,
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Login berhasil",
		"data":    response,
	})
}

func (c *AuthController) Keluar(ctx *gin.Context) {
	// Belum ada session atau token yang diinvalidasi.
	ctx.JSON(http.StatusOK, gin.H{
		"message": "Berhasil keluar",
	})
}

func (c *AuthController) LupaKataSandi(ctx *gin.Context) {
	var req dto.LupaKataSandiRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": "Email tidak valid",
		})
		return
	}

	// Email terdaftar atau tidak dijawab sama, supaya endpoint ini
	// tidak bisa dipakai menebak email mana yang punya akun.
	err := c.service.LupaKataSandi(ctx.Request.Context(), req)
	if err != nil && !errors.Is(err, service.ErrEmailNotFound) {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": "Permintaan gagal diproses",
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Jika email terdaftar, instruksi reset akan dikirim",
	})
}


func (c *AuthController) GantiKataSandi(ctx *gin.Context) {
    var req dto.GantiKataSandiRequest

    if err := ctx.ShouldBindJSON(&req); err != nil {
        ctx.JSON(http.StatusBadRequest, gin.H{
            "message": "Data ganti kata sandi tidak valid",
            "error":   err.Error(),
        })
        return
    }

    // AuthMiddleware menyimpan userID sebagai teks.
    idTeks := ctx.GetString("userID")
    userID, err := uuid.Parse(idTeks)
    if err != nil {
        ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Identitas pengguna tidak valid"})
        return
    }

    err = c.service.GantiKataSandi(
        ctx.Request.Context(),
        userID,
        req,
    )
    if err != nil {
        if err.Error() == "kata sandi lama tidak sesuai" {
            ctx.JSON(http.StatusBadRequest, gin.H{
                "message": err.Error(),
            })
            return
        }

        ctx.JSON(http.StatusInternalServerError, gin.H{
            "message": "Gagal mengganti kata sandi",
        })
        return
    }

    ctx.JSON(http.StatusOK, gin.H{
        "message": "Kata sandi berhasil diperbarui",
    })
}