package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/diarra/backend/internal/auth"
	"github.com/diarra/backend/internal/cache"
	"github.com/diarra/backend/internal/email"
	"github.com/diarra/backend/internal/handler"
	"github.com/diarra/backend/internal/middleware"
	"github.com/diarra/backend/internal/model"
	"github.com/diarra/backend/internal/otp"
	"github.com/diarra/backend/internal/payment"
	"github.com/diarra/backend/internal/realtime"
	"github.com/diarra/backend/internal/repository"
	"github.com/diarra/backend/internal/service"
	"github.com/diarra/backend/internal/sms"
	"github.com/diarra/backend/internal/storage"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	chiCors "github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx := context.Background()
	startTime := time.Now()

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

	// Storage objet (S3-compatible: Tigris, MinIO, R2... — optionnel en dev local)
	var storageService handler.StorageService
	s3, err := storage.NewS3Storage(storage.S3Config{
		Endpoint:        os.Getenv("S3_ENDPOINT"),
		AccessKeyID:     os.Getenv("S3_ACCESS_KEY_ID"),
		SecretAccessKey: os.Getenv("S3_SECRET_ACCESS_KEY"),
		Bucket:          os.Getenv("S3_BUCKET"),
		Region:          os.Getenv("S3_REGION"),
	})
	if err != nil || os.Getenv("S3_ENDPOINT") == "" {
		log.Println("WARNING: stockage objet non configuré, upload désactivé")
		storageService = nil
		s3 = nil
	} else {
		storageService = s3
	}

	// Pour le health check admin (interface minimale, nil explicite si
	// stockage désactivé — évite le piège de l'interface non-nil sur pointeur nil).
	var storageHealthPinger interface{ Ping(context.Context) error }
	if s3 != nil {
		storageHealthPinger = s3
	}

	// Cache Redis (catalogue, stats admin, solde vendeur, rate limiting) —
	// optionnel comme le stockage S3 ci-dessus : sans REDIS_URL, redisCache
	// reste un client no-op (voir internal/cache), tout continue de
	// fonctionner directement sur la base de données, juste sans mise en cache.
	redisCache, err := cache.New(os.Getenv("REDIS_URL"))
	if err != nil {
		log.Fatalf("REDIS_URL invalide: %v", err)
	}
	if os.Getenv("REDIS_URL") == "" {
		log.Println("WARNING: cache Redis non configuré, fonctionne sans mise en cache")
	}

	// Repositories
	userRepo := repository.NewUserRepo(pool)
	otpRepo := repository.NewOTPRepo(pool)
	productRepo := repository.NewProductRepo(pool)
	saleRepo := repository.NewSaleRepo(pool)
	deliveryRepo := repository.NewDeliveryRepo(pool)
	payoutRepo := repository.NewPayoutRepo(pool)
	referralRepo := repository.NewReferralRepo(pool)
	bundleRepo := repository.NewBundleRepo(pool)
	adminPermRepo := repository.NewAdminPermissionRepo(pool)
	settingsRepo := repository.NewSettingsRepo(pool)

	// OTP service
	otpService := otp.NewService(otpRepo)

	// SMS sender (stub en dev, renvoie le code dans la réponse API)
	var smsSender sms.Sender = sms.LogSender{}

	// Notifications email (optionnel en dev local)
	var notifications *email.NotificationService
	switch {
	case os.Getenv("SMTP_HOST") != "":
		port := 587
		if p := os.Getenv("SMTP_PORT"); p != "" {
			if parsed, err := strconv.Atoi(p); err == nil {
				port = parsed
			}
		}
		smtp, err := email.NewSMTPClient(email.SMTPConfig{
			Host:     os.Getenv("SMTP_HOST"),
			Port:     port,
			Username: os.Getenv("SMTP_USER"),
			Password: os.Getenv("SMTP_PASS"),
			From:     os.Getenv("SMTP_FROM"),
		})
		if err != nil {
			log.Fatalf("SMTP config invalide: %v", err)
		}
		notifications = email.NewNotificationService(smtp, os.Getenv("FRONTEND_URL"))
		log.Printf("Emails via SMTP (%s:%d)", os.Getenv("SMTP_HOST"), port)
	case os.Getenv("RESEND_API_KEY") != "":
		resend := email.NewResendClient(email.ResendConfig{
			APIKey: os.Getenv("RESEND_API_KEY"),
			From:   os.Getenv("RESEND_FROM"),
		})
		notifications = email.NewNotificationService(resend, os.Getenv("FRONTEND_URL"))
	case os.Getenv("MAILTRAP_API_KEY") != "":
		fromEmail := os.Getenv("MAILTRAP_FROM")
		if fromEmail == "" {
			fromEmail = "hello@demomailtrap.co"
		}
		mailtrap := email.NewMailtrapClient(email.MailtrapConfig{
			APIKey:    os.Getenv("MAILTRAP_API_KEY"),
			FromEmail: fromEmail,
			FromName:  "DIARRA",
			SandboxID: os.Getenv("MAILTRAP_SANDBOX_ID"),
		})
		notifications = email.NewNotificationService(mailtrap, os.Getenv("FRONTEND_URL"))
		mode := "send.api"
		if os.Getenv("MAILTRAP_SANDBOX_ID") != "" {
			mode = "sandbox (bac de test)"
		}
		log.Printf("Emails via Mailtrap (%s)", mode)
	default:
		log.Println("WARNING: aucun fournisseur email (SMTP_HOST, RESEND_API_KEY ou MAILTRAP_API_KEY), emails désactivés")
	}

	// Services
	authService := service.NewAuthService(userRepo, otpRepo, otpService, smsSender, jwtManager, notifications, adminPermRepo)

	// Connexion Google via Firebase Auth (optionnelle — nécessite un projet
	// Firebase avec le fournisseur Google activé, voir FIREBASE_PROJECT_ID).
	var firebaseVerifier *auth.FirebaseVerifier
	if projectID := os.Getenv("FIREBASE_PROJECT_ID"); projectID != "" {
		firebaseVerifier = auth.NewFirebaseVerifier(projectID)
	} else {
		log.Println("WARNING: FIREBASE_PROJECT_ID non configuré, connexion Google désactivée")
	}

	// Paiement PawaPay (mobile money, optionnel en dev local)
	var pawapay *payment.PawaPayClient
	if os.Getenv("PAWAPAY_API_KEY") != "" {
		pawapay = payment.NewPawaPayClient(payment.PawaPayConfig{
			APIKey:      os.Getenv("PAWAPAY_API_KEY"),
			BaseURL:     os.Getenv("PAWAPAY_BASE_URL"), // défaut: sandbox
			CallbackURL: os.Getenv("PAWAPAY_CALLBACK_URL"),
		})
	} else {
		log.Println("WARNING: PawaPay non configuré, paiement désactivé")
	}

	// IP autorisées pour les callbacks PawaPay (sécurité en plus du Content-Digest)
	allowedIPs := []string{}
	if ips := os.Getenv("PAWAPAY_CALLBACK_IPS"); ips != "" {
		allowedIPs = strings.Split(ips, ",")
	}

	// Notifications in-app
	notificationRepo := repository.NewNotificationRepo(pool)
	notificationHandler := handler.NewNotificationHandler(notificationRepo)

	// Programme de reversement automatique ("Fidélisation") — voir
	// service.DonationService pour la logique (cagnotte alimentée par une
	// part de la commission sur chaque vente, distribution PawaPay au-delà
	// d'un seuil configurable).
	donationRepo := repository.NewDonationRepo(pool)
	donationService := service.NewDonationService(donationRepo, settingsRepo, notificationRepo, userRepo, pawapay)

	// Handlers
	healthHandler := handler.NewHealthHandler(pool)
	authHandler := handler.NewAuthHandler(authService, firebaseVerifier)
	productHandler := handler.NewProductHandler(productRepo, userRepo, saleRepo, storageService, os.Getenv("FRONTEND_URL"), redisCache)
	saleHandler := handler.NewSaleHandler(saleRepo, productRepo, referralRepo, userRepo, settingsRepo, pawapay, notifications, os.Getenv("FRONTEND_URL"))
	closerHandler := handler.NewCloserHandler(referralRepo, productRepo, os.Getenv("FRONTEND_URL"))
	bundleHandler := handler.NewBundleHandler(bundleRepo, productRepo)
	webhookHandler := handler.NewWebhookHandler(saleRepo, userRepo, productRepo, payoutRepo, pawapay, donationService, notifications, notificationRepo, s3, allowedIPs, redisCache)
	feedHandler := handler.NewFeedHandler(productRepo, os.Getenv("FRONTEND_URL"))
	donationHandler := handler.NewDonationHandler(donationRepo, settingsRepo, donationService)

	// Temps réel (LISTEN/NOTIFY + WebSocket)
	hub := realtime.NewHub(pool)
	realtimeHandler := handler.NewRealtimeHandler(hub)

	// Livraison (liens signés du stockage objet)
	var deliveryHandler *handler.DeliveryHandler
	if s3 != nil {
		deliveryHandler = handler.NewDeliveryHandler(deliveryRepo, saleRepo, productRepo, s3)
	} else {
		deliveryHandler = nil
		log.Println("WARNING: livraison désactivée (stockage objet non configuré)")
	}

	// Versements & revenus vendeur
	payoutHandler := handler.NewPayoutHandler(payoutRepo, saleRepo, productRepo, userRepo, settingsRepo, pawapay, redisCache)

	// Support tickets
	ticketRepo := repository.NewTicketRepo(pool)
	ticketHandler := handler.NewTicketHandler(ticketRepo, adminPermRepo)

	// Widget de contact support public (visiteur non authentifié) + registre
	// des agents notifiés par email à chaque nouveau message.
	supportContactRepo := repository.NewSupportContactRepo(pool)
	supportContactHandler := handler.NewSupportContactHandler(supportContactRepo, notifications)

	// Administration
	adminHandler := handler.NewAdminHandler(productRepo, saleRepo, userRepo, referralRepo, adminPermRepo, payoutRepo, settingsRepo, ticketRepo, pool, storageHealthPinger, startTime, pawapay, notifications, redisCache)

	r := chi.NewRouter()

	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(middleware.NewRateLimiter(redisCache, 10, 40).Middleware)
	r.Use(middleware.SecurityHeaders)

	// CORS restrictif (ne plus utiliser "*" avec credentials)
	corsOrigins := []string{"http://localhost:3000"}
	if origins := os.Getenv("CORS_ALLOWED_ORIGINS"); origins != "" {
		corsOrigins = strings.Split(origins, ",")
	}
	corsHandler := chiCors.New(chiCors.Options{
		AllowedOrigins:   corsOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	})
	r.Use(corsHandler.Handler)

	r.Get("/health", healthHandler.ServeHTTP)

	// Auth routes. authRateLimiter est volontairement bien plus strict que la
	// limite globale (10 req/s) : le brute-force sur mot de passe/OTP se
	// mesure en tentatives par minute, pas par seconde — 0.2 req/s (1 toutes
	// les 5s en régime soutenu) avec une rafale de 8 laisse une marge pour un
	// utilisateur qui se trompe plusieurs fois de suite, sans laisser un
	// script tenter des centaines de mots de passe par minute.
	authRateLimiter := middleware.NewRateLimiter(redisCache, 0.2, 8)
	r.Route("/api/auth", func(r chi.Router) {
		r.Use(authRateLimiter.Middleware)
		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)
		r.Post("/google", authHandler.GoogleLogin)
		r.Post("/logout", authHandler.Logout)
		r.Post("/refresh", authHandler.Refresh)
		r.Post("/verify-email", authHandler.VerifyEmail) // Ancien flux (lien) — conservé
		r.Post("/forgot-password", authHandler.ForgotPassword)
		r.Post("/reset-password", authHandler.ResetPassword)
		r.With(middleware.RequireAuth(jwtManager)).Post("/send-otp", authHandler.SendOTP)
		r.With(middleware.RequireAuth(jwtManager)).Post("/verify-otp", authHandler.VerifyOTP)
		r.With(middleware.RequireAuth(jwtManager)).Post("/verify-phone-firebase", authHandler.VerifyPhoneFirebase)
		r.With(middleware.RequireAuth(jwtManager)).Get("/me", authHandler.Me)
	})

	// Compte (libre-service, tout utilisateur connecté — client, vendeur ou closer)
	r.Route("/api/account", func(r chi.Router) {
		r.Use(middleware.RequireAuth(jwtManager))
		r.Post("/roles", authHandler.AddRole)
		r.Put("/profile", authHandler.UpdateProfile)
		r.Put("/ad-tracking", authHandler.UpdateAdTracking)
		// Moyen de versement/retrait — ouvert à tout utilisateur connecté (pas
		// seulement les vendeurs), ex: un client enregistre un numéro pour
		// recevoir un remboursement. Pas de vérification téléphone requise ici
		// (juste enregistrer/modifier le numéro) — seul le déclenchement d'un
		// vrai versement (POST /payouts) l'exige encore.
		r.Get("/payout-method", payoutHandler.GetPayoutMethod)
		r.Put("/payout-method", payoutHandler.SetPayoutMethod)
	})

	// Packs de produits — lecture publique, gestion vendeur.
	r.Get("/api/bundles/{id}", bundleHandler.Get)
	r.Route("/api/vendor/bundles", func(r chi.Router) {
		r.Use(middleware.RequireAuth(jwtManager))
		r.Use(middleware.RequireRole(model.RoleVendeur))
		r.Get("/", bundleHandler.ListVendor)
		r.Post("/", bundleHandler.Create)
		r.Delete("/{id}", bundleHandler.Delete)
	})

	// Public product routes
	r.Route("/api/products", func(r chi.Router) {
		r.Use(middleware.OptionalAuth(jwtManager))
		r.Get("/", productHandler.List)
		r.Get("/{id}", productHandler.Get)
		r.Get("/{id}/cover", productHandler.Cover)
		r.Get("/{id}/preview/{index}", productHandler.Preview)
	})

	// Boutique publique d'un vendeur (partageable via QR code)
	r.Get("/api/vendors/{id}/shop", productHandler.Shop)

	// Vendor product routes (vendeur authentifié + email vérifié)
	r.Route("/api/vendor/products", func(r chi.Router) {
		r.Use(middleware.RequireAuth(jwtManager))
		r.Use(middleware.RequireRole(model.RoleVendeur))
		r.Use(middleware.RequireVerifiedEmail(userRepo))
		r.Get("/", productHandler.ListVendor)
		r.Post("/", productHandler.Create)
		r.Post("/upload", productHandler.Upload)
		r.Put("/{id}", productHandler.Update)
		r.Delete("/{id}", productHandler.Delete)
	})

	// Orders
	r.Route("/api/orders", func(r chi.Router) {
		// OptionalAuth : accessible sans compte (guest checkout), mais si un
		// token valide est envoyé, l'acheteur est bien identifié comme
		// connecté (sinon SaleHandler.Create le traite toujours comme un
		// invité et exige un email, même déjà connecté).
		r.With(middleware.OptionalAuth(jwtManager)).Post("/", saleHandler.Create)
		r.Get("/status", saleHandler.CheckoutStatus) // Public (suivi par token, ex: /api/orders/status?token=...)
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAuth(jwtManager))
			r.Get("/", saleHandler.List)
			r.Get("/{id}", saleHandler.Get)
		})
	})

	// Notifications in-app
	r.Route("/api/notifications", func(r chi.Router) {
		r.Use(middleware.RequireAuth(jwtManager))
		r.Get("/", notificationHandler.List)
		r.Get("/unread-count", notificationHandler.UnreadCount)
		r.Put("/read-all", notificationHandler.MarkAllRead)
		r.Put("/{id}/read", notificationHandler.MarkRead)
	})

	// Closer (affiliation) — liens + stats (email vérifié requis)
	r.Route("/api/closer", func(r chi.Router) {
		r.Use(middleware.RequireAuth(jwtManager))
		r.Use(middleware.RequireRole(model.RoleCloser))
		r.Use(middleware.RequireVerifiedEmail(userRepo))
		r.Post("/links", closerHandler.CreateLink)
		r.Get("/links", closerHandler.ListLinks)
	})

	// Redirection publique d'un lien d'affiliation (compte les clics)
	r.Get("/r/{slug}", closerHandler.Redirect)

	// Lien de partage produit (carte Open Graph pour WhatsApp/Facebook/etc.)
	r.Get("/p/{id}", productHandler.Share)

	// Flux produits pour les plateformes publicitaires (import par URL)
	r.Get("/feed/google-merchant.xml", feedHandler.GoogleMerchant)
	r.Get("/feed/facebook.csv", feedHandler.Facebook)

	// Sitemap XML (produits + boutiques) — généré à la demande côté backend,
	// voir le commentaire sur FeedHandler.Sitemap pour le pourquoi.
	r.Get("/sitemap.xml", feedHandler.Sitemap)

	// Webhooks (pas de JWT)
	r.Route("/api/webhooks", func(r chi.Router) {
		r.Post("/pawapay", webhookHandler.PawaPayWebhook)
		r.Post("/pawapay/payout", webhookHandler.PawaPayPayoutWebhook)
		r.Post("/pawapay/refund", webhookHandler.PawaPayRefundWebhook)
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
			r.Get("/support", realtimeHandler.SupportWS)
		})
	})

	// Livraison
	if deliveryHandler != nil {
		r.Route("/api/orders/{id}/delivery", func(r chi.Router) {
			r.Use(middleware.RequireAuth(jwtManager))
			r.Post("/", deliveryHandler.Generate)
		})
		r.Post("/api/orders/delivery", deliveryHandler.GenerateByToken) // Public (acheteur invité, par checkout_token)
		r.Get("/api/delivery/{token}", deliveryHandler.Download)
	}

	// Versements vendeur (téléphone vérifié requis pour les versements)
	r.Route("/api/vendor", func(r chi.Router) {
		r.Use(middleware.RequireAuth(jwtManager))
		r.Use(middleware.RequireRole(model.RoleVendeur))
		r.Get("/earnings", payoutHandler.Earnings)
		r.Get("/payout-limits", payoutHandler.Limits)
		r.Get("/payout-method", payoutHandler.GetPayoutMethod)
		r.Put("/payout-method", payoutHandler.SetPayoutMethod)
		r.With(middleware.RequireVerifiedPhone(userRepo)).Post("/payouts", payoutHandler.Create)
		r.Get("/payouts", payoutHandler.Earnings)
		r.Get("/sales", saleHandler.ListVendor)
	})

	// Routes admin (authentifié + admin). Un admin sans scope assigné garde
	// l'accès complet (legacy) ; RequireAdminScope filtre les admins restreints.
	r.Route("/api/admin", func(r chi.Router) {
		r.Use(middleware.RequireAuth(jwtManager))
		r.Use(middleware.RequireAdmin)

		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAdminScope(model.AdminPermModeration))
			r.Get("/products/pending", adminHandler.PendingProducts)
			r.Get("/products", adminHandler.ListProducts)
			r.Put("/products/{id}/moderate", adminHandler.Moderate)
			r.Delete("/products/{id}", adminHandler.ConfirmDeletion)
			r.Put("/products/{id}/cancel-deletion", adminHandler.CancelDeletion)
		})

		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAdminScope(model.AdminPermUsers))
			r.Get("/users", adminHandler.Users)
			r.Put("/users/{id}/role", adminHandler.SetRole)
			r.Put("/users/{id}/suspend", adminHandler.SuspendUser)
			r.Put("/users/{id}/reactivate", adminHandler.ReactivateUser)
		})

		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAdminScope(model.AdminPermFinance))
			r.Get("/stats", adminHandler.Stats)
			r.Get("/sales", adminHandler.Sales)
			r.Get("/analytics", adminHandler.Analytics)
			r.Post("/sales/{id}/refund", adminHandler.RefundSale)
			r.Get("/settings", adminHandler.GetSettings)
			r.Put("/settings", adminHandler.UpdateSettings)
			r.Get("/payouts", adminHandler.Payouts)
			r.Post("/payouts/{id}/retry", adminHandler.RetryPayout)
			r.Get("/activity", adminHandler.ActivityFeed)
			// Clé pour la création de produit automatisée (voir /api/automation/products ci-dessous).
			r.Get("/automation/key", adminHandler.GetAutomationKey)
			r.Post("/automation/key/regenerate", adminHandler.RegenerateAutomationKey)

			// Programme de reversement automatique ("Fidélisation") — cagnotte,
			// destinataires, historique des versements.
			r.Get("/donations", donationHandler.Get)
			r.Post("/donations/recipients", donationHandler.CreateRecipient)
			r.Put("/donations/recipients/{id}", donationHandler.UpdateRecipient)
			r.Delete("/donations/recipients/{id}", donationHandler.DeleteRecipient)
			r.Post("/donations/payouts/{id}/retry", donationHandler.RetryPayout)
		})

		// Notifications : accessible à tout admin (même restreint), pour que
		// chacun voie au moins ce qui concerne son propre périmètre.
		r.Get("/notifications", adminHandler.Notifications)

		// Support : accessible à tout admin (pas de scope dédié) — n'importe
		// quel agent peut prendre en charge un ticket.
		r.Put("/tickets/{id}/claim", ticketHandler.Claim)
		r.Put("/tickets/{id}/assign", ticketHandler.Assign)
		r.Get("/tickets/assignees", ticketHandler.Assignees)

		// Widget de contact support public : registre des agents notifiés +
		// historique des demandes. Même accès que les tickets (tout admin).
		r.Get("/support-agents", supportContactHandler.ListAgents)
		r.Post("/support-agents", supportContactHandler.CreateAgent)
		r.Put("/support-agents/{id}", supportContactHandler.UpdateAgent)
		r.Delete("/support-agents/{id}", supportContactHandler.DeleteAgent)
		r.Get("/support-contacts", supportContactHandler.ListContacts)

		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAdminScope(model.AdminPermInfra))
			r.Get("/system/health", adminHandler.SystemHealth)
		})

		// Gestion des accès elle-même : réservée aux admins non restreints,
		// pour qu'un admin scoped ne puisse pas s'auto-accorder plus de droits.
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireUnrestrictedAdmin)
			r.Get("/admins", adminHandler.Admins)
			r.Put("/users/{id}/admin", adminHandler.SetAdminStatus)
			r.Put("/admins/{id}/permission", adminHandler.SetAdminPermission)
		})
	})

	// Création de produit automatisée (script/IA externe) — accepte soit une
	// session admin classique, soit la clé d'automatisation dédiée (voir
	// GET/POST /api/admin/automation/key et middleware.RequireAutomation).
	r.Route("/api/automation/products", func(r chi.Router) {
		r.Use(middleware.RequireAutomation(jwtManager, func(ctx context.Context) string {
			return settingsRepo.Get(ctx, model.SettingAutomationAPIKey, "")
		}))
		r.Post("/", productHandler.AutoCreate)
		r.Put("/{id}/file", productHandler.AttachFile)
		r.Put("/{id}", productHandler.UpdateAutomation)
		r.Put("/{id}/cover", productHandler.UpdateAutomationCover)
	})

	// Widget de contact support public (visiteur non authentifié, sans compte
	// requis) — soumis à la limite de débit globale par IP (voir plus haut).
	r.Post("/api/support/contact", supportContactHandler.Contact)

	// Support tickets (utilisateur connecté, admin pour tous)
	r.Route("/api/tickets", func(r chi.Router) {
		r.Use(middleware.RequireAuth(jwtManager))
		r.Post("/", ticketHandler.Create)
		r.Get("/", ticketHandler.List)
		r.Get("/{id}/messages", ticketHandler.GetMessages)
		r.Post("/{id}/messages", ticketHandler.AddMessage)
	})

	// Génère le slug des produits créés avant l'introduction de la colonne
	// (voir migration 019). Idempotent, rapide (quelques lignes), fait avant
	// d'accepter du trafic pour que sitemap/flux/partage servent tout de
	// suite des URLs lisibles plutôt que l'UUID.
	if err := productRepo.BackfillSlugs(ctx); err != nil {
		log.Printf("WARNING: échec du rattrapage des slugs produits: %v", err)
	}

	// Relance les aperçus filigranés restés bloqués "pending" suite à un
	// redémarrage précédent survenu pendant leur génération.
	go productHandler.RecoverStuckPreviews(context.Background())

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
