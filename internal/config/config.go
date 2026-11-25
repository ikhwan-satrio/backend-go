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
	DBUrl              string
	DBPassword         string
	Port               string
	SessionCookieName  string
	PasswordCost       int
	SessionDuration    time.Duration
	IsHTTPS            bool
	FrontendURL        string
	GitHubClientID     string
	GitHubClientSecret string
	GitHubRedirectURL  string
}

func UseApp() AppConfig {
	// Load .env file if present (in Docker/production there is usually no .env file, OS env is used)
	if err := godotenv.Load(); err != nil {
		log.Println(".env file not found, using system environment variables")
	}

	dbHost := getEnv("DB_HOST", "localhost")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "postgres")
	dbName := getEnv("DB_NAME", "backend-go_db")
	dbPort := getEnv("DB_PORT", "5432")
	dbSSLMode := getEnv("DB_SSLMODE", "disable")

	dbURL := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=Asia/Jakarta",
		dbHost, dbUser, dbPassword, dbName, dbPort, dbSSLMode,
	)

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
		DBUrl:              dbURL,
		DBPassword:         dbPassword,
		Port:               port,
		SessionCookieName:  getEnv("SESSION_COOKIE_NAME", "session_id"),
		PasswordCost:       passwordCost,
		SessionDuration:    sessionDuration,
		IsHTTPS:            getEnvBool("IS_HTTPS", false),
		FrontendURL:        getEnv("FRONTEND_URL", ""),
		GitHubClientID:     getEnv("GITHUB_CLIENT_ID", ""),
		GitHubClientSecret: getEnv("GITHUB_CLIENT_SECRET", ""),
		GitHubRedirectURL:  getEnv("GITHUB_REDIRECT_URL", ""),
	}
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	raw, ok := os.LookupEnv(key)
	if !ok || raw == "" {
		return fallback
	}

	val, err := strconv.ParseBool(raw)
	if err != nil {
		log.Printf("invalid value for %s=%q, using default %t", key, raw, fallback)
		return fallback
	}
	return val
}
