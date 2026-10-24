package config

import (
	"fmt"
	"os"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func SetUpDatabaseConnection() *gorm.DB {
	if _, err := os.Stat("database"); os.IsNotExist(err) {
		err := os.Mkdir("database", 0755)
		if err != nil {
			fmt.Println(err)
			panic(err)
		}
	}

	database, err := gorm.Open(sqlite.Open("database/db.sqlite3"), &gorm.Config{})
	if err != nil {
		fmt.Println(err)
		panic(err)
	}

	sqlDB, _ := database.DB()
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	return database
}

func CloseDatabaseConnection(db *gorm.DB) {
	dbSQL, err := db.DB()
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	dbSQL.Close()
}
