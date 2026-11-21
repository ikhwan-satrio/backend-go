package app

import (
	"backend-go/internal/app/controller"
	"backend-go/internal/config"
	"backend-go/internal/middleware"

	"github.com/go-chi/chi/v5"
)

func (a *AppBootstrap) CreateRouter() chi.Router {
	r := a.router
	cfg := config.UseApp()
	authController := controller.AuthController{
		AppConfig: cfg,
	}

	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", authController.RegisterPost)
		r.Post("/login", authController.LoginPost)

		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireSession())
			r.Get("/me", authController.MeGet)
		})
	})

	return r
}
