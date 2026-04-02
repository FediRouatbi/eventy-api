package categories

import (
	"context"
	"strings"
)

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) Create(ctx context.Context, input CreateCategoryInput) (Category, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Slug = strings.TrimSpace(input.Slug)
	input.ImageURL = strings.TrimSpace(input.ImageURL)

	if err := validateCreateCategoryInput(input); err != nil {
		return Category{}, err
	}

	return s.repository.Create(ctx, input)
}

func (s *Service) List(ctx context.Context) ([]Category, error) {
	return s.repository.List(ctx)
}
