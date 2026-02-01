package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Host           string
	Port           int
	MaxConnections int
}

func LoadConfig() *Config {
	_ = godotenv.Load()

	host := os.Getenv("HOST")

	if host == "" {
		host = "localhost"
	}

	port, err := strconv.Atoi(os.Getenv("PORT"))

	if err != nil || port <= 0 || port > 65535 {
		port = 8080
	}

	maxConnections, err := strconv.Atoi(os.Getenv("MAX_CONNECTIONS"))

	if err != nil || maxConnections <= 0 {
		maxConnections = 50
	}

	return &Config{
		Host:           host,
		Port:           port,
		MaxConnections: maxConnections,
	}

}
