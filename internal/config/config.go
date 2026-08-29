package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port          string
	GinMode       string
	SessionSecret string
	AdminUsername string
	AdminPassword string
}

var App *Config

func Load() {
	// .env dosyasını yükle (yoksa sistem ortam değişkenlerini kullan)
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	App = &Config{
		Port:          getEnv("PORT", "8080"),
		GinMode:       getEnv("GIN_MODE", "debug"),
		SessionSecret: getEnv("SESSION_SECRET", "gowatch-secret-key-change-me"),
		AdminUsername: getEnv("ADMIN_USERNAME", "admin"),
		AdminPassword: getEnv("ADMIN_PASSWORD", "admin123"),
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
