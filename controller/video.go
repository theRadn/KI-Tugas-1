package controller

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/theRadn/KI-Tugas-1/service"
)

type VideoController struct {
	service *service.VideoService
}

func NewVideoController(service *service.VideoService) *VideoController {
	return &VideoController{service: service}
}

func (v *VideoController) Start(c *gin.Context) {
	userID, ok := videoUserID(c)
	if !ok {
		return
	}
	var req struct {
		ChunkCount int   `json:"chunk_count" binding:"required"`
		SizeBytes  int64 `json:"size_bytes" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "chunk_count dan size_bytes diperlukan"})
		return
	}
	id, err := v.service.Start(userID, req.ChunkCount, req.SizeBytes)
	if err != nil {
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
	if err := v.service.PutChunk(userID, videoID, index, req.Data); err != nil {
		videoError(c, err)
		return
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
	case errors.Is(err, service.ErrInvalidVideoChunk):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "pemrosesan video gagal"})
	}
}
