package routes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/theRadn/KI-Tugas-1/controller"
	"github.com/theRadn/KI-Tugas-1/middleware"
)

type RateLimiters struct {
	Global    *middleware.IPRateLimiter
	Login     *middleware.IPRateLimiter
	Register  *middleware.IPRateLimiter
	Handshake *middleware.IPRateLimiter
}

func SetupRoutes(router *gin.Engine, db *gorm.DB, sessionManager *middleware.SessionManager, handshakeController *controller.HandshakeController, berandaController controller.BerandaController, analisisController controller.AnalisisController, limiters RateLimiters) {
	router.Use(limiters.Global.Middleware())

	api := router.Group("/api")
	api.POST("/handshake",
		limiters.Handshake.Middleware(),
		handshakeController.Handshake,
	)

	encrypted := api.Group("")
	encrypted.Use(middleware.Encryption(sessionManager))
	{
		RegisterAuthRoutes(encrypted, db, limiters.Login, limiters.Register)
		RegisterBiodataRoutes(encrypted, db)
		RegisterBerkasRoutes(encrypted, db)
		Beranda(encrypted, berandaController)
		Analisis(encrypted, analisisController)
	}
}
