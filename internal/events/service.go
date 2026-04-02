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

func (s *Service) List(ctx context.Context, claims *jwt.Claims) ([]Event, error) {
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
