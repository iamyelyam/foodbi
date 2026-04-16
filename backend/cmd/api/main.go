package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/foodbi/backend/internal/ai"
	"github.com/foodbi/backend/internal/auth"
	"github.com/foodbi/backend/internal/cache"
	"github.com/foodbi/backend/internal/dashboard"
	"github.com/foodbi/backend/internal/database"
	"github.com/foodbi/backend/internal/employees"
	"github.com/foodbi/backend/internal/files"
	"github.com/foodbi/backend/internal/locations"
	"github.com/foodbi/backend/internal/middleware"
	"github.com/foodbi/backend/internal/numiersync"
	gosync "github.com/foodbi/backend/internal/sync"
	"github.com/foodbi/backend/internal/notifications"
	"github.com/foodbi/backend/internal/payments"
	"github.com/foodbi/backend/internal/profiles"
	"github.com/foodbi/backend/internal/purchases"
	"github.com/foodbi/backend/internal/revenue"
	"github.com/foodbi/backend/internal/statistics"
	"github.com/foodbi/backend/internal/stock"
	"github.com/foodbi/backend/internal/supplying"
	"github.com/foodbi/backend/internal/telegram"
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

	// Auto-apply pending migrations on startup
	migrationsDir := os.Getenv("MIGRATIONS_DIR")
	if migrationsDir == "" {
		migrationsDir = "./migrations"
	}
	if err := database.RunMigrations(context.Background(), db, migrationsDir); err != nil {
		log.Warn().Err(err).Msg("migration: runner failed — continuing to serve")
	}

	// Telegram bot (start if token configured)
	botCtx, botCancel := context.WithCancel(context.Background())
	var tgBot *telegram.Bot
	if token := os.Getenv("TELEGRAM_BOT_TOKEN"); token != "" {
		tgBot = telegram.NewBot(token, db)
		go tgBot.Start(botCtx)
	} else {
		log.Warn().Msg("TELEGRAM_BOT_TOKEN not set — telegram bot disabled")
	}

	dashCache := cache.New()
	defer dashCache.Close()

	authService := auth.NewService(db)
	authHandler := auth.NewHandler(authService)
	syncService := gosync.NewService(db)
	numierSyncSvc := numiersync.NewService(db)
	locHandler := locations.NewHandler(db, syncService, numierSyncSvc)
	dashHandler := dashboard.NewHandler(db, dashCache)
	revHandler := revenue.NewHandler(db)
	purchHandler := purchases.NewHandler(db)
	statsHandler := statistics.NewHandler(db)
	stockHandler := stock.NewHandler(db)
	supplyHandler := supplying.NewHandler(db)
	transferHandler := transfers.NewHandler(db)
	empHandler := employees.NewHandler(db)
	profHandler := profiles.NewHandler(db)
	notifHandler := notifications.NewHandler(db)
	aiHandler := ai.NewHandler(db, os.Getenv("OPENAI_API_KEY"))
	fileHandler := files.NewHandler(db)
	paymentHandler := payments.NewHandler(db, tgBot)

	r := chi.NewRouter()

	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(middleware.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(30 * time.Second))
	r.Use(middleware.SecurityHeaders)
	r.Use(middleware.MaxBodySize(10 << 20)) // 10MB max request body
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

	// Webhook endpoints (no JWT auth — use HMAC signature per company)
	r.Post("/api/v1/webhooks/payment/{companyID}", paymentHandler.HandleWebhook)

	// Rate limiters to prevent brute-force and abuse.
	// Per-IP sliding windows; tuned conservatively for mobile app usage.
	loginLimiter := middleware.NewRateLimiter(10, time.Minute)            // 10 logins/min per IP
	registerLimiter := middleware.NewRateLimiter(5, time.Hour)            // 5 registrations/hour per IP
	forgotLimiter := middleware.NewRateLimiter(5, 10*time.Minute)         // 5 password-reset triggers/10min per IP
	defer loginLimiter.Close()
	defer registerLimiter.Close()
	defer forgotLimiter.Close()

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.With(registerLimiter.Middleware).Post("/register", authHandler.Register)
			r.With(loginLimiter.Middleware).Post("/login", authHandler.Login)
			r.With(loginLimiter.Middleware).Post("/verify-otp", authHandler.VerifyOTP)
			r.Post("/refresh", authHandler.RefreshToken)
			r.With(registerLimiter.Middleware).Post("/accept-invite", authHandler.AcceptInvite)
			r.With(forgotLimiter.Middleware).Post("/forgot-password", authHandler.ForgotPassword)
			r.With(forgotLimiter.Middleware).Post("/reset-password", authHandler.ResetPassword)
		})

		r.Group(func(r chi.Router) {
			r.Use(middleware.JWTAuth)
			r.Use(middleware.TenantContext)

			r.Post("/auth/logout", authHandler.Logout)
			r.Get("/auth/me", authHandler.Me)
			r.Post("/auth/invite", authHandler.Invite)

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
	botCancel() // stop telegram bot
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("server forced shutdown")
	}
	paymentHandler.Stop() // drain telegram notify worker pool
	log.Info().Msg("server stopped")
}
