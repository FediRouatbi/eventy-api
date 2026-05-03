package app

import (
	"context"
	"database/sql"
	"eventy-api/internal/admins"
	"eventy-api/internal/auth"
	"eventy-api/internal/categories"
	"eventy-api/internal/config"
	"eventy-api/internal/events"
	httpmiddleware "eventy-api/internal/http/middleware"
	"eventy-api/internal/http/router"
	"eventy-api/internal/notifications"
	pgdb "eventy-api/internal/platform/db"
	"eventy-api/internal/platform/db/sqlc"
	"eventy-api/internal/platform/email"
	eventyfirebase "eventy-api/internal/platform/firebase"
	"eventy-api/internal/platform/jwt"
	"eventy-api/internal/tickets"
	"eventy-api/internal/users"
	"log"
	"net/http"
	"time"
)

type App struct {
	Router http.Handler
	db     *sql.DB
	stop   context.CancelFunc
}

func New(cfg config.Config) (*App, error) {
	postgresDB, err := pgdb.Open(cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}

	queries := sqlc.New(postgresDB)
	tokenManager := jwt.NewManager(cfg.JWTSecret, cfg.JWTIssuer, cfg.JWTTTL)
	firebaseServices, err := eventyfirebase.NewServices(context.Background(), eventyfirebase.Config{
		ProjectID:       cfg.FirebaseProjectID,
		CredentialsFile: cfg.FirebaseCredentialsFile,
		CredentialsJSON: cfg.FirebaseCredentialsJSON,
		GoogleClientIDs: cfg.GoogleClientIDs,
	})
	if err != nil {
		return nil, err
	}
	registrationMailer := email.NewMailjetMailer(
		cfg.MailjetAPIKey,
		cfg.MailjetSecretKey,
		cfg.MailjetFromEmail,
		cfg.MailjetFromName,
	)
	authRepository := auth.NewRepository(queries)
	authService := auth.NewService(authRepository, tokenManager, firebaseServices.FirebaseVerifier, cfg.RefreshTokenTTL)
	authHandler := auth.NewHandler(authService, auth.CookieSettings{
		RefreshTokenName: cfg.RefreshCookieName,
		Domain:           cfg.RefreshCookieDomain,
		Secure:           cfg.RefreshCookieSecure,
	})
	adminsRepository := admins.NewRepository(postgresDB)
	adminsService := admins.NewService(adminsRepository)
	adminsHandler := admins.NewHandler(adminsService)
	categoriesRepository := categories.NewRepository(queries)
	eventsRepository := events.NewRepository(postgresDB, queries)
	ticketsRepository := tickets.NewRepository(postgresDB)
	eventsService := events.NewService(eventsRepository, events.StripeConfig{
		SecretKey:     cfg.StripeSecretKey,
		WebhookSecret: cfg.StripeWebhookKey,
		SuccessURL:    cfg.StripeSuccessURL,
		CancelURL:     cfg.StripeCancelURL,
	}, cfg.WebBaseURL, registrationMailer, ticketsRepository)
	eventsHandler := events.NewHandler(eventsService)
	categoriesService := categories.NewService(categoriesRepository, eventsRepository)
	categoriesHandler := categories.NewHandler(categoriesService)
	usersRepository := users.NewRepository(queries)
	usersService := users.NewService(usersRepository, firebaseServices.FirebaseVerifier)
	usersHandler := users.NewHandler(usersService)
	notificationsRepository := notifications.NewRepository(postgresDB)
	notificationsService := notifications.NewService(notificationsRepository)
	notificationsHandler := notifications.NewHandler(notificationsService)
	_ = notifications.NewSender(firebaseServices.Messaging)
	ticketsService := tickets.NewService(ticketsRepository)
	ticketsHandler := tickets.NewHandler(ticketsService)
	authMiddleware := httpmiddleware.NewAuthMiddleware(tokenManager)
	corsMiddleware := httpmiddleware.NewCORS(cfg.CORSAllowedOrigins)

	ctx, stop := context.WithCancel(context.Background())
	startMaintenance(ctx, eventsRepository)

	return &App{
		Router: router.New(adminsHandler, authHandler, categoriesHandler, eventsHandler, notificationsHandler, ticketsHandler, usersHandler, authMiddleware, corsMiddleware),
		db:     postgresDB,
		stop:   stop,
	}, nil
}

func startMaintenance(ctx context.Context, eventsRepository *events.Repository) {
	ticker := time.NewTicker(1 * time.Minute)

	go func() {
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := eventsRepository.CompletePastEventSessions(context.Background()); err != nil {
					log.Printf("maintenance: complete past sessions: %v", err)
				}
			}
		}
	}()
}

func (a *App) Close() error {
	if a.stop != nil {
		a.stop()
	}

	if a.db == nil {
		return nil
	}

	return a.db.Close()
}
