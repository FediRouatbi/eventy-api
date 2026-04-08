package categories

import (
	"context"
	"strings"

	"eventy-api/internal/events"

	"github.com/google/uuid"
)

type Service struct {
	repository       *Repository
	eventsRepository *events.Repository
}

func NewService(repository *Repository, eventsRepository *events.Repository) *Service {
	return &Service{
		repository:       repository,
		eventsRepository: eventsRepository,
	}
}

func (s *Service) Create(ctx context.Context, input CreateCategoryInput) (Category, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Slug = strings.TrimSpace(input.Slug)
	input.Description = strings.TrimSpace(input.Description)
	input.ImageURL = strings.TrimSpace(input.ImageURL)

	if err := validateCreateCategoryInput(input); err != nil {
		return Category{}, err
	}

	return s.repository.Create(ctx, input)
}

func (s *Service) Update(ctx context.Context, categoryID uuid.UUID, input UpdateCategoryInput) (Category, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Slug = strings.TrimSpace(input.Slug)
	input.Description = strings.TrimSpace(input.Description)
	input.ImageURL = strings.TrimSpace(input.ImageURL)

	if err := validateUpdateCategoryInput(input); err != nil {
		return Category{}, err
	}

	return s.repository.Update(ctx, categoryID, input)
}

func (s *Service) List(ctx context.Context) ([]Category, error) {
	return s.repository.List(ctx)
}

func (s *Service) Delete(ctx context.Context, categoryID uuid.UUID) error {
	return s.repository.Delete(ctx, categoryID)
}

func (s *Service) GetPublicBySlug(ctx context.Context, slug string) (PublicCategoryDetail, error) {
	category, err := s.repository.GetBySlug(ctx, slug)
	if err != nil {
		return PublicCategoryDetail{}, err
	}

	publicEvents, err := s.eventsRepository.ListPublic(ctx)
	if err != nil {
		return PublicCategoryDetail{}, err
	}

	categoryEvents := make([]events.PublicEvent, 0)
	for _, event := range publicEvents {
		if event.CategoryID == category.ID {
			categoryEvents = append(categoryEvents, event)
		}
	}

	return PublicCategoryDetail{
		Category: category,
		Events:   categoryEvents,
	}, nil
}
