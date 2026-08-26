package config

import (
	"log"

	"github.com/joho/godotenv"
)

func NewConfig() {
	err := godotenv.Load(".env")

	if err != nil {
		log.Fatal("Tidak Valid .env File")
	}
}
