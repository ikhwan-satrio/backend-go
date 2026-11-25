package auth

import (
	"backend-go/internal/app/database"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"gorm.io/gorm"
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

func SetSessionCookie(w http.ResponseWriter, name, rawToken string, expiresAt time.Time, isHTTPS bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    rawToken,
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
		HttpOnly: true, // Not readable by JS (prevents theft via XSS)
		SameSite: http.SameSiteLaxMode,
		Secure:   isHTTPS, // Set to true in production (HTTPS)
	})
}

func ClearSessionCookie(w http.ResponseWriter, name string, isHTTPS bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   isHTTPS,
	})
}

type SessionService struct {
	DB         *gorm.DB
	CookieName string
	SessionTTL time.Duration
	IsHTTPS    bool
}

func (s *SessionService) CreateSession(ctx context.Context, w http.ResponseWriter, userID uint) (*database.Session, error) {
	rawToken, tokenHash, err := GenerateSessionToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session token: %w", err)
	}

	sess := &database.Session{
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(s.SessionTTL),
	}
	if err := s.DB.WithContext(ctx).Create(sess).Error; err != nil {
		return nil, fmt.Errorf("failed to save session to database: %w", err)
	}

	SetSessionCookie(w, s.CookieName, rawToken, sess.ExpiresAt, s.IsHTTPS)
	return sess, nil
}

func (s *SessionService) DeleteSession(ctx context.Context, w http.ResponseWriter, rawToken string) error {
	if rawToken != "" {
		tokenHash := HashSessionToken(rawToken)
		if err := s.DB.WithContext(ctx).Where("token_hash = ?", tokenHash).Delete(&database.Session{}).Error; err != nil {
			return fmt.Errorf("failed to delete session: %w", err)
		}
	}

	ClearSessionCookie(w, s.CookieName, s.IsHTTPS)
	return nil
}
