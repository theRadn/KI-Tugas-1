package migration

import (
	"log"

	"github.com/theRadn/KI-Tugas-1/entity"
	"gorm.io/gorm"
)

func Migrate(db *gorm.DB) {
	log.Println("Migration started...")

	err := db.AutoMigrate(
		&entity.Example{},
	)

	if err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	log.Println("Migration successfully completed!")
}

func ResetDatabase(db *gorm.DB) {
	log.Println("Resetting database...")

	err := db.Migrator().DropTable(
		&entity.Example{},
	)
	if err != nil {
		log.Fatalf("Failed to reset database: %v", err)
	}

	log.Println("Database reset successfully completed!")
}
