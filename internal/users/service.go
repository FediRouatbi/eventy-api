package users

import (
	"context"

	"github.com/google/uuid"
)

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) GetProfile(ctx context.Context, userID uuid.UUID) (Profile, error) {
	return s.repository.GetProfileByID(ctx, userID)
}
