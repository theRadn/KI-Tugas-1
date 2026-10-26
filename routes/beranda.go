package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/theRadn/KI-Tugas-1/controller"
	"github.com/theRadn/KI-Tugas-1/middleware"
)

func Beranda(route *gin.Engine, beranda controller.BerandaController) {
	// endpoint untuk dapetin data beranda user (status biodata per bagian, jumlah berkas, dan total ukuran berkas)
	route.GET("/api/beranda", middleware.AuthMiddleware(), beranda.Get)
}
