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
		&entity.User{},
		&entity.Biodata{},
		&entity.Pendidikan{},
		&entity.NilaiTKA{},
		&entity.BiodataKeluarga{},
		&entity.Rumah{},
		&entity.CatatanKhusus{},
		&entity.PendukungEkonomi{},
		&entity.Dokumen{},
		&entity.Berkas{},
		&entity.EnkripsiBerkas{},
		&entity.HasilAnalisis{},
		&entity.HasilAnalisisAlgoritma{},
	)

	if err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	log.Println("Migration successfully completed!")
}

func ResetDatabase(db *gorm.DB) {
	log.Println("Resetting database...")

	err := db.Migrator().DropTable(
		&entity.HasilAnalisisAlgoritma{},
		&entity.HasilAnalisis{},
		&entity.EnkripsiBerkas{},
		&entity.Berkas{},
		&entity.Dokumen{},
		&entity.PendukungEkonomi{},
		&entity.CatatanKhusus{},
		&entity.Rumah{},
		&entity.BiodataKeluarga{},
		&entity.NilaiTKA{},
		&entity.Pendidikan{},
		&entity.Biodata{},
		&entity.User{},
		&entity.Example{},
	)
	if err != nil {
		log.Fatalf("Failed to reset database: %v", err)
	}

	log.Println("Database reset successfully completed!")
}
