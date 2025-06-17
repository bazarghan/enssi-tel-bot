package database

import (
	"fmt"
	"strconv"

	"github.com/2000ostd/enssi-tel-bot/internal/platform/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func ConnectDB(cfg config.Database) (*gorm.DB, error) {
	// Validate required fields
	if cfg.Host == "" || cfg.User == "" || cfg.Password == "" || cfg.Name == "" || cfg.Port == 0 {
		return nil, fmt.Errorf("database configuration is incomplete")
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=UTC",
		cfg.Host,
		strconv.Itoa(cfg.Port), // Convert int port to string
		cfg.User,
		cfg.Password,
		cfg.Name,
		cfg.SSLMode,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	return db, nil
}

