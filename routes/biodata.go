package routes

import (
	"KI-Tugas-1/controller"
	"KI-Tugas-1/repository"
	"KI-Tugas-1/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterBiodataRoutes(
	router *gin.RouterGroup,
	db *gorm.DB,
) {
	repo := repository.NewBiodataRepository(db)
	svc := service.NewBiodataService(repo)
	ctrl := controller.NewBiodataController(svc)

	group := router.Group("/biodata")

	group.GET("/:tab", ctrl.GetTab)
	group.PUT("/:tab", ctrl.SaveTab)
}