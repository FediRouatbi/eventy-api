package events

import (
	"context"
	"strings"
	"time"

	"eventy-api/internal/platform/jwt"
	"eventy-api/internal/platform/roles"

	"github.com/google/uuid"
)

type Service struct {
	repository *Repository
	stripeCfg  StripeConfig
}

type StripeConfig struct {
	SecretKey     string
	WebhookSecret string
	SuccessURL    string
	CancelURL     string
}

func NewService(repository *Repository, stripeCfg StripeConfig) *Service {
	return &Service{repository: repository, stripeCfg: stripeCfg}
}

func (s *Service) Create(ctx context.Context, claims *jwt.Claims, input CreateEventInput) (Event, error) {
	input.OrganizerID = strings.TrimSpace(input.OrganizerID)
	input.CategoryID = strings.TrimSpace(input.CategoryID)
	input.Title = strings.TrimSpace(input.Title)
	input.Slug = strings.TrimSpace(input.Slug)
	input.Description = strings.TrimSpace(input.Description)
	input.VenueName = strings.TrimSpace(input.VenueName)
	input.VenueAddress = strings.TrimSpace(input.VenueAddress)
	input.City = strings.TrimSpace(input.City)
	input.Country = strings.TrimSpace(input.Country)
	input.BannerURL = strings.TrimSpace(input.BannerURL)
	input.PosterURL = strings.TrimSpace(input.PosterURL)
	input.Status = strings.TrimSpace(input.Status)
	input.Currency = strings.TrimSpace(strings.ToUpper(input.Currency))

	if err := validateCreateEventInput(input); err != nil {
		return Event{}, err
	}

	organizerID, err := resolveOrganizerScopeForCreate(claims, input.OrganizerID)
	if err != nil {
		return Event{}, err
	}

	categoryID, err := uuid.Parse(input.CategoryID)
	if err != nil {
		return Event{}, ErrInvalidCategoryID
	}

	return s.repository.Create(ctx, organizerID, categoryID, input)
}

func (s *Service) Update(ctx context.Context, claims *jwt.Claims, eventID uuid.UUID, input UpdateEventInput) (Event, error) {
	input.CategoryID = strings.TrimSpace(input.CategoryID)
	input.Title = strings.TrimSpace(input.Title)
	input.Slug = strings.TrimSpace(input.Slug)
	input.Description = strings.TrimSpace(input.Description)
	input.VenueName = strings.TrimSpace(input.VenueName)
	input.VenueAddress = strings.TrimSpace(input.VenueAddress)
	input.City = strings.TrimSpace(input.City)
	input.Country = strings.TrimSpace(input.Country)
	input.BannerURL = strings.TrimSpace(input.BannerURL)
	input.PosterURL = strings.TrimSpace(input.PosterURL)
	input.Status = strings.TrimSpace(input.Status)
	input.Currency = strings.TrimSpace(strings.ToUpper(input.Currency))

	if err := validateUpdateEventInput(input); err != nil {
		return Event{}, err
	}

	event, err := s.getAccessibleEvent(ctx, claims, eventID)
	if err != nil {
		return Event{}, err
	}

	categoryID, err := uuid.Parse(input.CategoryID)
	if err != nil {
		return Event{}, ErrInvalidCategoryID
	}

	updated, err := s.repository.Update(ctx, event.ID, categoryID, input)
	if err != nil {
		return Event{}, err
	}

	return updated, nil
}

func (s *Service) Delete(ctx context.Context, claims *jwt.Claims, eventID uuid.UUID) error {
	if _, err := s.getAccessibleEvent(ctx, claims, eventID); err != nil {
		return err
	}

	return s.repository.Delete(ctx, eventID)
}

func (s *Service) List(ctx context.Context, claims *jwt.Claims) ([]EventListItem, error) {
	switch claims.Role {
	case roles.SuperAdmin:
		return s.repository.List(ctx)
	case roles.OrganizerAdmin:
		if claims.OrganizerID == nil {
			return nil, ErrOrganizerScopeRequired
		}

		return s.repository.ListByOrganizerID(ctx, *claims.OrganizerID)
	default:
		return nil, ErrUnsupportedRole
	}
}

func (s *Service) ListPublic(ctx context.Context) ([]PublicEvent, error) {
	return s.repository.ListPublic(ctx)
}

