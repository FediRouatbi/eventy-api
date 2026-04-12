package users

import (
	"context"
	"strings"

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

func (s *Service) UpdateProfile(ctx context.Context, userID uuid.UUID, input UpdateProfileInput) (Profile, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Email = strings.TrimSpace(strings.ToLower(input.Email))

	if err := validateUpdateProfileInput(input); err != nil {
		return Profile{}, err
	}

	return s.repository.UpdateProfile(ctx, userID, input)
}

func (s *Service) DeleteProfileWithPassword(ctx context.Context, userID uuid.UUID, input DeleteAccountInput) error {
	input.CurrentPassword = strings.TrimSpace(input.CurrentPassword)

	if err := validateDeleteAccountInput(input); err != nil {
		return err
	}

	return s.repository.DeleteProfileWithPassword(ctx, userID, input)
}
