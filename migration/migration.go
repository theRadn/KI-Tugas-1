package migration

import (
	"log"

	"github.com/theRadn/KI-Tugas-1/entity"
	"gorm.io/gorm"
)

func Migrate(db *gorm.DB) {
	log.Println("Migration started...")

	// Indeks lama hanya memuat kolom kode, sehingga kode berkas unik untuk
	// seluruh aplikasi dan pengguna kedua tidak bisa mengunggah kode yang sama.
	for _, lama := range []struct {
		model any
		nama  string
	}{
		{&entity.Berkas{}, "idx_biodata_kode_berkas"},
		{&entity.Dokumen{}, "idx_biodata_dokumen"},
	} {
		if db.Migrator().HasIndex(lama.model, lama.nama) {
			if err := db.Migrator().DropIndex(lama.model, lama.nama); err != nil {
				log.Fatalf("Migration failed: %v", err)
			}
		}
	}

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
