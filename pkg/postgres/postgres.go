package postgres

import (
	"fmt"
	"log"
	"os"

	"EquiliLearn/internal/entity"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func StartPostgres() *gorm.DB {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Jakarta",
		os.Getenv("DB_HOST"), os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"), os.Getenv("DB_PORT"))

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Cannot connect to database: %s", err.Error())
	}

	migrate(db)

	return db
}

func migrate(db *gorm.DB) {
	err := db.AutoMigrate(&entity.User{})
	if err != nil {
		log.Fatalf("Failed to run migration: %s", err.Error())
	}
}