func (s *Service) GetPublicByID(ctx context.Context, eventID uuid.UUID) (PublicEventDetail, error) {
	return s.repository.GetPublicDetailByID(ctx, eventID)
}

func (s *Service) UpsertReservation(ctx context.Context, input UpsertTicketReservationInput) (TicketReservation, error) {
	input.ReservationID = strings.TrimSpace(input.ReservationID)
	input.ReservationToken = strings.TrimSpace(input.ReservationToken)

	for index := range input.Items {
		input.Items[index].TicketTypeID = strings.TrimSpace(input.Items[index].TicketTypeID)
	}

	return s.repository.UpsertReservation(ctx, input)
}

func (s *Service) GetReservation(ctx context.Context, reservationID uuid.UUID, reservationToken string) (TicketReservation, error) {
	reservationToken = strings.TrimSpace(reservationToken)
	if reservationToken == "" {
		return TicketReservation{}, ErrInvalidReservationToken
	}

	return s.repository.GetReservation(ctx, reservationID, reservationToken)
}

func (s *Service) DeleteReservation(ctx context.Context, reservationID uuid.UUID, reservationToken string) error {
	reservationToken = strings.TrimSpace(reservationToken)
	if reservationToken == "" {
		return ErrInvalidReservationToken
	}

	return s.repository.DeleteReservation(ctx, reservationID, reservationToken)
}

func (s *Service) CreateCheckoutOrder(ctx context.Context, input CreateCheckoutOrderInput) (CheckoutOrder, error) {
	input.ReservationID = strings.TrimSpace(input.ReservationID)
	input.ReservationToken = strings.TrimSpace(input.ReservationToken)
	input.CustomerName = strings.TrimSpace(input.CustomerName)
	input.CustomerEmail = strings.TrimSpace(strings.ToLower(input.CustomerEmail))

	if err := validateCreateCheckoutOrderInput(input); err != nil {
		return CheckoutOrder{}, err
	}

	return s.repository.CreateCheckoutOrder(ctx, input)
}

func (s *Service) GetCheckoutOrder(ctx context.Context, orderID uuid.UUID, orderToken string) (CheckoutOrder, error) {
	orderToken = strings.TrimSpace(orderToken)
	if orderToken == "" {
		return CheckoutOrder{}, ErrInvalidOrderToken
	}

	return s.repository.GetCheckoutOrder(ctx, orderID, orderToken)
}

func (s *Service) CreateStripeCheckoutSession(ctx context.Context, orderID uuid.UUID, orderToken string) (StripeCheckoutSessionResponse, error) {
	if strings.TrimSpace(s.stripeCfg.SecretKey) == "" || strings.TrimSpace(s.stripeCfg.SuccessURL) == "" || strings.TrimSpace(s.stripeCfg.CancelURL) == "" {
		return StripeCheckoutSessionResponse{}, ErrStripeNotConfigured
	}

	order, err := s.repository.GetCheckoutOrder(ctx, orderID, orderToken)
	if err != nil {
		return StripeCheckoutSessionResponse{}, err
	}

	if order.Status != "pending_payment" {
		return StripeCheckoutSessionResponse{}, ErrCheckoutOrderNotPayable
	}

	session, err := s.repository.CreateStripeCheckoutSession(ctx, s.stripeCfg.SecretKey, s.stripeCfg.SuccessURL, s.stripeCfg.CancelURL, order)
	if err != nil {
		return StripeCheckoutSessionResponse{}, err
	}

	return StripeCheckoutSessionResponse{
		SessionID:   session.ID,
		CheckoutURL: session.URL,
		OrderID:     order.ID.String(),
		OrderNumber: order.OrderNumber,
		ExpiresAt:   order.ExpiresAt.Format(time.RFC3339),
	}, nil
}

func (s *Service) GetCheckoutOrderByStripeSessionID(ctx context.Context, stripeSessionID string) (CheckoutOrderSummary, error) {
	stripeSessionID = strings.TrimSpace(stripeSessionID)
	if stripeSessionID == "" {
		return CheckoutOrderSummary{}, ErrInvalidStripeSessionID
	}

	return s.repository.GetCheckoutOrderByStripeSessionID(ctx, stripeSessionID)
}

