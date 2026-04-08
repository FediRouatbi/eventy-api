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
    latitude,
    longitude,
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
    ?,
    ?,
    ?
);

-- name: GetEventByID :one
SELECT id, organizer_id, category_id, title, slug, description, venue_name, venue_address, city, country, latitude, longitude, banner_url, poster_url, status, currency, is_featured, created_at, updated_at
FROM events
WHERE id = ?
LIMIT 1;

-- name: ListEvents :many
SELECT id, organizer_id, category_id, title, slug, description, venue_name, venue_address, city, country, latitude, longitude, banner_url, poster_url, status, currency, is_featured, created_at, updated_at
FROM events
ORDER BY created_at DESC, title ASC;

-- name: ListEventsByOrganizerID :many
SELECT id, organizer_id, category_id, title, slug, description, venue_name, venue_address, city, country, latitude, longitude, banner_url, poster_url, status, currency, is_featured, created_at, updated_at
FROM events
WHERE organizer_id = ?
ORDER BY created_at DESC, title ASC;

-- name: UpdateEvent :one
UPDATE events
SET category_id = ?,
    title = ?,
    slug = ?,
    description = ?,
    venue_name = ?,
    venue_address = ?,
    city = ?,
    country = ?,
    latitude = ?,
    longitude = ?,
    banner_url = ?,
    poster_url = ?,
    status = ?,
    currency = ?,
    is_featured = ?
WHERE id = ?;

-- name: DeleteEvent :execrows
DELETE FROM events
WHERE id = ?;

-- name: ListPublicEvents :many
WITH next_sessions AS (
    SELECT
        es.id AS session_id,
        es.event_id,
        es.starts_at,
        es.ends_at,
        es.sales_starts_at,
        es.sales_ends_at,
        ROW_NUMBER() OVER (
            PARTITION BY es.event_id
            ORDER BY es.starts_at ASC, es.created_at ASC
        ) AS row_num
    FROM event_sessions es
    WHERE es.status = 'scheduled'
      AND es.starts_at >= UTC_TIMESTAMP()
),
ticket_summaries AS (
    SELECT
        tt.event_session_id,
        MIN(tt.price) AS price_from,
        COALESCE(SUM(tt.quantity), 0) AS tickets_left
    FROM ticket_types tt
    GROUP BY tt.event_session_id
)
SELECT
    e.id,
    e.organizer_id,
    o.name AS organizer_name,
    o.slug AS organizer_slug,
    e.category_id,
    c.name AS category_name,
    c.slug AS category_slug,
    c.description AS category_description,
    c.image_url AS category_image_url,
    e.title,
    e.slug,
    e.description,
    e.venue_name,
    e.venue_address,
    e.city,
    e.country,
    e.latitude,
    e.longitude,
    e.banner_url,
    e.poster_url,
    e.status,
    e.currency,
    e.is_featured,
    e.created_at,
    e.updated_at,
    ns.session_id AS next_session_id,
    ns.starts_at AS next_session_starts_at,
    ns.ends_at AS next_session_ends_at,
    ns.sales_starts_at AS next_sales_starts_at,
    ns.sales_ends_at AS next_sales_ends_at,
    COALESCE(ts.price_from, 0) AS price_from,
    COALESCE(ts.tickets_left, 0) AS tickets_left
FROM events e
JOIN organizers o ON o.id = e.organizer_id
JOIN categories c ON c.id = e.category_id
JOIN next_sessions ns
  ON ns.event_id = e.id
 AND ns.row_num = 1
LEFT JOIN ticket_summaries ts
  ON ts.event_session_id = ns.session_id
WHERE e.status = 'published'
ORDER BY e.is_featured DESC, ns.starts_at ASC, e.title ASC;

