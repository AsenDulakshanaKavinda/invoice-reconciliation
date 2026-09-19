package config

import (
	"log"

	"github.com/caarlos0/env/v10"
	"github.com/joho/godotenv"
)

type Config struct {
	Port string `env:"PORT" envDefault:"8080"`
	Environment string `env:"ENV" envDefault:"development"`
}

func LoadConfig() *Config {
	// load .env file if precent 
	_ = godotenv.Load()

	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		log.Fatalf("Failed to parse env vars: %v", err)
	}
	return cfg
}