package routes

import (
	"github.com/theRadn/KI-Tugas-1/controller"
	"github.com/theRadn/KI-Tugas-1/middleware"
	"github.com/theRadn/KI-Tugas-1/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterBerkasRoutes(router *gin.RouterGroup, db *gorm.DB) {
	ctrl := controller.NewBerkasController(service.NewBerkasService(db))

	group := router.Group("/berkas", middleware.AuthMiddleware())
	group.GET("", ctrl.Daftar)
	group.POST("/:kode", ctrl.Unggah)
	group.GET("/:id/unduh", ctrl.Unduh)
	group.GET("/:id/ciphertext", ctrl.Ciphertext)
	group.DELETE("/:id", ctrl.Hapus)
}
