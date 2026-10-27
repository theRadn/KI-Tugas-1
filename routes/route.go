package routes

import (
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/theRadn/KI-Tugas-1/controller"
	"github.com/theRadn/KI-Tugas-1/middleware"
)

type RateLimiters struct {
	Global    *middleware.IPRateLimiter
	Login     *middleware.IPRateLimiter
	Register  *middleware.IPRateLimiter
	Refresh   *middleware.IPRateLimiter
	Handshake *middleware.IPRateLimiter
	// Potongan video dan dokumen: satu berkas 50 MB sudah 100 permintaan,
	// jadi rute potongan memakai kuota sendiri dan dilewati limiter global.
	Potongan *middleware.IPRateLimiter
}

func SetupRoutes(router *gin.Engine, db *gorm.DB, sessionManager *middleware.SessionManager, handshakeController *controller.HandshakeController, berandaController controller.BerandaController, analisisController controller.AnalisisController, videoController *controller.VideoController, documentController *controller.DocumentController, limiters RateLimiters) {
	global := limiters.Global.Middleware()
	router.Use(func(c *gin.Context) {
		if strings.Contains(c.FullPath(), "/chunks/") {
			c.Next()
			return
		}
		global(c)
	})
	potongan := limiters.Potongan.Middleware()

	api := router.Group("/api")
	api.POST("/handshake",
		limiters.Handshake.Middleware(),
		handshakeController.Handshake,
	)

	encrypted := api.Group("")
	encrypted.Use(middleware.Encryption(sessionManager))
	{
		RegisterAuthRoutes(encrypted, db, limiters.Login, limiters.Register, limiters.Refresh)
		RegisterBiodataRoutes(encrypted, db)
		RegisterBerkasRoutes(encrypted, db)
		Beranda(encrypted, berandaController)
		Analisis(encrypted, analisisController)

		// Video diunggah dan diunduh per potongan (maks 512 KB) agar muat
		// di amplop enkripsi in-transit.
		videos := encrypted.Group("/videos", middleware.AuthMiddleware())
		videos.POST("", videoController.Start)
		videos.PUT("/:id/chunks/:index", potongan, videoController.PutChunk)
		videos.GET("/:id/chunks/:index", potongan, videoController.GetChunk)
		videos.GET("/:id/chunks/:index/ciphertext", potongan, videoController.GetCiphertext)

		// Dokumen diunggah dan diunduh per potongan (maks 512 KB) agar muat
		// di amplop enkripsi in-transit.
		documents := encrypted.Group("/documents", middleware.AuthMiddleware())
		documents.POST("", documentController.Start)
		documents.PUT("/:id/chunks/:index", potongan, documentController.PutChunk)
		documents.GET("/:id/chunks/:index", potongan, documentController.GetChunk)
		documents.GET("/:id/chunks/:index/ciphertext", potongan, documentController.GetCiphertext)
	}
}
