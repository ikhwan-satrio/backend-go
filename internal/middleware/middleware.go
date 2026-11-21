package middleware

import (
	"backend-go/internal/app/database"
	"backend-go/internal/auth"
	"backend-go/internal/config"
	"backend-go/internal/httpx"
	"context"
	"net/http"
	"time"
)

type contextKey string

const userIDContextKey contextKey = "user_id"

func UserIDFromContext(ctx context.Context) (uint, bool) {
	id, ok := ctx.Value(userIDContextKey).(uint)
	return id, ok
}

func RequireSession() func(http.Handler) http.Handler {
	db, _ := database.InitDB()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cfg := config.UseApp()

			cookie, err := r.Cookie(cfg.SessionCookieName)
			if err != nil || cookie.Value == "" {
				httpx.Error(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			tokenHash := auth.HashSessionToken(cookie.Value)

			var sess database.Session
			err = db.WithContext(r.Context()).
				Where("token_hash = ? AND expires_at > ?", tokenHash, time.Now()).
				First(&sess).Error
			if err != nil {
				httpx.Error(w, http.StatusUnauthorized, "invalid or expired session")
				return
			}

			ctx := context.WithValue(r.Context(), userIDContextKey, sess.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
