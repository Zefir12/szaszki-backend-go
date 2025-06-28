package config

import (
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

type Config struct {
	WS_PORT   string
	GRPC_PORT string
}

var AppConfig Config

func Load() {
	paths := []string{
		".env",
		filepath.Join("..", "..", ".env"), // from /cmd/server
	}

	var loaded bool
	for _, path := range paths {
		if err := godotenv.Load(path); err == nil {
			log.Println("Loaded env from", path)
			loaded = true
			break
		}
	}

	if !loaded {
		log.Println("No .env file found.")
	}

	AppConfig = Config{
		WS_PORT:   os.Getenv("WS_PORT"),
		GRPC_PORT: os.Getenv("GRPC_PORT"),
	}
}
