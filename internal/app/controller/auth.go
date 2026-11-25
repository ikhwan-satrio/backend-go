package controller

import (
	"backend-go/internal/app/database"
	"backend-go/internal/auth"
	"backend-go/internal/config"
	"backend-go/internal/dto/login"
	"backend-go/internal/dto/users"
	"backend-go/internal/httpx"
	"backend-go/internal/middleware"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthController struct {
	AppConfig config.AppConfig
	DB        *gorm.DB
	Sessions  *auth.SessionService
}

func (c *AuthController) RegisterPost(w http.ResponseWriter, r *http.Request) {
	var req users.CreateUserRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.Name == "" || req.Email == "" || req.Password == "" {
		httpx.Error(w, http.StatusBadRequest, "name, email and password are required")
		return
	}

	ctx := r.Context()

	_, err := gorm.G[database.User](c.DB).Where("email = ?", req.Email).First(ctx)
	if err == nil {
		httpx.Error(w, http.StatusConflict, fmt.Sprintf("email %s is already registered", req.Email))
		return
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Printf("query user by email error: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to check email")
		return
	}

	hashedPw, err := bcrypt.GenerateFromPassword([]byte(req.Password), c.AppConfig.PasswordCost)
	if err != nil {
		log.Printf("hash password error: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	newUsr := &database.User{
		Name:     req.Name,
		Email:    req.Email,
		Password: string(hashedPw),
	}

	if err := gorm.G[database.User](c.DB).Create(ctx, newUsr); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("error: %s", err.Error()))
		return
	}

	httpx.JSON(w, http.StatusCreated, map[string]any{
		"message": "user created successfully!",
	})
}

func (c *AuthController) LoginPost(w http.ResponseWriter, r *http.Request) {
	var req login.LoginRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	email := strings.TrimSpace(strings.ToLower(req.Email))
	if email == "" || req.Password == "" {
		httpx.Error(w, http.StatusBadRequest, "email and password are required")
		return
	}

	ctx := r.Context()

	var user database.User
	if err := c.DB.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			httpx.Error(w, http.StatusUnauthorized, "invalid email or password")
			return
		}
		log.Printf("login query user error: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to process login")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		httpx.Error(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	sess, err := c.Sessions.CreateSession(ctx, w, user.ID)
	if err != nil {
		log.Printf("create session error: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to create session")
		return
	}

	httpx.JSON(w, http.StatusOK, &login.LoginResponse{
		Message:   "login successful",
		ExpiresAt: sess.ExpiresAt,
		User: login.User{
			ID:    user.ID,
			Name:  user.Name,
			Email: user.Email,
		},
	})
}

func (c *AuthController) LogoutPost(w http.ResponseWriter, r *http.Request) {
	if err := c.logout(w, r); err != nil {
		log.Printf("logout error: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to logout")
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]string{"message": "logout successful"})
}

func (c *AuthController) LogoutGet(w http.ResponseWriter, r *http.Request) {
	if err := c.logout(w, r); err != nil {
		log.Printf("logout error: %v", err)
		http.Error(w, "failed to logout", http.StatusInternalServerError)
		return
	}

	if c.AppConfig.FrontendURL != "" {
		http.Redirect(w, r, c.AppConfig.FrontendURL, http.StatusSeeOther)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `<!doctype html>
<html lang="en">
<head><meta charset="utf-8"><title>Logged out</title></head>
<body style="font-family:sans-serif;text-align:center;margin-top:4rem">
<h1>Logged out successfully</h1>
<p>Your session has been cleared. You can close this tab.</p>
</body>
</html>`)
}

func (c *AuthController) logout(w http.ResponseWriter, r *http.Request) error {
	var rawToken string
	if cookie, err := r.Cookie(c.AppConfig.SessionCookieName); err == nil {
		rawToken = cookie.Value
	}
	return c.Sessions.DeleteSession(r.Context(), w, rawToken)
}

func (c *AuthController) MeGet(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var user database.User
	if err := c.DB.WithContext(r.Context()).First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			httpx.Error(w, http.StatusNotFound, "user not found")
			return
		}
		log.Printf("get user error: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to get user")
		return
	}

	httpx.JSON(w, http.StatusOK, &login.User{
		ID:    user.ID,
		Name:  user.Name,
		Email: user.Email,
	})
}
