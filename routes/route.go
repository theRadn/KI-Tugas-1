package routes

import (
    "log"
    "net/http"

    "github.com/gin-gonic/gin"

    "github.com/theRadn/KI-Tugas-1/controller"
    "github.com/theRadn/KI-Tugas-1/middleware"
)

func SetupRoutes(router *gin.Engine, sessionManager *middleware.SessionManager, handshakeController *controller.HandshakeController) {
    router.POST("/api/handshake", handshakeController.Handshake)

    api := router.Group("/api")
    api.Use(middleware.Encryption(sessionManager))
    api.POST("/hello", hello)

}


func hello(c *gin.Context) {
    var req struct {
        Message string `json:"message"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
        return
    }
    log.Printf("[hello] plaintext received from client: %s", req.Message)
    c.JSON(http.StatusOK, gin.H{"reply": "hi"})
}
