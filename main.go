package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/theRadn/KI-Tugas-1/config"
	"github.com/theRadn/KI-Tugas-1/controller"
	"github.com/theRadn/KI-Tugas-1/middleware"
	"github.com/theRadn/KI-Tugas-1/migration"
	"github.com/theRadn/KI-Tugas-1/seeding"
	"github.com/theRadn/KI-Tugas-1/service"
	"gorm.io/gorm"
)

func main() {
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

	handshakeService := service.NewHandshakeService(
		sessionManager,
		15*time.Minute,
	)

	handshakeController := controller.NewHandshakeController(
		handshakeService,
	)

	fmt.Println(handshakeController)

	fmt.Println("Hello, World!")
}

func runMigrations(db *gorm.DB) {
	migration.Migrate(db)
}

func runSeeding(db *gorm.DB) {
	seeding.ExampleSeed(db)
}
