package users

import (
	"context"
	"strings"

	"github.com/google/uuid"
)

type Service struct {
	repository      *Repository
	firebaseDeleter FirebaseDeleter
}

type FirebaseDeleter interface {
	DeleteUser(ctx context.Context, uid string) error
}

func NewService(repository *Repository, firebaseDeleter FirebaseDeleter) *Service {
	return &Service{
		repository:      repository,
		firebaseDeleter: firebaseDeleter,
	}
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

func (s *Service) DeleteProfile(ctx context.Context, userID uuid.UUID) error {
	profile, err := s.repository.GetProfileByID(ctx, userID)
	if err != nil {
		return err
	}

	if profile.FirebaseUID != nil && s.firebaseDeleter != nil {
		if err := s.firebaseDeleter.DeleteUser(ctx, *profile.FirebaseUID); err != nil {
			return err
		}
	}

	return s.repository.DeleteProfile(ctx, userID)
}
