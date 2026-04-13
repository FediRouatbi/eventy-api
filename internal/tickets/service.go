package tickets

import (
	"context"
	"math"
	"strings"

	"eventy-api/internal/platform/jwt"
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
	return s.repository.CheckInByCode(ctx, claims, code)
}
