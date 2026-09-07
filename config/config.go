package config

import (
	"log"

	"github.com/joho/godotenv"
)

func NewConfig() {
	err := godotenv.Load(".env")

	if err != nil {
		log.Println("No .env file found, reading from environment variables")
	}
}
