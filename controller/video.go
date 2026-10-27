package controller

import (
	"errors"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/theRadn/KI-Tugas-1/service"
)

type VideoController struct {
	service *service.VideoService
	berkas  *service.BerkasService
}

func NewVideoController(service *service.VideoService, berkas *service.BerkasService) *VideoController {
	return &VideoController{service: service, berkas: berkas}
}

func (v *VideoController) Start(c *gin.Context) {
	userID, ok := videoUserID(c)
	if !ok {
		return
	}
	var req struct {
		ChunkCount int    `json:"chunk_count" binding:"required"`
		SizeBytes  int64  `json:"size_bytes" binding:"required"`
		NamaAsli   string `json:"nama_asli" binding:"required,max=255"`
		TipeMIME   string `json:"tipe_mime" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "chunk_count, size_bytes, nama_asli, dan tipe_mime diperlukan"})
		return
	}
	if err := service.ValidasiVideo(req.TipeMIME, req.SizeBytes); err != nil {
		videoError(c, err)
		return
	}
	id, err := v.service.Start(userID, req.ChunkCount, req.SizeBytes)
	if err != nil {
		videoError(c, err)
		return
	}
	// Dicatat di tabel berkas supaya muncul di daftar Berkas dan Beranda,
	// dan ikut terhapus lewat DELETE /api/berkas/:id.
	folder := v.service.Folder(userID, id)
	if err := v.berkas.CatatVideo(c.Request.Context(), userID, id, req.NamaAsli, req.TipeMIME, req.SizeBytes, folder); err != nil {
		_ = os.RemoveAll(folder)
		videoError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"video_id": id.String(), "chunk_bytes": service.MaxVideoChunkBytes})
}

func (v *VideoController) PutChunk(c *gin.Context) {
	userID, videoID, index, ok := videoParams(c)
	if !ok {
		return
	}
	var req struct {
		Data []byte `json:"data" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "data base64 diperlukan"})
		return
	}
	if index == 0 {
		if err := v.berkas.CekAwalVideo(c.Request.Context(), userID, videoID, req.Data); err != nil {
			// Isi bukan video: buang baris dan foldernya agar tidak tersisa
			// unggahan "belum selesai".
			if errors.Is(err, service.ErrBerkasTidakSah) {
				_ = v.berkas.Hapus(c.Request.Context(), userID, videoID.String())
			}
			videoError(c, err)
			return
		}
	}
	if err := v.service.PutChunk(userID, videoID, index, req.Data); err != nil {
		videoError(c, err)
		return
	}
	if lengkap, err := v.service.Lengkap(userID, videoID); err == nil && lengkap {
		if err := v.berkas.TandaiVideoTerunggah(c.Request.Context(), userID, videoID); err != nil {
			videoError(c, err)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"index": index})
}

func (v *VideoController) GetChunk(c *gin.Context) {
	userID, videoID, index, ok := videoParams(c)
	if !ok {
		return
	}
	algorithm := c.DefaultQuery("algoritma", "AES")
	chunk, count, err := v.service.GetChunk(userID, videoID, index, algorithm)
	if err != nil {
		videoError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": chunk, "index": index, "chunk_count": count})
}

func (v *VideoController) GetCiphertext(c *gin.Context) {
	userID, videoID, index, ok := videoParams(c)
	if !ok {
		return
	}
	ciphertext, count, err := v.service.CiphertextChunk(userID, videoID, index, c.DefaultQuery("algoritma", "AES"))
	if err != nil {
		videoError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ciphertext": ciphertext, "index": index, "chunk_count": count})
}

func videoUserID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.GetString("userID"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "pengguna tidak valid"})
		return uuid.Nil, false
	}
	return id, true
}

func videoParams(c *gin.Context) (uuid.UUID, uuid.UUID, int, bool) {
	userID, ok := videoUserID(c)
	if !ok {
		return uuid.Nil, uuid.Nil, 0, false
	}
	videoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "video_id tidak valid"})
		return uuid.Nil, uuid.Nil, 0, false
	}
	index, err := strconv.Atoi(c.Param("index"))
	if err != nil || index < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "index tidak valid"})
		return uuid.Nil, uuid.Nil, 0, false
	}
	return userID, videoID, index, true
}

func videoError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrVideoNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrInvalidVideoChunk),
		errors.Is(err, service.ErrBerkasTidakSah):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrBerkasNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "pemrosesan video gagal"})
	}
}
