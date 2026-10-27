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

type DocumentController struct {
	service *service.DocumentService
	berkas  *service.BerkasService
}

func NewDocumentController(
	documentService *service.DocumentService,
	berkasService *service.BerkasService,
) *DocumentController {
	return &DocumentController{service: documentService, berkas: berkasService}
}

func (d *DocumentController) Start(c *gin.Context) {
	userID, ok := videoUserID(c)
	if !ok {
		return
	}

	var req struct {
		Filename  string `json:"filename" binding:"required"`
		SizeBytes int64  `json:"size_bytes" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "filename dan size_bytes diperlukan",
		})
		return
	}

	if err := service.ValidasiDokumen(req.SizeBytes); err != nil {
		documentError(c, err)
		return
	}

	id, count, err := d.service.Start(
		userID,
		req.Filename,
		req.SizeBytes,
	)
	if err != nil {
		documentError(c, err)
		return
	}

	// Dicatat di tabel berkas supaya muncul di daftar Berkas dan Beranda,
	// dan ikut terhapus lewat DELETE /api/berkas/:id.
	folder := d.service.Folder(userID, id)
	nama, tipe, err := d.service.Info(userID, id)
	if err == nil {
		err = d.berkas.CatatDokumen(c.Request.Context(), userID, id, nama, tipe, req.SizeBytes, folder)
	}
	if err != nil {
		_ = os.RemoveAll(folder)
		documentError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"document_id": id.String(),
		"chunk_count": count,
		"chunk_bytes": service.MaxDocumentChunkBytes,
	})
}

func (d *DocumentController) PutChunk(c *gin.Context) {
	userID, documentID, index, ok := documentParams(c)
	if !ok {
		return
	}

	var req struct {
		Data []byte `json:"data" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "data base64 diperlukan",
		})
		return
	}

	if index == 0 {
		if err := d.berkas.CekAwalDokumen(c.Request.Context(), userID, documentID, req.Data); err != nil {
			// Isi tidak sesuai format: buang baris dan foldernya.
			if errors.Is(err, service.ErrBerkasTidakSah) {
				_ = d.berkas.Hapus(c.Request.Context(), userID, documentID.String())
			}
			documentError(c, err)
			return
		}
	}

	if err := d.service.PutChunk(
		userID,
		documentID,
		index,
		req.Data,
	); err != nil {
		documentError(c, err)
		return
	}

	if lengkap, err := d.service.Lengkap(userID, documentID); err == nil && lengkap {
		if err := d.berkas.TandaiDokumenTerunggah(c.Request.Context(), userID, documentID); err != nil {
			documentError(c, err)
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"index": index})
}

func (d *DocumentController) GetChunk(c *gin.Context) {
	userID, documentID, index, ok := documentParams(c)
	if !ok {
		return
	}

	algorithm := c.DefaultQuery("algoritma", "AES")

	chunk, manifest, err := d.service.GetChunk(
		userID,
		documentID,
		index,
		algorithm,
	)
	if err != nil {
		documentError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":         chunk,
		"index":        index,
		"chunk_count":  manifest.ChunkCount,
		"filename":     manifest.Filename,
		"content_type": manifest.ContentType,
		"size_bytes":   manifest.SizeBytes,
	})
}

func (d *DocumentController) GetCiphertext(c *gin.Context) {
	userID, documentID, index, ok := documentParams(c)
	if !ok {
		return
	}
	ciphertext, count, err := d.service.CiphertextChunk(userID, documentID, index, c.DefaultQuery("algoritma", "AES"))
	if err != nil {
		documentError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ciphertext": ciphertext, "index": index, "chunk_count": count})
}

func documentParams(
	c *gin.Context,
) (uuid.UUID, uuid.UUID, int, bool) {
	userID, ok := videoUserID(c)
	if !ok {
		return uuid.Nil, uuid.Nil, 0, false
	}

	documentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "document_id tidak valid",
		})
		return uuid.Nil, uuid.Nil, 0, false
	}

	index, err := strconv.Atoi(c.Param("index"))
	if err != nil || index < 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "index tidak valid",
		})
		return uuid.Nil, uuid.Nil, 0, false
	}

	return userID, documentID, index, true
}

func documentError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrDocumentNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrInvalidDocument):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrUnsupportedDocumentType),
		errors.Is(err, service.ErrBerkasTidakSah):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrBerkasNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "pemrosesan dokumen gagal",
		})
	}
}
