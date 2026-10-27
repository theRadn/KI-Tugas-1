
package routes

import (
	"log"
	"time"

	"github.com/theRadn/KI-Tugas-1/controller"
	"github.com/theRadn/KI-Tugas-1/middleware"
	"github.com/theRadn/KI-Tugas-1/repository"
	"github.com/theRadn/KI-Tugas-1/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterAuthRoutes(router *gin.RouterGroup, db *gorm.DB) {
	authRepository := repository.NewAuthRepository(db)
	authService := service.NewAuthService(authRepository)
	authController := controller.NewAuthController(authService)

	// Maksimal 10 percobaan masuk per menit per alamat IP.
	limiter, err := middleware.NewIPRateLimiter(10, 10, time.Minute, 10*time.Minute, time.Minute)
	if err != nil {
		log.Fatal(err)
	}

	auth := router.Group("/auth")
	{
		auth.POST("/daftar", authController.Daftar)
		auth.POST("/masuk", limiter.Middleware(), authController.Masuk)
		auth.POST("/keluar", authController.Keluar)
		auth.POST("/lupa_katasandi", authController.LupaKataSandi)
		auth.PUT("/ganti-katasandi", middleware.AuthMiddleware(), authController.GantiKataSandi)
	}
}