func (s *Service) HandleStripeWebhook(ctx context.Context, payload []byte, signature string) error {
	if strings.TrimSpace(s.stripeCfg.WebhookSecret) == "" {
		return ErrStripeNotConfigured
	}

	return s.repository.HandleStripeWebhook(ctx, s.stripeCfg.SecretKey, s.stripeCfg.WebhookSecret, payload, signature)
}

func (s *Service) GetByID(ctx context.Context, claims *jwt.Claims, eventID uuid.UUID) (Event, error) {
	event, err := s.repository.GetByID(ctx, eventID)
	if err != nil {
		return Event{}, err
	}

	if !canAccessEvent(claims, event) {
		if claims.Role == roles.OrganizerAdmin && claims.OrganizerID == nil {
			return Event{}, ErrOrganizerScopeRequired
		}

		return Event{}, ErrForbidden
	}

	return event, nil
}

func (s *Service) GetDetailByID(ctx context.Context, claims *jwt.Claims, eventID uuid.UUID) (EventDetail, error) {
	_, err := s.getAccessibleEvent(ctx, claims, eventID)
	if err != nil {
		return EventDetail{}, err
	}

	return s.repository.GetDetailByID(ctx, eventID)
}

func (s *Service) CreateSession(ctx context.Context, claims *jwt.Claims, eventID uuid.UUID, input CreateEventSessionInput) (EventSession, error) {
	input.Status = strings.TrimSpace(input.Status)

	if err := validateEventSessionInput(input.StartsAt, input.EndsAt, input.SalesStartsAt, input.SalesEndsAt, input.Status); err != nil {
		return EventSession{}, err
	}

	if _, err := s.getAccessibleEvent(ctx, claims, eventID); err != nil {
		return EventSession{}, err
	}

	return s.repository.CreateSession(ctx, eventID, input)
}

func (s *Service) ListSessions(ctx context.Context, claims *jwt.Claims, eventID uuid.UUID) ([]EventSession, error) {
	if _, err := s.getAccessibleEvent(ctx, claims, eventID); err != nil {
		return nil, err
	}

	return s.repository.ListSessionsByEventID(ctx, eventID)
}

func (s *Service) UpdateSession(ctx context.Context, claims *jwt.Claims, eventID uuid.UUID, sessionID uuid.UUID, input UpdateEventSessionInput) (EventSession, error) {
	input.Status = strings.TrimSpace(input.Status)

	if err := validateEventSessionInput(input.StartsAt, input.EndsAt, input.SalesStartsAt, input.SalesEndsAt, input.Status); err != nil {
		return EventSession{}, err
	}

	if _, err := s.getAccessibleEvent(ctx, claims, eventID); err != nil {
		return EventSession{}, err
	}

	session, err := s.repository.GetSessionByID(ctx, sessionID)
	if err != nil {
		return EventSession{}, err
	}

	if session.EventID != eventID {
		return EventSession{}, ErrEventSessionNotFound
	}

	return s.repository.UpdateSession(ctx, sessionID, input)
}

func (s *Service) DeleteSession(ctx context.Context, claims *jwt.Claims, eventID uuid.UUID, sessionID uuid.UUID) error {
	if _, err := s.getAccessibleEvent(ctx, claims, eventID); err != nil {
		return err
	}

	session, err := s.repository.GetSessionByID(ctx, sessionID)
	if err != nil {
		return err
	}

	if session.EventID != eventID {
		return ErrEventSessionNotFound
	}

	return s.repository.DeleteSession(ctx, sessionID)
}

func (s *Service) CreateTicketType(ctx context.Context, claims *jwt.Claims, eventID uuid.UUID, sessionID uuid.UUID, input CreateTicketTypeInput) (TicketType, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Description = strings.TrimSpace(input.Description)

	if err := validateTicketTypeInput(input.Name, input.Price, input.Quantity, input.MaxPerOrder); err != nil {
		return TicketType{}, err
	}

	if _, err := s.getAccessibleEvent(ctx, claims, eventID); err != nil {
		return TicketType{}, err
	}

	session, err := s.repository.GetSessionByID(ctx, sessionID)
	if err != nil {
		return TicketType{}, err
	}

	if session.EventID != eventID {
		return TicketType{}, ErrEventSessionNotFound
	}

	return s.repository.CreateTicketType(ctx, sessionID, input)
}

