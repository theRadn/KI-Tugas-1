package routes

import (
	"github.com/theRadn/KI-Tugas-1/controller"
	"github.com/theRadn/KI-Tugas-1/repository"
	"github.com/theRadn/KI-Tugas-1/service"

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