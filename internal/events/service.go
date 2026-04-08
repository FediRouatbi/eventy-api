package events

import (
	"context"
	"strings"

	"eventy-api/internal/platform/jwt"
	"eventy-api/internal/platform/roles"

	"github.com/google/uuid"
)

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{repository: repository}
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
