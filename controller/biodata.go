package controller

import (
	"errors"
	"io"
	"net/http"

	"github.com/theRadn/KI-Tugas-1/entity"
	"github.com/theRadn/KI-Tugas-1/service"

	"github.com/gin-gonic/gin"
)

type BiodataController struct {
	service *service.BiodataService
}

func NewBiodataController(
	s *service.BiodataService,
) *BiodataController {
	return &BiodataController{service: s}
}

func (c *BiodataController) GetTab(ctx *gin.Context) {
	// Biodata disimpan dalam tiga versi ciphertext; ?algoritma= memilih
	// versi yang didekripsi (bawaan AES).
	algoritma := ctx.DefaultQuery("algoritma", "AES")
	if algoritma != "AES" && algoritma != "DES" && algoritma != "RC4" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": "algoritma harus AES, DES, atau RC4",
		})
		return
	}

	data, err := c.service.GetTab(
		entity.DenganAlgoritma(ctx.Request.Context(), algoritma),
		ctx.GetString("userID"),
		ctx.Param("tab"),
	)

	if err != nil {
		writeBiodataError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Biodata berhasil diambil",
		"data":    data,
	})
}

func (c *BiodataController) SaveTab(ctx *gin.Context) {
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": "Body request tidak valid",
		})
		return
	}

	err = c.service.SaveTab(
		ctx.Request.Context(),
		ctx.GetString("userID"),
		ctx.Param("tab"),
		body,
	)

	if err != nil {
		writeBiodataError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Tab biodata berhasil disimpan",
	})
}

func writeBiodataError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidUserID),
		errors.Is(err, service.ErrInvalidTab):
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": err.Error(),
		})

	case errors.Is(err, service.ErrBiodataNotFound):
		ctx.JSON(http.StatusNotFound, gin.H{
			"message": err.Error(),
		})

	default:
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": "Gagal memproses biodata",
			"error":   err.Error(),
		})
	}
}