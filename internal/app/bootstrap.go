package app

import (
	"backend-go/internal/app/database"
	"backend-go/internal/config"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"gorm.io/gorm"
)

type AppBootstrap struct {
	router chi.Router
	cfg    config.AppConfig
	db     *gorm.DB
}

func NewAppBootstrap() *AppBootstrap {
	cfg := config.UseApp()

	db, err := database.InitDB(cfg)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	return &AppBootstrap{
		router: chi.NewRouter(),
		cfg:    cfg,
		db:     db,
	}
}

func (a *AppBootstrap) Run() {
	c := chi.NewRouter()
	c.Use(middleware.Logger)
	c.Use(middleware.Recoverer)

	c.Mount("/api", a.CreateRouter())

	c.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("good"))
	})

	addr := a.cfg.Port
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
