package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/diarra/backend/internal/auth"
	"github.com/diarra/backend/internal/email"
	"github.com/diarra/backend/internal/handler"
	"github.com/diarra/backend/internal/middleware"
	"github.com/diarra/backend/internal/payment"
	"github.com/diarra/backend/internal/realtime"
	"github.com/diarra/backend/internal/repository"
	"github.com/diarra/backend/internal/service"
	"github.com/diarra/backend/internal/storage"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx := context.Background()

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET is required")
	}
	refreshSecret := os.Getenv("REFRESH_SECRET")
	if refreshSecret == "" {
		log.Fatal("REFRESH_SECRET is required")
	}

	jwtManager := auth.NewJWTManager(jwtSecret, refreshSecret)

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("Unable to ping database: %v", err)
	}

	// Storage R2 (optionnel en dev local)
	var storageService handler.StorageService
	r2, err := storage.NewR2Storage(storage.R2Config{
		AccountID:       os.Getenv("R2_ACCOUNT_ID"),
		AccessKeyID:     os.Getenv("R2_ACCESS_KEY_ID"),
		SecretAccessKey: os.Getenv("R2_SECRET_ACCESS_KEY"),
		Bucket:          os.Getenv("R2_BUCKET"),
		Endpoint:        os.Getenv("R2_ENDPOINT"),
	})
	if err != nil || os.Getenv("R2_ACCOUNT_ID") == "" {
		log.Println("WARNING: R2 storage non configuré, upload désactivé")
		storageService = nil
	} else {
		storageService = r2
	}

	// Repositories
	userRepo := repository.NewUserRepo(pool)
	productRepo := repository.NewProductRepo(pool)
	saleRepo := repository.NewSaleRepo(pool)
	deliveryRepo := repository.NewDeliveryRepo(pool)
	payoutRepo := repository.NewPayoutRepo(pool)

	// Notifications email (optionnel en dev local)
	var notifications *email.NotificationService
	if os.Getenv("RESEND_API_KEY") != "" {
		resend := email.NewResendClient(email.ResendConfig{
			APIKey: os.Getenv("RESEND_API_KEY"),
			From:   os.Getenv("RESEND_FROM"),
		})
		notifications = email.NewNotificationService(resend, os.Getenv("FRONTEND_URL"))
	} else {
		log.Println("WARNING: Resend non configuré, emails désactivés")
	}

	// Services
	authService := service.NewAuthService(userRepo, jwtManager)

	// Paiement PayDunya (optionnel en dev local)
	var paydunya *payment.PayDunyaClient
	if os.Getenv("PAYDUNYA_MASTER_KEY") != "" {
		paydunya = payment.NewPayDunyaClient(payment.PayDunyaConfig{
			MasterKey:  os.Getenv("PAYDUNYA_MASTER_KEY"),
			PrivateKey: os.Getenv("PAYDUNYA_PRIVATE_KEY"),
			Token:      os.Getenv("PAYDUNYA_TOKEN"),
			ReturnURL:  os.Getenv("PAYDUNYA_RETURN_URL"),
			CancelURL:  os.Getenv("PAYDUNYA_CANCEL_URL"),
			WebhookURL: os.Getenv("PAYDUNYA_WEBHOOK_URL"),
		})
	} else {
		log.Println("WARNING: PayDunya non configuré, paiement désactivé")
	}

	// Handlers
	healthHandler := handler.NewHealthHandler(pool)
	authHandler := handler.NewAuthHandler(authService)
	productHandler := handler.NewProductHandler(productRepo, storageService)
	saleHandler := handler.NewSaleHandler(saleRepo, productRepo, paydunya)
	webhookHandler := handler.NewWebhookHandler(saleRepo, userRepo, productRepo, notifications, os.Getenv("PAYDUNYA_WEBHOOK_SECRET"))

	// Temps réel (LISTEN/NOTIFY + WebSocket)
	hub := realtime.NewHub(pool)
	realtimeHandler := handler.NewRealtimeHandler(hub)

	// Livraison (liens signés R2)
	var deliveryHandler *handler.DeliveryHandler
	if r2 != nil {
		deliveryHandler = handler.NewDeliveryHandler(deliveryRepo, saleRepo, productRepo, r2)
	} else {
		deliveryHandler = nil
		log.Println("WARNING: livraison désactivée (R2 non configuré)")
	}

	// Versements & revenus vendeur
	payoutHandler := handler.NewPayoutHandler(payoutRepo, saleRepo, productRepo)

	// Administration
	adminHandler := handler.NewAdminHandler(productRepo, saleRepo, userRepo)

	// Support tickets
	ticketRepo := repository.NewTicketRepo(pool)
	ticketHandler := handler.NewTicketHandler(ticketRepo)

	r := chi.NewRouter()

	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(middleware.NewRateLimiter(10, 40).Middleware)
	r.Use(middleware.SecurityHeaders)

	cors := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	})
	r.Use(cors.Handler)

	r.Get("/health", healthHandler.ServeHTTP)

	// Auth routes
	r.Route("/api/auth", func(r chi.Router) {
		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)
		r.Post("/logout", authHandler.Logout)
		r.Post("/refresh", authHandler.Refresh)
		r.Post("/verify-email", authHandler.VerifyEmail)
		r.Post("/forgot-password", authHandler.ForgotPassword)
		r.Post("/reset-password", authHandler.ResetPassword)
	})

	// Public product routes
	r.Route("/api/products", func(r chi.Router) {
		r.Get("/", productHandler.List)
		r.Get("/{id}", productHandler.Get)
	})

	// Vendor product routes (authentifié)
	r.Route("/api/vendor/products", func(r chi.Router) {
		r.Use(middleware.RequireAuth(jwtManager))
		r.Get("/", productHandler.ListVendor)
		r.Post("/", productHandler.Create)
		r.Post("/upload", productHandler.Upload)
		r.Put("/{id}", productHandler.Update)
		r.Delete("/{id}", productHandler.Delete)
	})

	// Orders (authentifié)
	r.Route("/api/orders", func(r chi.Router) {
		r.Use(middleware.RequireAuth(jwtManager))
		r.Post("/", saleHandler.Create)
		r.Get("/", saleHandler.List)
		r.Get("/{id}", saleHandler.Get)
	})

	// Webhooks (pas de JWT)
	r.Route("/api/webhooks", func(r chi.Router) {
		r.Post("/paydunya", webhookHandler.PayDunyaWebhook)
	})

	// WebSocket temps réel
	r.Route("/ws", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAuth(jwtManager))
			r.Get("/order/{id}", realtimeHandler.OrderWS)
		})
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAuth(jwtManager))
			r.Use(middleware.RequireAdmin)
			r.Get("/admin", realtimeHandler.ModerationWS)
		})
	})

	// Livraison
	if deliveryHandler != nil {
		r.Route("/api/orders/{id}/delivery", func(r chi.Router) {
			r.Use(middleware.RequireAuth(jwtManager))
			r.Post("/", deliveryHandler.Generate)
		})
		r.Get("/api/delivery/{token}", deliveryHandler.Download)
	}

	// Versements vendeur
	r.Route("/api/vendor", func(r chi.Router) {
		r.Use(middleware.RequireAuth(jwtManager))
		r.Get("/earnings", payoutHandler.Earnings)
		r.Post("/payouts", payoutHandler.Create)
		r.Get("/payouts", payoutHandler.Earnings)
	})

	// Routes admin (authentifié + admin)
	r.Route("/api/admin", func(r chi.Router) {
		r.Use(middleware.RequireAuth(jwtManager))
		r.Use(middleware.RequireAdmin)
		r.Get("/products/pending", adminHandler.PendingProducts)
		r.Put("/products/{id}/moderate", adminHandler.Moderate)
		r.Get("/users", adminHandler.Users)
		r.Put("/users/{id}/suspend", adminHandler.SuspendUser)
		r.Get("/stats", adminHandler.Stats)
		r.Get("/sales", adminHandler.Sales)
	})

	// Support tickets (utilisateur connecté, admin pour tous)
	r.Route("/api/tickets", func(r chi.Router) {
		r.Use(middleware.RequireAuth(jwtManager))
		r.Post("/", ticketHandler.Create)
		r.Get("/", ticketHandler.List)
		r.Get("/{id}/messages", ticketHandler.GetMessages)
		r.Post("/{id}/messages", ticketHandler.AddMessage)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Server starting on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