func (s *Service) ListTicketTypes(ctx context.Context, claims *jwt.Claims, eventID uuid.UUID, sessionID uuid.UUID) ([]TicketType, error) {
	if _, err := s.getAccessibleEvent(ctx, claims, eventID); err != nil {
		return nil, err
	}

	session, err := s.repository.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	if session.EventID != eventID {
		return nil, ErrEventSessionNotFound
	}

	return s.repository.ListTicketTypesBySessionID(ctx, sessionID)
}

func (s *Service) UpdateTicketType(ctx context.Context, claims *jwt.Claims, eventID uuid.UUID, sessionID uuid.UUID, ticketTypeID uuid.UUID, input UpdateTicketTypeInput) (TicketType, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Description = strings.TrimSpace(input.Description)

	if err := validateTicketTypeInput(input.Name, input.Price, input.Quantity, input.MaxPerOrder); err != nil {
		return TicketType{}, err
	}

	if _, err := s.getAccessibleEvent(ctx, claims, eventID); err != nil {
		return TicketType{}, err
	}

	session, err := s.repository.GetSessionByID(ctx, sessionID)
	if err != nil {
		return TicketType{}, err
	}

	if session.EventID != eventID {
		return TicketType{}, ErrEventSessionNotFound
	}

	ticketType, err := s.repository.GetTicketTypeByID(ctx, ticketTypeID)
	if err != nil {
		return TicketType{}, err
	}

	if ticketType.EventSessionID != sessionID {
		return TicketType{}, ErrTicketTypeNotFound
	}

	return s.repository.UpdateTicketType(ctx, ticketTypeID, input)
}

func (s *Service) DeleteTicketType(ctx context.Context, claims *jwt.Claims, eventID uuid.UUID, sessionID uuid.UUID, ticketTypeID uuid.UUID) error {
	if _, err := s.getAccessibleEvent(ctx, claims, eventID); err != nil {
		return err
	}

	session, err := s.repository.GetSessionByID(ctx, sessionID)
	if err != nil {
		return err
	}

	if session.EventID != eventID {
		return ErrEventSessionNotFound
	}

	ticketType, err := s.repository.GetTicketTypeByID(ctx, ticketTypeID)
	if err != nil {
		return err
	}

	if ticketType.EventSessionID != sessionID {
		return ErrTicketTypeNotFound
	}

	return s.repository.DeleteTicketType(ctx, ticketTypeID)
}

func (s *Service) getAccessibleEvent(ctx context.Context, claims *jwt.Claims, eventID uuid.UUID) (Event, error) {
	event, err := s.repository.GetByID(ctx, eventID)
	if err != nil {
		return Event{}, err
	}

	if !canAccessEvent(claims, event) {
		if claims.Role == roles.OrganizerAdmin && claims.OrganizerID == nil {
			return Event{}, ErrOrganizerScopeRequired
		}

		return Event{}, ErrForbidden
	}

	return event, nil
}

func resolveOrganizerScopeForCreate(claims *jwt.Claims, requestedOrganizerID string) (uuid.UUID, error) {
	switch claims.Role {
	case roles.SuperAdmin:
		if requestedOrganizerID == "" {
			return uuid.UUID{}, ErrInvalidOrganizerID
		}

		organizerID, err := uuid.Parse(requestedOrganizerID)
		if err != nil {
			return uuid.UUID{}, ErrInvalidOrganizerID
		}

		return organizerID, nil
	case roles.OrganizerAdmin:
		if claims.OrganizerID == nil {
			return uuid.UUID{}, ErrOrganizerScopeRequired
		}

		if requestedOrganizerID != "" && requestedOrganizerID != claims.OrganizerID.String() {
			return uuid.UUID{}, ErrForbidden
		}

		return *claims.OrganizerID, nil
	default:
		return uuid.UUID{}, ErrUnsupportedRole
	}
}

func canAccessEvent(claims *jwt.Claims, event Event) bool {
	switch claims.Role {
	case roles.SuperAdmin:
		return true
	case roles.OrganizerAdmin:
		return claims.OrganizerID != nil && *claims.OrganizerID == event.OrganizerID
	default:
		return false
	}
}
