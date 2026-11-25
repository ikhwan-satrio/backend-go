package controller

import (
	"backend-go/internal/app/database"
	"backend-go/internal/auth"
	"backend-go/internal/auth/providers"
	"backend-go/internal/config"
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"gorm.io/gorm"
)

const (
	oauthStateCookie = "oauth_state"
	oauthSuccessPath = "/api/auth/callback"
)

type OAuthController struct {
	AppConfig config.AppConfig
	DB        *gorm.DB
	Sessions  *auth.SessionService
	Providers map[string]providers.OAuthProvider
}

func (h *OAuthController) Login(providerName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, ok := h.Providers[providerName]
		if !ok {
			http.Error(w, "Unknown provider", http.StatusNotFound)
			return
		}

		state, err := providers.GenerateState(32)
		if err != nil {
			http.Error(w, "Failed to generate oauth state", http.StatusInternalServerError)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     oauthStateCookie,
			Value:    state,
			Path:     "/",
			MaxAge:   300,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Secure:   h.AppConfig.IsHTTPS,
		})

		http.Redirect(w, r, p.AuthURL(state), http.StatusTemporaryRedirect)
	}
}

func (h *OAuthController) Callback(providerName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, ok := h.Providers[providerName]
		if !ok {
			http.Error(w, "Unknown provider", http.StatusNotFound)
			return
		}

		if errParam := r.URL.Query().Get("error"); errParam != "" {
			desc := r.URL.Query().Get("error_description")
			http.Error(w, fmt.Sprintf("OAuth error (%s): %s", errParam, desc), http.StatusBadRequest)
			return
		}

		cookie, err := r.Cookie(oauthStateCookie)
		stateParam := r.URL.Query().Get("state")
		if err != nil || stateParam == "" || stateParam != cookie.Value {
			http.Error(w, "Invalid OAuth state", http.StatusUnauthorized)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     oauthStateCookie,
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Secure:   h.AppConfig.IsHTTPS,
		})

		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "Missing code parameter", http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		profile, err := p.ExchangeUser(ctx, code)
		if err != nil {
			http.Error(w, "Failed to fetch profile: "+err.Error(), http.StatusBadGateway)
			return
		}

		user, err := h.upsertUserAndAccount(ctx, profile)
		if err != nil {
			http.Error(w, "Failed to save user/account: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if _, err := h.Sessions.CreateSession(ctx, w, user.ID); err != nil {
			http.Error(w, "Failed to create session: "+err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, oauthSuccessPath, http.StatusSeeOther)
	}
}

func (h *OAuthController) upsertUserAndAccount(ctx context.Context, prof *providers.Profile) (*database.User, error) {
	var user database.User

	err := h.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var account database.Account
		err := tx.Where("provider = ? AND provider_account_id = ?", prof.Provider, prof.ProviderAccountID).First(&account).Error
		switch {
		case err == nil:
			if prof.AccessToken != "" {
				account.AccessToken = prof.AccessToken
				if err := tx.Save(&account).Error; err != nil {
					return err
				}
			}
			return tx.First(&user, account.UserID).Error
		case !errors.Is(err, gorm.ErrRecordNotFound):
			return err
		}

		if prof.Email == "" {
			return errors.New("email is not available from provider")
		}

		var existing database.User
		err = tx.Where("email = ?", prof.Email).First(&existing).Error
		switch {
		case err == nil:
			user = existing
			if user.AvatarURL == "" && prof.AvatarURL != "" {
				user.AvatarURL = prof.AvatarURL
				if err := tx.Save(&user).Error; err != nil {
					return err
				}
			}
		case errors.Is(err, gorm.ErrRecordNotFound):
			user = database.User{
				Name:      prof.Name,
				Email:     prof.Email,
				AvatarURL: prof.AvatarURL,
			}
			if err := tx.Create(&user).Error; err != nil {
				return err
			}
		default:
			return err
		}

		account = database.Account{
			UserID:            user.ID,
			Provider:          prof.Provider,
			ProviderAccountID: prof.ProviderAccountID,
			AccessToken:       prof.AccessToken,
			RefreshToken:      prof.RefreshToken,
			ExpiresAt:         prof.ExpiresAt,
		}
		return tx.Create(&account).Error
	})
	if err != nil {
		return nil, err
	}

	return &user, nil
}
