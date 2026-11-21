package database

import (
	"backend-go/internal/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func InitDB() (*gorm.DB, error) {
	cfg := config.UseApp()
	db, err := gorm.Open(postgres.Open(cfg.DBUrl), &gorm.Config{})

	if err != nil {
		return nil, err
	}

	return db, nil
}
