package app

import (
	"backend-go/internal/config"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type AppBootstrap struct {
	router chi.Router
}

func NewAppBootstrap() *AppBootstrap {
	return &AppBootstrap{
		router: chi.NewRouter(),
	}
}

func (a *AppBootstrap) Run() {
	cfg := config.UseApp()

	c := chi.NewRouter()
	c.Use(middleware.Logger)
	c.Use(middleware.Recoverer)

	r := a.CreateRouter()
	c.Mount("/api", r)

	c.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("good"))
	})

	addr := cfg.Port
	if addr != "" && !strings.Contains(addr, ":") {
		addr = ":" + addr
	}

	s := &http.Server{
		Addr:           addr,
		Handler:        c,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	log.Printf("Starting server on %s", addr)
	log.Fatal(s.ListenAndServe())
}
