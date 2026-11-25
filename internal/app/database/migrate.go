package database

import (
	"fmt"

	"gorm.io/gorm"
)

func Migrate(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}

	return db.AutoMigrate(
		&User{},
		&Account{},
		&Session{},
	)
}
