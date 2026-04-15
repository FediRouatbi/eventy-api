package router

import (
	"net/http"

	"eventy-api/internal/admins"
	"eventy-api/internal/auth"
	"eventy-api/internal/categories"
	"eventy-api/internal/docs"
	"eventy-api/internal/events"
	"eventy-api/internal/http/middleware"
	"eventy-api/internal/platform/roles"
	"eventy-api/internal/tickets"
	"eventy-api/internal/users"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

func New(adminsHandler *admins.Handler, authHandler *auth.Handler, categoriesHandler *categories.Handler, eventsHandler *events.Handler, ticketsHandler *tickets.Handler, usersHandler *users.Handler, authMiddleware *middleware.AuthMiddleware) http.Handler {
	r := chi.NewRouter()

	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Logger)
	r.Use(middleware.CORS)
	r.Use(middleware.Recoverer)

	r.Get("/docs", docs.DocsHandler())
	r.Get("/openapi.json", docs.OpenAPIJSONHandler())

	r.Route("/v1", func(r chi.Router) {
		r.Post("/stripe/webhook", eventsHandler.StripeWebhook)

		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", authHandler.Register)
			r.Post("/register/resend-otp", authHandler.ResendRegisterOTP)
			r.Post("/register/verify", authHandler.VerifyRegisterOTP)
			r.Post("/forgot-password", authHandler.ForgotPassword)
			r.Post("/reset-password", authHandler.ResetPassword)
			r.Post("/login", authHandler.Login)
			r.Post("/refresh", authHandler.RefreshSession)
			r.With(authMiddleware.RequireAuth).Patch("/change-password", authHandler.ChangePassword)
			r.With(authMiddleware.RequireAuth).Post("/logout", authHandler.Logout)
		})

		r.Route("/categories", func(r chi.Router) {
			r.Get("/", categoriesHandler.List)
			r.With(authMiddleware.RequireAuth, authMiddleware.RequireRoles(roles.SuperAdmin)).Post("/", categoriesHandler.Create)
			r.With(authMiddleware.RequireAuth, authMiddleware.RequireRoles(roles.SuperAdmin)).Patch("/{categoryID}", categoriesHandler.Update)
			r.With(authMiddleware.RequireAuth, authMiddleware.RequireRoles(roles.SuperAdmin)).Delete("/{categoryID}", categoriesHandler.Delete)
		})

		r.Route("/public", func(r chi.Router) {
			r.Get("/categories/{categorySlug}", categoriesHandler.GetPublicBySlug)
			r.Get("/events", eventsHandler.ListPublic)
			r.Get("/events/{eventID}", eventsHandler.GetPublicByID)
			r.Post("/reservations", eventsHandler.UpsertReservation)
			r.Get("/reservations/{reservationID}", eventsHandler.GetReservation)
			r.Delete("/reservations/{reservationID}", eventsHandler.DeleteReservation)
			r.Post("/checkout-orders", eventsHandler.CreateCheckoutOrder)
			r.Get("/checkout-orders/{orderID}", eventsHandler.GetCheckoutOrder)
			r.Post("/checkout-orders/{orderID}/stripe-session", eventsHandler.CreateStripeCheckoutSession)
			r.Get("/stripe-sessions/{stripeSessionID}/checkout-order", eventsHandler.GetCheckoutOrderByStripeSession)
		})

		r.With(authMiddleware.RequireAuth).Get("/orders", eventsHandler.ListMyCheckoutOrders)
		r.With(authMiddleware.RequireAuth).Get("/orders/{orderID}", eventsHandler.GetMyCheckoutOrder)
		r.With(authMiddleware.RequireAuth).Get("/orders/stripe-sessions/{stripeSessionID}/checkout-order", eventsHandler.GetMyCheckoutOrderByStripeSession)

		r.With(authMiddleware.RequireAuth).Get("/tickets", ticketsHandler.ListMyTickets)
		r.With(authMiddleware.RequireAuth, authMiddleware.RequireRoles(roles.SuperAdmin, roles.OrganizerAdmin)).Get("/tickets/code/{ticketCode}", ticketsHandler.GetByCode)
		r.With(authMiddleware.RequireAuth, authMiddleware.RequireRoles(roles.SuperAdmin, roles.OrganizerAdmin)).Post("/tickets/check-in", ticketsHandler.CheckInTicket)

		r.Route("/admins", func(r chi.Router) {
			r.With(authMiddleware.RequireAuth, authMiddleware.RequireRoles(roles.SuperAdmin, roles.OrganizerAdmin)).Get("/overview", adminsHandler.GetOverview)
			r.With(authMiddleware.RequireAuth, authMiddleware.RequireRoles(roles.SuperAdmin, roles.OrganizerAdmin)).Get("/payments", adminsHandler.GetPayments)
			r.With(authMiddleware.RequireAuth, authMiddleware.RequireRoles(roles.SuperAdmin)).Get("/organizers", adminsHandler.ListOrganizers)
			r.With(authMiddleware.RequireAuth, authMiddleware.RequireRoles(roles.SuperAdmin)).Post("/organizers", adminsHandler.CreateOrganizerAdmin)
			r.With(authMiddleware.RequireAuth, authMiddleware.RequireRoles(roles.SuperAdmin)).Get("/organizers/{organizerID}", adminsHandler.GetOrganizer)
			r.With(authMiddleware.RequireAuth, authMiddleware.RequireRoles(roles.SuperAdmin)).Patch("/organizers/{organizerID}", adminsHandler.UpdateOrganizer)
			r.With(authMiddleware.RequireAuth, authMiddleware.RequireRoles(roles.SuperAdmin)).Delete("/organizers/{organizerID}", adminsHandler.DeleteOrganizer)
			r.With(authMiddleware.RequireAuth, authMiddleware.RequireRoles(roles.SuperAdmin)).Get("/organizers/{organizerID}/admins", adminsHandler.ListOrganizerAdmins)
			r.With(authMiddleware.RequireAuth, authMiddleware.RequireRoles(roles.SuperAdmin)).Post("/organizers/{organizerID}/admins", adminsHandler.AddOrganizerAdmin)
			r.With(authMiddleware.RequireAuth, authMiddleware.RequireRoles(roles.SuperAdmin)).Get("/organizers/{organizerID}/admins/{adminID}", adminsHandler.GetOrganizerAdmin)
			r.With(authMiddleware.RequireAuth, authMiddleware.RequireRoles(roles.SuperAdmin)).Patch("/organizers/{organizerID}/admins/{adminID}", adminsHandler.UpdateOrganizerAdmin)
			r.With(authMiddleware.RequireAuth, authMiddleware.RequireRoles(roles.SuperAdmin)).Patch("/organizers/{organizerID}/admins/{adminID}/password", adminsHandler.ResetOrganizerAdminPassword)
			r.With(authMiddleware.RequireAuth, authMiddleware.RequireRoles(roles.SuperAdmin)).Delete("/organizers/{organizerID}/admins/{adminID}", adminsHandler.DeleteOrganizerAdmin)
		})

		r.Route("/events", func(r chi.Router) {
			r.With(authMiddleware.RequireAuth, authMiddleware.RequireRoles(roles.OrganizerAdmin, roles.SuperAdmin)).Get("/", eventsHandler.List)
			r.With(authMiddleware.RequireAuth, authMiddleware.RequireRoles(roles.OrganizerAdmin, roles.SuperAdmin)).Post("/", eventsHandler.Create)
			r.With(authMiddleware.RequireAuth, authMiddleware.RequireRoles(roles.OrganizerAdmin, roles.SuperAdmin)).Get("/{eventID}", eventsHandler.GetByID)
			r.With(authMiddleware.RequireAuth, authMiddleware.RequireRoles(roles.OrganizerAdmin, roles.SuperAdmin)).Patch("/{eventID}", eventsHandler.Update)
			r.With(authMiddleware.RequireAuth, authMiddleware.RequireRoles(roles.OrganizerAdmin, roles.SuperAdmin)).Delete("/{eventID}", eventsHandler.Delete)
			r.With(authMiddleware.RequireAuth, authMiddleware.RequireRoles(roles.OrganizerAdmin, roles.SuperAdmin)).Get("/{eventID}/sessions", eventsHandler.ListSessions)
			r.With(authMiddleware.RequireAuth, authMiddleware.RequireRoles(roles.OrganizerAdmin, roles.SuperAdmin)).Post("/{eventID}/sessions", eventsHandler.CreateSession)
			r.With(authMiddleware.RequireAuth, authMiddleware.RequireRoles(roles.OrganizerAdmin, roles.SuperAdmin)).Patch("/{eventID}/sessions/{sessionID}", eventsHandler.UpdateSession)
			r.With(authMiddleware.RequireAuth, authMiddleware.RequireRoles(roles.OrganizerAdmin, roles.SuperAdmin)).Delete("/{eventID}/sessions/{sessionID}", eventsHandler.DeleteSession)
			r.With(authMiddleware.RequireAuth, authMiddleware.RequireRoles(roles.OrganizerAdmin, roles.SuperAdmin)).Get("/{eventID}/sessions/{sessionID}/ticket-types", eventsHandler.ListTicketTypes)
			r.With(authMiddleware.RequireAuth, authMiddleware.RequireRoles(roles.OrganizerAdmin, roles.SuperAdmin)).Post("/{eventID}/sessions/{sessionID}/ticket-types", eventsHandler.CreateTicketType)
			r.With(authMiddleware.RequireAuth, authMiddleware.RequireRoles(roles.OrganizerAdmin, roles.SuperAdmin)).Patch("/{eventID}/sessions/{sessionID}/ticket-types/{ticketTypeID}", eventsHandler.UpdateTicketType)
			r.With(authMiddleware.RequireAuth, authMiddleware.RequireRoles(roles.OrganizerAdmin, roles.SuperAdmin)).Delete("/{eventID}/sessions/{sessionID}/ticket-types/{ticketTypeID}", eventsHandler.DeleteTicketType)
		})

		r.With(authMiddleware.RequireAuth).Get("/users/me", usersHandler.GetMe)
		r.With(authMiddleware.RequireAuth).Patch("/users/me", usersHandler.UpdateMe)
		r.With(authMiddleware.RequireAuth).Delete("/users/me", usersHandler.DeleteMe)
	})

	return r
}
