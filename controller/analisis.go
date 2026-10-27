package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/theRadn/KI-Tugas-1/dto"
	"github.com/theRadn/KI-Tugas-1/service"
	"gorm.io/gorm"
)

type AnalisisController interface {
	DaftarBerkas(ctx *gin.Context)
	UjiDekripsi(ctx *gin.Context)
}

type analisisController struct {
	service service.AnalisisService
}

func NewAnalisisController(service service.AnalisisService) AnalisisController {
	return &analisisController{service: service}
}

func (c *analisisController) DaftarBerkas(ctx *gin.Context) {
	userID, err := uuid.Parse(ctx.GetString("userID"))
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "pengguna tidak valid"})
		return
	}
	result, err := c.service.DaftarBerkas(ctx.Request.Context(), userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "gagal memuat berkas"})
		return
	}
	ctx.JSON(http.StatusOK, result)
}

func (c *analisisController) UjiDekripsi(ctx *gin.Context) {
	userID, err := uuid.Parse(ctx.GetString("userID"))
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "pengguna tidak valid"})
		return
	}
	var req dto.UjiDekripsiRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "berkas_id dan jumlah_percobaan (1-100) diperlukan"})
		return
	}
	if _, err := uuid.Parse(req.BerkasID); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "berkas_id tidak valid"})
		return
	}
	result, err := c.service.UjiDekripsi(ctx.Request.Context(), userID, req)
	if err != nil {
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			ctx.JSON(http.StatusNotFound, gin.H{"error": "berkas tidak ditemukan"})
		case errors.Is(err, service.BerkasUnready):
			ctx.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "uji dekripsi gagal"})
		}
		return
	}
	ctx.JSON(http.StatusOK, result)
}
