package config

import (
	"os"
)

type Config struct {
	Port          string
	DatabaseURL   string
	MigrationPath string
}

func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/catalog_db?sslmode=disable"
	}

	migrationPath := os.Getenv("MIGRATION_PATH")
	if migrationPath == "" {
		migrationPath = "./internal/repository/postgres/migrations/001_init.sql"
	}

	return &Config{
		Port:          port,
		DatabaseURL:   dbURL,
		MigrationPath: migrationPath,
	}
}
