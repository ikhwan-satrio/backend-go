package database

import (
	"backend-go/internal/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func InitDB(cfg config.AppConfig) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(cfg.DBUrl), &gorm.Config{})
}
