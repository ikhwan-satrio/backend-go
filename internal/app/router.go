package app

import (
	"backend-go/internal/app/controller"
	"backend-go/internal/auth"
	"backend-go/internal/auth/providers"
	"backend-go/internal/middleware"

	"github.com/go-chi/chi/v5"
)

func (a *AppBootstrap) CreateRouter() chi.Router {
	r := a.router
	cfg := a.cfg

	sessions := &auth.SessionService{
		DB:         a.db,
		CookieName: cfg.SessionCookieName,
		SessionTTL: cfg.SessionDuration,
		IsHTTPS:    cfg.IsHTTPS,
	}

	githubProvider := providers.NewGitHubProvider(providers.GitHubConfig{
		ClientID:     cfg.GitHubClientID,
		ClientSecret: cfg.GitHubClientSecret,
		RedirectURL:  cfg.GitHubRedirectURL,
	})

	authController := &controller.AuthController{
		AppConfig: cfg,
		DB:        a.db,
		Sessions:  sessions,
	}
	oauthController := &controller.OAuthController{
		AppConfig: cfg,
		DB:        a.db,
		Sessions:  sessions,
		Providers: map[string]providers.OAuthProvider{
			githubProvider.ProviderName(): githubProvider,
		},
	}

	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", authController.RegisterPost)
		r.Post("/login", authController.LoginPost)
		r.Post("/logout", authController.LogoutPost)
		r.Get("/logout", authController.LogoutGet)

		r.Get("/oauth/github", oauthController.Login("github"))
		r.Get("/oauth/github/callback", oauthController.Callback("github"))

		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireSession(a.db, cfg))
			r.Get("/me", authController.MeGet)
			r.Get("/callback", authController.MeGet)
		})
	})

	return r
}
