package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/theRadn/KI-Tugas-1/controller"
	"github.com/theRadn/KI-Tugas-1/middleware"
	"github.com/theRadn/KI-Tugas-1/service"
	"gorm.io/gorm"
)

func RegisterBerbagiRoutes(r *gin.RouterGroup, db *gorm.DB) {
	berbagiHandler := controller.NewBerbagiController(service.NewBerbagiService(db))

	g := r.Group("/berbagi", middleware.AuthMiddleware())
	g.POST("", berbagiHandler.Buat)
	g.GET("/masuk", berbagiHandler.DaftarMasuk)
	g.GET("/keluar", berbagiHandler.DaftarKeluar)
	g.POST("/kirim-kunci", berbagiHandler.KirimKunci)
	g.GET("/:id", berbagiHandler.Detail)
	g.POST("/:id/jawab", berbagiHandler.Jawab)
	g.POST("/:id/cabut", berbagiHandler.Cabut)
	g.GET("/:id/biodata/:tab", berbagiHandler.LihatBiodata)
	g.GET("/:id/berkas/:berkasID", berbagiHandler.LihatBerkas)
	g.GET("/:id/berkas/:berkasID/chunk/:index", berbagiHandler.LihatChunk)
}
