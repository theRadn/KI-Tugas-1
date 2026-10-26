package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/theRadn/KI-Tugas-1/service"
	"gorm.io/gorm"
)

type BerandaController interface { 
	Get(ctx *gin.Context)
}

type berandaController struct {
	service service.BerandaService
}

func NewBerandaController(service service.BerandaService) BerandaController {
	return &berandaController{service: service}
}

// controller untuk dapetin data beranda user (status biodata per bagian, jumlah berkas, dan total ukuran berkas)
func (c *berandaController) Get(ctx *gin.Context) {
	userID, err := uuid.Parse(ctx.GetString("userID"))
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "pengguna tidak valid"})
		return
	}

	result, err := c.service.Get(ctx.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "pengguna tidak ditemukan"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "gagal memuat beranda"})
		return
	}
	ctx.JSON(http.StatusOK, result)
}
