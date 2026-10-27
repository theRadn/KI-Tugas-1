package routes

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
<<<<<<< HEAD
	"gorm.io/gorm"
=======
>>>>>>> d846f13 (video handling)

	"github.com/theRadn/KI-Tugas-1/controller"
	"github.com/theRadn/KI-Tugas-1/middleware"
)

type RateLimiters struct {
	Global    *middleware.IPRateLimiter
	Login     *middleware.IPRateLimiter
	Register  *middleware.IPRateLimiter
	Handshake *middleware.IPRateLimiter
}

<<<<<<< HEAD
func SetupRoutes(router *gin.Engine, db *gorm.DB, sessionManager *middleware.SessionManager, handshakeController *controller.HandshakeController, berandaController controller.BerandaController, analisisController controller.AnalisisController, limiters RateLimiters) {
=======
func SetupRoutes(router *gin.Engine, sessionManager *middleware.SessionManager, handshakeController *controller.HandshakeController, berandaController controller.BerandaController, analisisController controller.AnalisisController, videoController *controller.VideoController, limiters RateLimiters) {
>>>>>>> d846f13 (video handling)
	router.Use(limiters.Global.Middleware())

	api := router.Group("/api")
	api.POST("/handshake",
		limiters.Handshake.Middleware(),
		handshakeController.Handshake,
	)

	encrypted := api.Group("")
	encrypted.Use(middleware.Encryption(sessionManager))
	{
		encrypted.POST("/hello", hello)
<<<<<<< HEAD
		RegisterAuthRoutes(encrypted, db, limiters.Login, limiters.Register)
		RegisterBiodataRoutes(encrypted, db)
		RegisterBerkasRoutes(encrypted, db)
		Beranda(encrypted, berandaController)
		Analisis(encrypted, analisisController)
=======
		Beranda(encrypted, berandaController)
		Analisis(encrypted, analisisController)
		videos := encrypted.Group("/videos", middleware.AuthMiddleware())
		videos.POST("", videoController.Start)
		videos.PUT("/:id/chunks/:index", videoController.PutChunk)
		videos.GET("/:id/chunks/:index", videoController.GetChunk)
>>>>>>> d846f13 (video handling)
	}
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
