package auth

import (
	"backend-go/internal/config"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"time"
)

func GenerateSessionToken() (rawToken string, tokenHash string, err error) {
	b := make([]byte, 32) // 256-bit random token
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}

	rawToken = base64.URLEncoding.EncodeToString(b)
	tokenHash = HashSessionToken(rawToken)
	return rawToken, tokenHash, nil
}

func HashSessionToken(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:])
}

func SetSessionCookie(w http.ResponseWriter, rawToken string, expiresAt time.Time, isHTTPS bool) {
	cfg := config.UseApp()
	http.SetCookie(w, &http.Cookie{
		Name:     cfg.SessionCookieName,
		Value:    rawToken,
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
		HttpOnly: true, // Tidak bisa dibaca JS (mencegah pencurian via XSS)
		SameSite: http.SameSiteLaxMode,
		Secure:   isHTTPS, // Set true di production (HTTPS)
	})
}
