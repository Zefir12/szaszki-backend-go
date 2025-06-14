package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	POSTGRES_URI string
	WS_PORT      string
}

var AppConfig Config

func Load() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found.")
	}

	AppConfig = Config{
		POSTGRES_URI: os.Getenv("POSTGRES_URI"),
		WS_PORT:      os.Getenv("WS_PORT"),
	}
}
