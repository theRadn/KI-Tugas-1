package seeding

import (
	"log"

	"github.com/theRadn/KI-Tugas-1/entity"
	"gorm.io/gorm"
)

func ExampleSeed(db *gorm.DB) {
	log.Println("Seeding started...")

	example := entity.Example{
		Test: "This is a test example",
	}

	if err := db.Create(&example).Error; err != nil {
		log.Fatalf("Seeding failed: %v", err)
	}

	log.Println("Seeding successfully completed!")
}
