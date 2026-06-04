package db

import (
	"log"

	"github.com/jobhaunt/backend/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Init(databasePath string) {
	var err error
	DB, err = gorm.Open(sqlite.Open(databasePath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	err = DB.AutoMigrate(
		&models.User{},
		&models.Job{},
		&models.Application{},
		&models.Resume{},
		&models.SearchQuery{},
	)
	if err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	log.Println("database initialized successfully")
}
