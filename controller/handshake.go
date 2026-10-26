package controller

import (
	"encoding/base64"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/theRadn/KI-Tugas-1/service"
)

type HandshakeController struct {
	handshakeService *service.HandshakeService
}

type handshakeRequest struct {
	ClientPublicKey string `json:"client_public_key" binding:"required"`
}

type handshakeResponse struct {
	SessionID       string `json:"session_id"`
	ServerPublicKey string `json:"server_public_key"`
	ExpiresAt       string `json:"expires_at"`
}

func NewHandshakeController(
	handshakeService *service.HandshakeService,
) *HandshakeController {
	return &HandshakeController{
		handshakeService: handshakeService,
	}
}

func (h *HandshakeController) Handshake(c *gin.Context) {
	var req handshakeRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid handshake request",
		})
		return
	}

	clientPublicKey, err := base64.StdEncoding.DecodeString(
		req.ClientPublicKey,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid client public key encoding",
		})
		return
	}

	result, err := h.handshakeService.CreateSession(clientPublicKey)
	if err != nil {
		if errors.Is(err, service.ErrInvalidClientPublicKey) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid client public key",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to establish session",
		})
		return
	}

	c.JSON(http.StatusOK, handshakeResponse{
		SessionID: result.SessionID,
		ServerPublicKey: base64.StdEncoding.EncodeToString(
			result.ServerPublicKey,
		),
		ExpiresAt: result.ExpiresAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	})
}
