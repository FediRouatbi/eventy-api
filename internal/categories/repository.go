package categories

import (
	"context"
	"database/sql"
	"strings"

	"eventy-api/internal/platform/db/sqlc"

	"github.com/google/uuid"
)

type Repository struct {
	queries *sqlc.Queries
}

func NewRepository(queries *sqlc.Queries) *Repository {
	return &Repository{queries: queries}
}

func (r *Repository) Create(ctx context.Context, input CreateCategoryInput) (Category, error) {
	categoryID := uuid.New()
	imageURL := strings.TrimSpace(input.ImageURL)

	dbCategory, err := r.queries.CreateCategory(ctx, sqlc.CreateCategoryParams{
		ID:       categoryID.String(),
		Name:     strings.TrimSpace(input.Name),
		Slug:     strings.TrimSpace(input.Slug),
		ImageURL: nullableString(imageURL),
	})
	if err != nil {
		return Category{}, err
	}

	return mapCategory(dbCategory)
}

func (r *Repository) List(ctx context.Context) ([]Category, error) {
	dbCategories, err := r.queries.ListCategories(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]Category, 0, len(dbCategories))
	for _, dbCategory := range dbCategories {
		item, err := mapCategory(dbCategory)
		if err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	return items, nil
}

func mapCategory(dbCategory sqlc.Category) (Category, error) {
	categoryID, err := uuid.Parse(dbCategory.ID)
	if err != nil {
		return Category{}, err
	}

	return Category{
		ID:        categoryID,
		Name:      dbCategory.Name,
		Slug:      dbCategory.Slug,
		ImageURL:  dbCategory.ImageURL.String,
		CreatedAt: dbCategory.CreatedAt,
		UpdatedAt: dbCategory.UpdatedAt,
	}, nil
}

func nullableString(value string) sql.NullString {
	if value == "" {
		return sql.NullString{}
	}

	return sql.NullString{
		String: value,
		Valid:  true,
	}
}
