package database

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:120;not null" json:"name"`
	Email     string    `gorm:"size:255;uniqueIndex;not null" json:"email"`
	Password  string    `gorm:"type:text" json:"-"`
	AvatarURL string    `gorm:"type:text" json:"avatar_url,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Accounts []Account `gorm:"constraint:OnDelete:CASCADE;" json:"-"`
	Sessions []Session `gorm:"constraint:OnDelete:CASCADE;" json:"-"`
}

type Account struct {
	gorm.Model
	ID                uint       `gorm:"primaryKey" json:"id"`
	UserID            uint       `gorm:"index:idx_user_provider,unique;not null" json:"user_id"`
	Provider          string     `gorm:"size:32;index:idx_user_provider,unique;not null" json:"provider"`
	ProviderAccountID string     `gorm:"size:128;uniqueIndex:idx_provider_account;not null" json:"provider_account_id"`
	AccessToken       string     `gorm:"type:text" json:"-"`
	RefreshToken      string     `gorm:"type:text" json:"-"`
	ExpiresAt         *time.Time `json:"expires_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`

	User User `gorm:"constraint:OnDelete:CASCADE;" json:"-"`
}

type Session struct {
	gorm.Model
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	TokenHash string    `gorm:"size:64;uniqueIndex;not null" json:"-"`
	ExpiresAt time.Time `gorm:"index;not null" json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	User User `gorm:"constraint:OnDelete:CASCADE;" json:"-"`
}
