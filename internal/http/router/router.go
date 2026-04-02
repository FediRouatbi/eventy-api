package router

import (
	"net/http"

	"eventy-api/internal/auth"
	"eventy-api/internal/categories"
	"eventy-api/internal/docs"
	"eventy-api/internal/events"
	"eventy-api/internal/http/middleware"
	"eventy-api/internal/platform/roles"
	"eventy-api/internal/users"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

func New(authHandler *auth.Handler, categoriesHandler *categories.Handler, eventsHandler *events.Handler, usersHandler *users.Handler, authMiddleware *middleware.AuthMiddleware) http.Handler {
	r := chi.NewRouter()

	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/docs", docs.DocsHandler())
	r.Get("/openapi.json", docs.OpenAPIJSONHandler())

	r.Route("/v1", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", authHandler.Register)
			r.Post("/register/resend-otp", authHandler.ResendRegisterOTP)
			r.Post("/register/verify", authHandler.VerifyRegisterOTP)
			r.Post("/forgot-password", authHandler.ForgotPassword)
			r.Post("/reset-password", authHandler.ResetPassword)
			r.Post("/login", authHandler.Login)
			r.Post("/refresh", authHandler.RefreshSession)
			r.With(authMiddleware.RequireAuth).Post("/logout", authHandler.Logout)
		})

		r.Route("/categories", func(r chi.Router) {
			r.Get("/", categoriesHandler.List)
			r.With(authMiddleware.RequireAuth, authMiddleware.RequireRoles(roles.SuperAdmin)).Post("/", categoriesHandler.Create)
		})

		r.Route("/events", func(r chi.Router) {
			r.With(authMiddleware.RequireAuth, authMiddleware.RequireRoles(roles.OrganizerAdmin, roles.SuperAdmin)).Get("/", eventsHandler.List)
			r.With(authMiddleware.RequireAuth, authMiddleware.RequireRoles(roles.OrganizerAdmin, roles.SuperAdmin)).Post("/", eventsHandler.Create)
			r.With(authMiddleware.RequireAuth, authMiddleware.RequireRoles(roles.OrganizerAdmin, roles.SuperAdmin)).Get("/{eventID}", eventsHandler.GetByID)
		})

		r.With(authMiddleware.RequireAuth).Get("/users/me", usersHandler.GetMe)
	})

	return r
}
