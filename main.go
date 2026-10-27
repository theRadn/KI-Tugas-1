package main

import (
	"flag"
	"log"
	"os"
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
	if err := godotenv.Load(); err != nil {
		log.Println("File .env tidak ditemukan, memakai variabel lingkungan sistem")
	}

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
	videoController := controller.NewVideoController(service.NewVideoService("storage/videos"))

	globalLimiter, err := middleware.NewIPRateLimiter(
		100, 100, time.Minute,
		10*time.Minute, time.Minute,
	)
	if err != nil {
		log.Fatal(err)
	}
	defer globalLimiter.Close()

	loginLimiter, err := middleware.NewIPRateLimiter(
		5, 5, time.Minute,
		10*time.Minute, time.Minute,
	)
	if err != nil {
		log.Fatal(err)
	}
	defer loginLimiter.Close()

	registerLimiter, err := middleware.NewIPRateLimiter(
		5, 5, time.Minute,
		10*time.Minute, time.Minute,
	)
	if err != nil {
		log.Fatal(err)
	}
	defer registerLimiter.Close()

	handshakeLimiter, err := middleware.NewIPRateLimiter(
		5, 1, time.Minute,
		5*time.Minute,
		time.Minute,
	)
	if err != nil {
		log.Fatal(err)
	}
	defer handshakeLimiter.Close()

	limiters := routes.RateLimiters{
		Global:    globalLimiter,
		Login:     loginLimiter,
		Register:  registerLimiter,
		Handshake: handshakeLimiter,
	}

	router := gin.Default()
	router.Use(middleware.CORSMiddleware())

	routes.SetupRoutes(
		router,
		db,
		sessionManager,
		handshakeController,
		berandaController,
		analisisController,
		videoController,
		limiters,
	)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Fatal(router.Run(":" + port))
}

func runMigrations(db *gorm.DB) {
	migration.Migrate(db)
}

func runSeeding(db *gorm.DB) {
	seeding.ExampleSeed(db)
}
