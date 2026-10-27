
package routes

import (
	"KI-Tugas-1/controller"
	"KI-Tugas-1/repository"
	"KI-Tugas-1/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterAuthRoutes(router *gin.RouterGroup, db *gorm.DB) {
	authRepository := repository.NewAuthRepository(db)
	authService := service.NewAuthService(authRepository)
	authController := controller.NewAuthController(authService)

	auth := router.Group("/auth")
	{
		auth.POST("/daftar", authController.Daftar)
		auth.POST("/masuk", authController.Masuk)
		auth.POST("/keluar", authController.Keluar)
		auth.POST("/lupa_katasandi", authController.LupaKataSandi)
		auth.PUT("/ganti-katasandi", authController.GantiKataSandi)
	}
}
