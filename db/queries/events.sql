-- name: CreateEvent :one
INSERT INTO events (
    id,
    organizer_id,
    category_id,
    title,
    slug,
    description,
    venue_name,
    venue_address,
    city,
    country,
    banner_url,
    poster_url,
    status,
    currency,
    is_featured
) VALUES (
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?
);

-- name: GetEventByID :one
SELECT id, organizer_id, category_id, title, slug, description, venue_name, venue_address, city, country, banner_url, poster_url, status, currency, is_featured, created_at, updated_at
FROM events
WHERE id = ?
LIMIT 1;

-- name: ListEvents :many
SELECT id, organizer_id, category_id, title, slug, description, venue_name, venue_address, city, country, banner_url, poster_url, status, currency, is_featured, created_at, updated_at
FROM events
ORDER BY created_at DESC, title ASC;

-- name: ListEventsByOrganizerID :many
SELECT id, organizer_id, category_id, title, slug, description, venue_name, venue_address, city, country, banner_url, poster_url, status, currency, is_featured, created_at, updated_at
FROM events
WHERE organizer_id = ?
ORDER BY created_at DESC, title ASC;
