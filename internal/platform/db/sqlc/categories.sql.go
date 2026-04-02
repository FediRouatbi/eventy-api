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
    image_url
) VALUES (
    ?,
    ?,
    ?,
    ?
)
`

const getCategoryByIDQuery = `
SELECT id, name, slug, image_url, created_at, updated_at
FROM categories
WHERE id = ?
LIMIT 1
`

const listCategoriesQuery = `
SELECT id, name, slug, image_url, created_at, updated_at
FROM categories
ORDER BY name ASC
`

func (q *Queries) CreateCategory(ctx context.Context, arg CreateCategoryParams) (Category, error) {
	_, err := q.db.ExecContext(ctx, createCategoryQuery, arg.ID, arg.Name, arg.Slug, arg.ImageURL)
	if err != nil {
		return Category{}, err
	}

	return q.GetCategoryByID(ctx, arg.ID)
}

func (q *Queries) GetCategoryByID(ctx context.Context, id string) (Category, error) {
	row := q.db.QueryRowContext(ctx, getCategoryByIDQuery, id)

	var category Category
	err := row.Scan(
		&category.ID,
		&category.Name,
		&category.Slug,
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
