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
	pgdb "eventy-api/internal/platform/db"
	"eventy-api/internal/platform/db/sqlc"
	"eventy-api/internal/platform/email"
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
	registrationMailer := email.NewMailjetMailer(
		cfg.MailjetAPIKey,
		cfg.MailjetSecretKey,
		cfg.MailjetFromEmail,
		cfg.MailjetFromName,
	)
	authRepository := auth.NewRepository(queries)
	authService := auth.NewService(authRepository, tokenManager, registrationMailer, cfg.RegisterOTPTTL, cfg.RefreshTokenTTL)
	authHandler := auth.NewHandler(authService)
	adminsRepository := admins.NewRepository(postgresDB)
	adminsService := admins.NewService(adminsRepository)
	adminsHandler := admins.NewHandler(adminsService)
	categoriesRepository := categories.NewRepository(queries)
	eventsRepository := events.NewRepository(postgresDB, queries)
	eventsService := events.NewService(eventsRepository, events.StripeConfig{
		SecretKey:     cfg.StripeSecretKey,
		WebhookSecret: cfg.StripeWebhookKey,
		SuccessURL:    cfg.StripeSuccessURL,
		CancelURL:     cfg.StripeCancelURL,
	})
	eventsHandler := events.NewHandler(eventsService)
	categoriesService := categories.NewService(categoriesRepository, eventsRepository)
	categoriesHandler := categories.NewHandler(categoriesService)
	usersRepository := users.NewRepository(queries)
	usersService := users.NewService(usersRepository)
	usersHandler := users.NewHandler(usersService)
	ticketsRepository := tickets.NewRepository(postgresDB)
	ticketsService := tickets.NewService(ticketsRepository)
	ticketsHandler := tickets.NewHandler(ticketsService)
	authMiddleware := httpmiddleware.NewAuthMiddleware(tokenManager)

	ctx, stop := context.WithCancel(context.Background())
	startMaintenance(ctx, eventsRepository)

	return &App{
		Router: router.New(adminsHandler, authHandler, categoriesHandler, eventsHandler, ticketsHandler, usersHandler, authMiddleware),
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
