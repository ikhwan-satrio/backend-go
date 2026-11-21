package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

type AppConfig struct {
	DBUrl             string
	DBPassword        string
	Port              string
	SessionCookieName string
	PasswordCost      int
	SessionDuration   time.Duration
}

func UseApp() AppConfig {
	// Load file .env jika ada (di Docker/produksi biasanya tidak ada file .env, pakai env OS)
	if err := godotenv.Load(); err != nil {
		log.Println("File .env tidak ditemukan, menggunakan environment variable sistem")
	}

	dbHost := getEnv("DB_HOST", "localhost")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "postgres")
	dbName := getEnv("DB_NAME", "backend-go_db")
	dbPort := getEnv("DB_PORT", "5432")
	dbSSLMode := getEnv("DB_SSLMODE", "disable")

	// Prioritaskan DATABASE_URL / DB_URL jika diisi, jika kosong rakit DSN dari komponen DB_*
	dbURL := getEnv("DATABASE_URL", "")
	if dbURL == "" {
		dbURL = getEnv("DB_URL", "")
	}
	if dbURL == "" {
		dbURL = fmt.Sprintf(
			"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=Asia/Jakarta",
			dbHost, dbUser, dbPassword, dbName, dbPort, dbSSLMode,
		)
	}

	port := getEnv("APP_PORT", ":3000")
	if port != "" && port[0] != ':' {
		port = ":" + port
	}

	passwordCost := bcrypt.DefaultCost
	if rawCost := os.Getenv("PASSWORD_COST"); rawCost != "" {
		if parsed, err := strconv.Atoi(rawCost); err == nil && parsed >= bcrypt.MinCost && parsed <= bcrypt.MaxCost {
			passwordCost = parsed
		}
	}

	sessionDuration := 24 * time.Hour
	if rawDuration := os.Getenv("SESSION_DURATION"); rawDuration != "" {
		if parsed, err := time.ParseDuration(rawDuration); err == nil && parsed > 0 {
			sessionDuration = parsed
		}
	}

	return AppConfig{
		DBUrl:             dbURL,
		DBPassword:        dbPassword,
		Port:              port,
		SessionCookieName: getEnv("SESSION_COOKIE_NAME", "session_id"),
		PasswordCost:      passwordCost,
		SessionDuration:   sessionDuration,
	}
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return fallback
}
