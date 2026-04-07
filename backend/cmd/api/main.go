package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/foodbi/backend/internal/auth"
	"github.com/foodbi/backend/internal/dashboard"
	"github.com/foodbi/backend/internal/database"
	"github.com/foodbi/backend/internal/locations"
	"github.com/foodbi/backend/internal/middleware"
	"github.com/foodbi/backend/internal/purchases"
	"github.com/foodbi/backend/internal/revenue"
	"github.com/foodbi/backend/internal/statistics"
	"github.com/foodbi/backend/internal/ai"
	"github.com/foodbi/backend/internal/employees"
	"github.com/foodbi/backend/internal/files"
	"github.com/foodbi/backend/internal/notifications"
	"github.com/foodbi/backend/internal/profiles"
	"github.com/foodbi/backend/internal/stock"
	"github.com/foodbi/backend/internal/supplying"
	"github.com/foodbi/backend/internal/transfers"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	_ = godotenv.Load()

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	if os.Getenv("ENV") != "production" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	db, err := database.NewPool(context.Background())
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer db.Close()

	authService := auth.NewService(db)
	authHandler := auth.NewHandler(authService)
	locHandler := locations.NewHandler(db)
	dashHandler := dashboard.NewHandler(db)
	revHandler := revenue.NewHandler(db)
	purchHandler := purchases.NewHandler(db)
	statsHandler := statistics.NewHandler(db)
	stockHandler := stock.NewHandler(db)
	supplyHandler := supplying.NewHandler(db)
	transferHandler := transfers.NewHandler(db)
	empHandler := employees.NewHandler(db)
	profHandler := profiles.NewHandler(db)
	notifHandler := notifications.NewHandler(db)
	aiHandler := ai.NewHandler(db)
	fileHandler := files.NewHandler(db)

	r := chi.NewRouter()

	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(middleware.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(30 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok","time":"%s"}`, time.Now().UTC().Format(time.RFC3339))
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", authHandler.Register)
			r.Post("/login", authHandler.Login)
			r.Post("/verify-otp", authHandler.VerifyOTP)
			r.Post("/refresh", authHandler.RefreshToken)
		})

		r.Group(func(r chi.Router) {
			r.Use(middleware.JWTAuth)
			r.Use(middleware.TenantContext)

			r.Post("/auth/logout", authHandler.Logout)
			r.Get("/auth/me", authHandler.Me)

			r.Mount("/locations", locHandler.Routes())
			r.Mount("/dashboard", dashHandler.Routes())
			r.Mount("/revenue", revHandler.Routes())
			r.Mount("/purchases", purchHandler.Routes())
			r.Mount("/statistics", statsHandler.Routes())
			r.Mount("/stock", stockHandler.Routes())
			r.Mount("/supplying", supplyHandler.Routes())
			r.Mount("/transfers", transferHandler.Routes())
			r.Mount("/employees", empHandler.Routes())
			r.Mount("/profile", profHandler.Routes())
			r.Mount("/notifications", notifHandler.Routes())
			r.Mount("/ai", aiHandler.Routes())
			r.Mount("/files", fileHandler.Routes())
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info().Str("port", port).Msg("starting API server")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server failed")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("shutting down server")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("server forced shutdown")
	}
	log.Info().Msg("server stopped")
}
