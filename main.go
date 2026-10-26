package main

import (
	"flag"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv" // untuk load file .env
	"github.com/theRadn/KI-Tugas-1/config"
	"github.com/theRadn/KI-Tugas-1/controller"
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

	// Set up Gin router and routes
	router := gin.Default()
	routes.Beranda(router, controller.NewBerandaController(service.NewBerandaService(db)))
	routes.Analisis(router, controller.NewAnalisisController(service.NewAnalisisService(db)))
	log.Fatal(router.Run(":8080"))
}

func runMigrations(db *gorm.DB) {
	migration.Migrate(db)
}

func runSeeding(db *gorm.DB) {
	seeding.ExampleSeed(db)
}