-- name: GetPublicEventByID :one
WITH next_sessions AS (
    SELECT
        es.id AS session_id,
        es.event_id,
        es.starts_at,
        es.ends_at,
        es.sales_starts_at,
        es.sales_ends_at,
        ROW_NUMBER() OVER (
            PARTITION BY es.event_id
            ORDER BY es.starts_at ASC, es.created_at ASC
        ) AS row_num
    FROM event_sessions es
    WHERE es.status = 'scheduled'
),
ticket_summaries AS (
    SELECT
        tt.event_session_id,
        MIN(tt.price) AS price_from,
        COALESCE(SUM(tt.quantity), 0) AS tickets_left
    FROM ticket_types tt
    GROUP BY tt.event_session_id
)
SELECT
    e.id,
    e.organizer_id,
    o.name AS organizer_name,
    o.slug AS organizer_slug,
    e.category_id,
    c.name AS category_name,
    c.slug AS category_slug,
    c.description AS category_description,
    c.image_url AS category_image_url,
    e.title,
    e.slug,
    e.description,
    e.venue_name,
    e.venue_address,
    e.city,
    e.country,
    e.latitude,
    e.longitude,
    e.banner_url,
    e.poster_url,
    e.status,
    e.currency,
    e.is_featured,
    e.created_at,
    e.updated_at,
    ns.session_id AS next_session_id,
    ns.starts_at AS next_session_starts_at,
    ns.ends_at AS next_session_ends_at,
    ns.sales_starts_at AS next_sales_starts_at,
    ns.sales_ends_at AS next_sales_ends_at,
    COALESCE(ts.price_from, 0) AS price_from,
    COALESCE(ts.tickets_left, 0) AS tickets_left
FROM events e
JOIN organizers o ON o.id = e.organizer_id
JOIN categories c ON c.id = e.category_id
JOIN next_sessions ns
  ON ns.event_id = e.id
 AND ns.row_num = 1
LEFT JOIN ticket_summaries ts
  ON ts.event_session_id = ns.session_id
WHERE e.id = ?
  AND e.status = 'published'
LIMIT 1;

-- name: CreateEventSession :one
INSERT INTO event_sessions (
    id,
    event_id,
    starts_at,
    ends_at,
    sales_starts_at,
    sales_ends_at,
    status
) VALUES (
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?
);

-- name: GetEventSessionByID :one
SELECT id, event_id, starts_at, ends_at, sales_starts_at, sales_ends_at, status, created_at, updated_at
FROM event_sessions
WHERE id = ?
LIMIT 1;

-- name: ListEventSessionsByEventID :many
SELECT id, event_id, starts_at, ends_at, sales_starts_at, sales_ends_at, status, created_at, updated_at
FROM event_sessions
WHERE event_id = ?
ORDER BY starts_at ASC, created_at ASC;

-- name: ListPublicEventSessionsByEventID :many
SELECT id, event_id, starts_at, ends_at, sales_starts_at, sales_ends_at, status, created_at, updated_at
FROM event_sessions
WHERE event_id = ?
  AND status = 'scheduled'
ORDER BY starts_at ASC, created_at ASC;

-- name: UpdateEventSession :one
UPDATE event_sessions
SET starts_at = ?,
    ends_at = ?,
    sales_starts_at = ?,
    sales_ends_at = ?,
    status = ?
WHERE id = ?;

-- name: DeleteEventSession :execrows
DELETE FROM event_sessions
WHERE id = ?;

-- name: CreateTicketType :one
INSERT INTO ticket_types (
    id,
    event_session_id,
    name,
    description,
    price,
    quantity,
    max_per_order
) VALUES (
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?
);

-- name: GetTicketTypeByID :one
SELECT id, event_session_id, name, description, price, quantity, max_per_order, created_at, updated_at
FROM ticket_types
WHERE id = ?
LIMIT 1;

-- name: ListTicketTypesBySessionID :many
SELECT id, event_session_id, name, description, price, quantity, max_per_order, created_at, updated_at
FROM ticket_types
WHERE event_session_id = ?
ORDER BY created_at ASC, name ASC;

-- name: UpdateTicketType :one
UPDATE ticket_types
SET name = ?,
    description = ?,
    price = ?,
    quantity = ?,
    max_per_order = ?
WHERE id = ?;

-- name: DeleteTicketType :execrows
DELETE FROM ticket_types
WHERE id = ?;
