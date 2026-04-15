package tickets

import (
	"context"
	"math"
	"strings"

	"eventy-api/internal/platform/jwt"

	"github.com/google/uuid"
)

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) ListMyTickets(ctx context.Context, claims *jwt.Claims, limit int) ([]Ticket, error) {
	email := strings.ToLower(strings.TrimSpace(claims.Email))
	if email == "" {
		return nil, nil
	}

	if limit <= 0 {
		limit = 50
	}
	limit = int(math.Min(float64(limit), 200))

	return s.repository.ListByCustomerEmail(ctx, email, limit)
}

func (s *Service) CheckInTicket(ctx context.Context, claims *jwt.Claims, code string) (CheckInResult, error) {
	return s.repository.CheckInByCode(ctx, claims, code, nil, nil)
}

func (s *Service) CheckInTicketForSession(ctx context.Context, claims *jwt.Claims, code string, eventID string, sessionID string) (CheckInResult, error) {
	eventID = strings.TrimSpace(eventID)
	sessionID = strings.TrimSpace(sessionID)

	var expectedEventID *uuid.UUID
	if eventID != "" {
		parsed, err := uuid.Parse(eventID)
		if err != nil {
			return CheckInResult{}, ErrInvalidEventID
		}
		expectedEventID = &parsed
	}

	var expectedSessionID *uuid.UUID
	if sessionID != "" {
		parsed, err := uuid.Parse(sessionID)
		if err != nil {
			return CheckInResult{}, ErrInvalidSessionID
		}
		expectedSessionID = &parsed
	}

	return s.repository.CheckInByCode(ctx, claims, code, expectedEventID, expectedSessionID)
}

func (s *Service) GetTicketByCode(ctx context.Context, claims *jwt.Claims, code string) (Ticket, error) {
	return s.repository.GetByCode(ctx, claims, code)
}
