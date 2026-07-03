package notifications

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

func (s *Service) RegisterDeviceToken(ctx context.Context, userID uuid.UUID, input RegisterDeviceTokenInput) (MessageResponse, error) {
	input.Token = strings.TrimSpace(input.Token)
	input.Provider = strings.TrimSpace(input.Provider)
	input.Platform = strings.TrimSpace(input.Platform)
	input.DeviceName = strings.TrimSpace(input.DeviceName)

	if err := validateRegisterDeviceTokenInput(input); err != nil {
		return MessageResponse{}, err
	}

	if err := s.repository.UpsertDeviceToken(ctx, userID, input); err != nil {
		return MessageResponse{}, err
	}

	return MessageResponse{Message: "device token registered successfully"}, nil
}

func (s *Service) UnregisterDeviceToken(ctx context.Context, userID uuid.UUID, token string) (MessageResponse, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return MessageResponse{}, ErrInvalidDeviceToken
	}

	if err := s.repository.RevokeUserDeviceToken(ctx, userID, token); err != nil {
		return MessageResponse{}, err
	}

	return MessageResponse{Message: "device token unregistered successfully"}, nil
}
