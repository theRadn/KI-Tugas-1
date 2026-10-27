package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/theRadn/KI-Tugas-1/controller"
	"github.com/theRadn/KI-Tugas-1/middleware"
)
// gin.engine untuk bikin route inih
func Analisis(route *gin.RouterGroup, analisis controller.AnalisisController) {
	routes := route.Group("/analisis", middleware.AuthMiddleware()) // login check
	routes.GET("/berkas", analisis.DaftarBerkas) // ini endpoint untuk dapetin daftar berkas yang udah diupload sama user
	routes.POST("/uji-dekripsi", analisis.UjiDekripsi) // endpoint, user bisa upload berkas terenkripsi dan kunci, terus server bakal coba dekripsi pake kunci itu
}
