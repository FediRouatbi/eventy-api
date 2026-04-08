-- name: CreateCategory :one
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
);

-- name: GetCategoryByID :one
SELECT id, name, slug, description, image_url, created_at, updated_at
FROM categories
WHERE id = ?
LIMIT 1;

-- name: ListCategories :many
SELECT id, name, slug, description, image_url, created_at, updated_at
FROM categories
ORDER BY name ASC;
