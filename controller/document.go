package controller

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/theRadn/KI-Tugas-1/service"
)

type DocumentController struct {
	service *service.DocumentService
}

func NewDocumentController(
	documentService *service.DocumentService,
) *DocumentController {
	return &DocumentController{service: documentService}
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

	id, count, err := d.service.Start(
		userID,
		req.Filename,
		req.SizeBytes,
	)
	if err != nil {
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

	if err := d.service.PutChunk(
		userID,
		documentID,
		index,
		req.Data,
	); err != nil {
		documentError(c, err)
		return
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
	case errors.Is(err, service.ErrUnsupportedDocumentType):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "pemrosesan dokumen gagal",
		})
	}
}
