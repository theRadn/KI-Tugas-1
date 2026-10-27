package main

import (
	"flag"
	"log"
	"time"

	"github.com/theRadn/KI-Tugas-1/config"
	"github.com/theRadn/KI-Tugas-1/controller"
	"github.com/theRadn/KI-Tugas-1/middleware"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv" // untuk load file .env
	"github.com/theRadn/KI-Tugas-1/migration"
	"github.com/theRadn/KI-Tugas-1/routes"
	"github.com/theRadn/KI-Tugas-1/seeding"
	"github.com/theRadn/KI-Tugas-1/service"
	"gorm.io/gorm"
)

func main() {
	_ = godotenv.Load()
	migrateFlag := flag.Bool("migrate", false, "Run database migrations")
	seedFlag := flag.Bool("seed", false, "Run data seeding")
	flag.Parse()
	hasTaskRun := false

	db := config.SetUpDatabaseConnection()
	defer config.CloseDatabaseConnection(db)

	if *migrateFlag {
		runMigrations(db)
		hasTaskRun = true
	}

	if *seedFlag {
		runSeeding(db)
		hasTaskRun = true
	}

	if hasTaskRun {
		return
	}

	sessionManager := middleware.NewSessionManager(15 * time.Minute)

	handshakeService := service.NewHandshakeService(sessionManager, 15*time.Minute)
	handshakeController := controller.NewHandshakeController(handshakeService)
	berandaService := service.NewBerandaService(db)
	berandaController := controller.NewBerandaController(berandaService)
	analisisService := service.NewAnalisisService(db)
	analisisController := controller.NewAnalisisController(analisisService)

	router := gin.Default()
	router.Use(middleware.CORSMiddleware())

	routes.SetupRoutes(router, sessionManager, handshakeController, berandaController, analisisController)

	log.Fatal(router.Run(":8080"))
}

func runMigrations(db *gorm.DB) {
	migration.Migrate(db)
}

func runSeeding(db *gorm.DB) {
	seeding.ExampleSeed(db)
}
