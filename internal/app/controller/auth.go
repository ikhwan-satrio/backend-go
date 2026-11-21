package controller

import (
	"backend-go/internal/app/database"
	"backend-go/internal/auth"
	"backend-go/internal/config"
	"backend-go/internal/dto/login"
	"backend-go/internal/dto/users"
	"backend-go/internal/httpx"
	"backend-go/internal/middleware"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthController struct {
	AppConfig config.AppConfig
}

func (c *AuthController) RegisterPost(w http.ResponseWriter, r *http.Request) {
	var req users.CreateUserRequest
	db, err := database.InitDB()
	ctx := context.Background()

	if err != nil {
		httpx.Error(w, 300, err.Error())
		return
	}

	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.Name == "" || req.Email == "" {
		httpx.Error(w, http.StatusBadRequest, "name and email are required")
		return
	}

	_, err = gorm.G[database.User](db).Where("email = ?", req.Email).First(ctx)
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

	newUsr := &database.User{
		Name:     req.Name,
		Email:    req.Email,
		Password: string(hashedPw),
	}

	if err := gorm.G[database.User](db).Create(ctx, newUsr); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("error: %s", err.Error()))
		return
	}

	httpx.JSON(w, 200, map[string]any{
		"message": "user created successfully!",
	})
}

func (c *AuthController) LoginPost(w http.ResponseWriter, r *http.Request) {
	db, err := database.InitDB()
	ctx := r.Context()

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

	if db == nil {
		log.Println("login error: gorm db is nil")
		httpx.Error(w, http.StatusInternalServerError, "database unavailable")
		return
	}

	var user database.User
	if err := db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
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

	rawToken, tokenHash, err := auth.GenerateSessionToken()
	if err != nil {
		log.Printf("generate session token error: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to create session")
		return
	}

	expiresAt := time.Now().Add(c.AppConfig.SessionDuration)
	sess := &database.Session{
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
	}

	if err := db.WithContext(ctx).Create(sess).Error; err != nil {
		log.Printf("save session error: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "failed to save session")
		return
	}

	auth.SetSessionCookie(w, rawToken, expiresAt, false)

	httpx.JSON(w, http.StatusOK, &login.LoginResponse{
		Message:   "login successful",
		ExpiresAt: expiresAt,
		User: login.User{
			ID:    user.ID,
			Name:  user.Name,
			Email: user.Email,
		},
	})
}

func (c *AuthController) MeGet(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	db, err := database.InitDB()
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "database unavailable")
		return
	}

	var user database.User
	if err := db.WithContext(r.Context()).First(&user, userID).Error; err != nil {
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
