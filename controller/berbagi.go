package controller

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/theRadn/KI-Tugas-1/dto"
	"github.com/theRadn/KI-Tugas-1/service"
)

type BerbagiController struct {
	svc service.BerbagiService
}

func NewBerbagiController(svc service.BerbagiService) *BerbagiController {
	return &BerbagiController{svc: svc}
}

func userIDDariContext(c *gin.Context) (uuid.UUID, bool) {
	raw := c.GetString("user_id")
	if raw == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return uuid.Nil, false
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id"})
		return uuid.Nil, false
	}
	return id, true
}

func mapBerbagiError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrBerbagiTidakDitemukan):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrBerbagiTidakDiizinkan),
		errors.Is(err, service.ErrBerbagiBerkasSalah),
		errors.Is(err, service.ErrBerbagiDiriSendiri):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrBerbagiTidakAktif):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrBerbagiKedaluwarsa):
		c.JSON(http.StatusGone, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrBerbagiCakupanSalah),
		errors.Is(err, service.ErrBerbagiDurasiSalah):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

// POST /api/berbagi
func (h *BerbagiController) Buat(c *gin.Context) {
	userID, ok := userIDDariContext(c)
	if !ok {
		return
	}
	var req dto.BuatPermintaanBerbagiRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.svc.BuatPermintaan(c.Request.Context(), userID, req)
	if err != nil {
		mapBerbagiError(c, err)
		return
	}
	c.JSON(http.StatusCreated, resp)
}

// GET /api/berbagi/masuk
func (h *BerbagiController) DaftarMasuk(c *gin.Context) {
	userID, ok := userIDDariContext(c)
	if !ok {
		return
	}
	list, err := h.svc.DaftarMasuk(c.Request.Context(), userID)
	if err != nil {
		mapBerbagiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"permintaan": list})
}

// GET /api/berbagi/keluar
func (h *BerbagiController) DaftarKeluar(c *gin.Context) {
	userID, ok := userIDDariContext(c)
	if !ok {
		return
	}
	list, err := h.svc.DaftarKeluar(c.Request.Context(), userID)
	if err != nil {
		mapBerbagiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"permintaan": list})
}

// GET /api/berbagi/:id
func (h *BerbagiController) Detail(c *gin.Context) {
	userID, ok := userIDDariContext(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id tidak valid"})
		return
	}
	resp, err := h.svc.Detail(c.Request.Context(), userID, id)
	if err != nil {
		mapBerbagiError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// POST /api/berbagi/:id/jawab
func (h *BerbagiController) Jawab(c *gin.Context) {
	userID, ok := userIDDariContext(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id tidak valid"})
		return
	}
	var req dto.JawabPermintaanBerbagiRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.Jawab(c.Request.Context(), userID, id, req); err != nil {
		mapBerbagiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// POST /api/berbagi/:id/cabut
func (h *BerbagiController) Cabut(c *gin.Context) {
	userID, ok := userIDDariContext(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id tidak valid"})
		return
	}
	if err := h.svc.Cabut(c.Request.Context(), userID, id); err != nil {
		mapBerbagiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "dicabut"})
}

// GET /api/berbagi/:id/biodata/:tab
func (h *BerbagiController) LihatBiodata(c *gin.Context) {
	userID, ok := userIDDariContext(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id tidak valid"})
		return
	}
	data, err := h.svc.LihatBiodata(c.Request.Context(), userID, id, c.Param("tab"))
	if err != nil {
		mapBerbagiError(c, err)
		return
	}
	c.JSON(http.StatusOK, data)
}

// GET /api/berbagi/:id/berkas/:berkasID
func (h *BerbagiController) LihatBerkas(c *gin.Context) {
	userID, ok := userIDDariContext(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id tidak valid"})
		return
	}
	berkasID, err := uuid.Parse(c.Param("berkasID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "berkas id tidak valid"})
		return
	}
	resp, err := h.svc.LihatBerkas(c.Request.Context(), userID, id, berkasID)
	if err != nil {
		mapBerbagiError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// GET /api/berbagi/:id/berkas/:berkasID/chunk/:index
func (h *BerbagiController) LihatChunk(c *gin.Context) {
	userID, ok := userIDDariContext(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id tidak valid"})
		return
	}
	berkasID, err := uuid.Parse(c.Param("berkasID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "berkas id tidak valid"})
		return
	}
	var index int
	if _, err := fmt.Sscanf(c.Param("index"), "%d", &index); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "index tidak valid"})
		return
	}
	resp, err := h.svc.LihatChunk(c.Request.Context(), userID, id, berkasID, index)
	if err != nil {
		mapBerbagiError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}
