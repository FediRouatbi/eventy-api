package sqlc

import (
	"context"
	"database/sql"
	"errors"
)

const createCategoryQuery = `
INSERT INTO categories (
    id,
    name,
    slug,
    description,
    image_url
) VALUES (
    ?,
    ?,
    ?,
    ?,
    ?
)
`

const deleteCategoryQuery = `
DELETE FROM categories
WHERE id = ?
`

const getCategoryByIDQuery = `
SELECT id, name, slug, description, image_url, created_at, updated_at
FROM categories
WHERE id = ?
LIMIT 1
`

const listCategoriesQuery = `
SELECT id, name, slug, description, image_url, created_at, updated_at
FROM categories
ORDER BY name ASC
`

const updateCategoryQuery = `
UPDATE categories
SET
    name = ?,
    slug = ?,
    description = ?,
    image_url = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
`

func (q *Queries) CreateCategory(ctx context.Context, arg CreateCategoryParams) (Category, error) {
	_, err := q.db.ExecContext(ctx, createCategoryQuery, arg.ID, arg.Name, arg.Slug, arg.Description, arg.ImageURL)
	if err != nil {
		return Category{}, err
	}

	return q.GetCategoryByID(ctx, arg.ID)
}

func (q *Queries) DeleteCategory(ctx context.Context, id string) (int64, error) {
	result, err := q.db.ExecContext(ctx, deleteCategoryQuery, id)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

func (q *Queries) GetCategoryByID(ctx context.Context, id string) (Category, error) {
	row := q.db.QueryRowContext(ctx, getCategoryByIDQuery, id)

	var category Category
	err := row.Scan(
		&category.ID,
		&category.Name,
		&category.Slug,
		&category.Description,
		&category.ImageURL,
		&category.CreatedAt,
		&category.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Category{}, sql.ErrNoRows
	}

	return category, err
}

func (q *Queries) ListCategories(ctx context.Context) ([]Category, error) {
	rows, err := q.db.QueryContext(ctx, listCategoriesQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Category
	for rows.Next() {
		var category Category
		if err := rows.Scan(
			&category.ID,
			&category.Name,
			&category.Slug,
			&category.Description,
			&category.ImageURL,
			&category.CreatedAt,
			&category.UpdatedAt,
		); err != nil {
			return nil, err
		}

		items = append(items, category)
	}

	return items, rows.Err()
}

func (q *Queries) UpdateCategory(ctx context.Context, arg UpdateCategoryParams) (Category, error) {
	result, err := q.db.ExecContext(ctx, updateCategoryQuery, arg.Name, arg.Slug, arg.Description, arg.ImageURL, arg.ID)
	if err != nil {
		return Category{}, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return Category{}, err
	}

	if rowsAffected == 0 {
		return Category{}, sql.ErrNoRows
	}

	return q.GetCategoryByID(ctx, arg.ID)
}
