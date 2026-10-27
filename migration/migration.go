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
	if db.Migrator().HasIndex(&entity.Berkas{}, "idx_biodata_kode_berkas") {
		if err := db.Migrator().DropIndex(&entity.Berkas{}, "idx_biodata_kode_berkas"); err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
	}

	// Tabel yang tidak dipakai lagi: examples (sisa template) dan dokumen
	// (rancangan lama, dokumen kini disimpan di tabel berkas).
	for _, tabel := range []string{"examples", "dokumen"} {
		if db.Migrator().HasTable(tabel) {
			if err := db.Migrator().DropTable(tabel); err != nil {
				log.Fatalf("Migration failed: %v", err)
			}
		}
	}

	err := db.AutoMigrate(
		&entity.User{},
		&entity.Biodata{},
		&entity.Pendidikan{},
		&entity.NilaiTKA{},
		&entity.BiodataKeluarga{},
		&entity.Rumah{},
		&entity.CatatanKhusus{},
		&entity.PendukungEkonomi{},
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
		&entity.PendukungEkonomi{},
		&entity.CatatanKhusus{},
		&entity.Rumah{},
		&entity.BiodataKeluarga{},
		&entity.NilaiTKA{},
		&entity.Pendidikan{},
		&entity.Biodata{},
		&entity.User{},
	)
	if err != nil {
		log.Fatalf("Failed to reset database: %v", err)
	}

	log.Println("Database reset successfully completed!")
}
