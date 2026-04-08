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
		ID:          categoryID.String(),
		Name:        strings.TrimSpace(input.Name),
		Slug:        strings.TrimSpace(input.Slug),
		Description: nullableString(strings.TrimSpace(input.Description)),
		ImageURL:    nullableString(imageURL),
	})
	if err != nil {
		switch {
		case isUniqueViolation(err) && hasConstraint(err, "categories.name"):
			return Category{}, ErrCategoryNameUsed
		case isUniqueViolation(err) && hasConstraint(err, "categories.slug"):
			return Category{}, ErrCategorySlugUsed
		case isUniqueViolation(err):
			return Category{}, ErrCategoryExists
		}

		return Category{}, err
	}

	return mapCategory(dbCategory)
}

func (r *Repository) Update(ctx context.Context, categoryID uuid.UUID, input UpdateCategoryInput) (Category, error) {
	dbCategory, err := r.queries.UpdateCategory(ctx, sqlc.UpdateCategoryParams{
		ID:          categoryID.String(),
		Name:        strings.TrimSpace(input.Name),
		Slug:        strings.TrimSpace(input.Slug),
		Description: nullableString(strings.TrimSpace(input.Description)),
		ImageURL:    nullableString(strings.TrimSpace(input.ImageURL)),
	})
	if err != nil {
		switch {
		case err == sql.ErrNoRows:
			return Category{}, ErrCategoryNotFound
		case isUniqueViolation(err) && hasConstraint(err, "categories.name"):
			return Category{}, ErrCategoryNameUsed
		case isUniqueViolation(err) && hasConstraint(err, "categories.slug"):
			return Category{}, ErrCategorySlugUsed
		case isUniqueViolation(err):
			return Category{}, ErrCategoryExists
		}

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

func (r *Repository) GetBySlug(ctx context.Context, slug string) (Category, error) {
	categories, err := r.List(ctx)
	if err != nil {
		return Category{}, err
	}

	normalizedSlug := strings.TrimSpace(slug)
	for _, category := range categories {
		if category.Slug == normalizedSlug {
			return category, nil
		}
	}

	return Category{}, ErrCategoryNotFound
}

func (r *Repository) Delete(ctx context.Context, categoryID uuid.UUID) error {
	rowsAffected, err := r.queries.DeleteCategory(ctx, categoryID.String())
	if err != nil {
		if isForeignKeyViolation(err) && hasConstraint(err, "fk_events_category") {
			return ErrCategoryHasEvents
		}

		return err
	}

	if rowsAffected == 0 {
		return ErrCategoryNotFound
	}

	return nil
}

func mapCategory(dbCategory sqlc.Category) (Category, error) {
	categoryID, err := uuid.Parse(dbCategory.ID)
	if err != nil {
		return Category{}, err
	}

	return Category{
		ID:          categoryID,
		Name:        dbCategory.Name,
		Slug:        dbCategory.Slug,
		Description: dbCategory.Description.String,
		ImageURL:    dbCategory.ImageURL.String,
		CreatedAt:   dbCategory.CreatedAt,
		UpdatedAt:   dbCategory.UpdatedAt,
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
