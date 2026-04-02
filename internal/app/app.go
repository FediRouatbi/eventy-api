package app

import (
	"database/sql"
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
	"eventy-api/internal/users"
	"net/http"
)

type App struct {
	Router http.Handler
	db     *sql.DB
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
	categoriesRepository := categories.NewRepository(queries)
	categoriesService := categories.NewService(categoriesRepository)
	categoriesHandler := categories.NewHandler(categoriesService)
	eventsRepository := events.NewRepository(queries)
	eventsService := events.NewService(eventsRepository)
	eventsHandler := events.NewHandler(eventsService)
	usersRepository := users.NewRepository(queries)
	usersService := users.NewService(usersRepository)
	usersHandler := users.NewHandler(usersService)
	authMiddleware := httpmiddleware.NewAuthMiddleware(tokenManager)

	return &App{
		Router: router.New(authHandler, categoriesHandler, eventsHandler, usersHandler, authMiddleware),
		db:     postgresDB,
	}, nil
}

func (a *App) Close() error {
	if a.db == nil {
		return nil
	}

	return a.db.Close()
}
