package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/theRadn/KI-Tugas-1/dto"
	"github.com/theRadn/KI-Tugas-1/service"
)

type BerkasController struct {
	service *service.BerkasService
}

func NewBerkasController(s *service.BerkasService) *BerkasController {
	return &BerkasController{service: s}
}

func penggunaBerkas(ctx *gin.Context) (uuid.UUID, bool) {
	userID, err := uuid.Parse(ctx.GetString("userID"))
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"message": "Pengguna tidak valid"})
		return uuid.Nil, false
	}
	return userID, true
}

func tulisGalatBerkas(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrBerkasNotFound):
		ctx.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
	case errors.Is(err, service.ErrKodeBerkas),
		errors.Is(err, service.ErrBerkasTidakSah),
		errors.Is(err, service.ErrAlgoritma):
		ctx.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
	case errors.Is(err, service.ErrKunciBerkas):
		ctx.JSON(http.StatusServiceUnavailable, gin.H{"message": err.Error()})
	default:
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal memproses berkas"})
	}
}

func (c *BerkasController) Daftar(ctx *gin.Context) {
	userID, ok := penggunaBerkas(ctx)
	if !ok {
		return
	}
	res, err := c.service.Daftar(ctx.Request.Context(), userID)
	if err != nil {
		tulisGalatBerkas(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"data": res})
}

func (c *BerkasController) Unggah(ctx *gin.Context) {
	userID, ok := penggunaBerkas(ctx)
	if !ok {
		return
	}
	var req dto.UnggahBerkasRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "Data berkas tidak valid"})
		return
	}
	res, err := c.service.Unggah(ctx.Request.Context(), userID, ctx.Param("kode"), req)
	if err != nil {
		tulisGalatBerkas(ctx, err)
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{"message": "Berkas terenkripsi dan tersimpan", "data": res})
}

func (c *BerkasController) Unduh(ctx *gin.Context) {
	userID, ok := penggunaBerkas(ctx)
	if !ok {
		return
	}
	res, err := c.service.Unduh(ctx.Request.Context(), userID, ctx.Param("id"), ctx.Query("algoritma"))
	if err != nil {
		tulisGalatBerkas(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"data": res})
}

func (c *BerkasController) Ciphertext(ctx *gin.Context) {
	userID, ok := penggunaBerkas(ctx)
	if !ok {
		return
	}
	res, err := c.service.Ciphertext(ctx.Request.Context(), userID, ctx.Param("id"), ctx.Query("algoritma"))
	if err != nil {
		tulisGalatBerkas(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"data": res})
}

func (c *BerkasController) Hapus(ctx *gin.Context) {
	userID, ok := penggunaBerkas(ctx)
	if !ok {
		return
	}
	if err := c.service.Hapus(ctx.Request.Context(), userID, ctx.Param("id")); err != nil {
		tulisGalatBerkas(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "Berkas dihapus"})
}
