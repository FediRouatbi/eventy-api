package sqlc

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

const createEventQuery = `
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
)
`

const getEventByIDQuery = `
SELECT id, organizer_id, category_id, title, slug, description, venue_name, venue_address, city, country, latitude, longitude, banner_url, poster_url, status, currency, is_featured, created_at, updated_at
FROM events
WHERE id = ?
LIMIT 1
`

const listEventsQuery = `
SELECT id, organizer_id, category_id, title, slug, description, venue_name, venue_address, city, country, latitude, longitude, banner_url, poster_url, status, currency, is_featured, created_at, updated_at
FROM events
ORDER BY created_at DESC, title ASC
`

const listEventsByOrganizerIDQuery = `
SELECT id, organizer_id, category_id, title, slug, description, venue_name, venue_address, city, country, latitude, longitude, banner_url, poster_url, status, currency, is_featured, created_at, updated_at
FROM events
WHERE organizer_id = ?
ORDER BY created_at DESC, title ASC
`

const updateEventQuery = `
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
WHERE id = ?
`

const deleteEventQuery = `
DELETE FROM events
WHERE id = ?
`

const listPublicEventsQuery = `
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
ORDER BY e.is_featured DESC, ns.starts_at ASC, e.title ASC
`

const getPublicEventByIDQuery = `
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
LIMIT 1
`

const createEventSessionQuery = `
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
)
`

const getEventSessionByIDQuery = `
SELECT id, event_id, starts_at, ends_at, sales_starts_at, sales_ends_at, status, created_at, updated_at
FROM event_sessions
WHERE id = ?
LIMIT 1
`

const listEventSessionsByEventIDQuery = `
SELECT id, event_id, starts_at, ends_at, sales_starts_at, sales_ends_at, status, created_at, updated_at
FROM event_sessions
WHERE event_id = ?
ORDER BY starts_at ASC, created_at ASC
`

const listPublicEventSessionsByEventIDQuery = `
SELECT id, event_id, starts_at, ends_at, sales_starts_at, sales_ends_at, status, created_at, updated_at
FROM event_sessions
WHERE event_id = ?
  AND status = 'scheduled'
ORDER BY starts_at ASC, created_at ASC
`

const updateEventSessionQuery = `
UPDATE event_sessions
SET starts_at = ?,
    ends_at = ?,
    sales_starts_at = ?,
    sales_ends_at = ?,
    status = ?
WHERE id = ?
`

const deleteEventSessionQuery = `
DELETE FROM event_sessions
WHERE id = ?
`

const createTicketTypeQuery = `
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
)
`

const getTicketTypeByIDQuery = `
SELECT id, event_session_id, name, description, price, quantity, max_per_order, created_at, updated_at
FROM ticket_types
WHERE id = ?
LIMIT 1
`

const listTicketTypesBySessionIDQuery = `
SELECT id, event_session_id, name, description, price, quantity, max_per_order, created_at, updated_at
FROM ticket_types
WHERE event_session_id = ?
ORDER BY created_at ASC, name ASC
`

const updateTicketTypeQuery = `
UPDATE ticket_types
SET name = ?,
    description = ?,
    price = ?,
    quantity = ?,
    max_per_order = ?
WHERE id = ?
`

const deleteTicketTypeQuery = `
DELETE FROM ticket_types
WHERE id = ?
`

type ListPublicEventsRow struct {
	ID                  string
	OrganizerID         string
	OrganizerName       string
	OrganizerSlug       string
	CategoryID          string
	CategoryName        string
	CategorySlug        string
	CategoryDescription sql.NullString
	CategoryImageURL    sql.NullString
	Title               string
	Slug                string
	Description         string
	VenueName           string
	VenueAddress        string
	City                string
	Country             string
	Latitude            sql.NullFloat64
	Longitude           sql.NullFloat64
	BannerURL           sql.NullString
	PosterURL           sql.NullString
	Status              string
	Currency            string
	IsFeatured          bool
	CreatedAt           time.Time
	UpdatedAt           time.Time
	NextSessionID       string
	NextSessionStartsAt time.Time
	NextSessionEndsAt   time.Time
	NextSalesStartsAt   sql.NullTime
	NextSalesEndsAt     sql.NullTime
	PriceFrom           float64
	TicketsLeft         int64
}

type GetPublicEventByIDRow struct {
	ID                  string
	OrganizerID         string
	OrganizerName       string
	OrganizerSlug       string
	CategoryID          string
	CategoryName        string
	CategorySlug        string
	CategoryDescription sql.NullString
	CategoryImageURL    sql.NullString
	Title               string
	Slug                string
	Description         string
	VenueName           string
	VenueAddress        string
	City                string
	Country             string
	Latitude            sql.NullFloat64
	Longitude           sql.NullFloat64
	BannerURL           sql.NullString
	PosterURL           sql.NullString
	Status              string
	Currency            string
	IsFeatured          bool
	CreatedAt           time.Time
	UpdatedAt           time.Time
	NextSessionID       string
	NextSessionStartsAt time.Time
	NextSessionEndsAt   time.Time
	NextSalesStartsAt   sql.NullTime
	NextSalesEndsAt     sql.NullTime
	PriceFrom           float64
	TicketsLeft         int64
}

func (q *Queries) CreateEvent(ctx context.Context, arg CreateEventParams) (Event, error) {
	_, err := q.db.ExecContext(
		ctx,
		createEventQuery,
		arg.ID,
		arg.OrganizerID,
		arg.CategoryID,
		arg.Title,
		arg.Slug,
		arg.Description,
		arg.VenueName,
		arg.VenueAddress,
		arg.City,
		arg.Country,
		arg.Latitude,
		arg.Longitude,
		arg.BannerURL,
		arg.PosterURL,
		arg.Status,
		arg.Currency,
		arg.IsFeatured,
	)
	if err != nil {
		return Event{}, err
	}

	return q.GetEventByID(ctx, arg.ID)
}

func (q *Queries) UpdateEvent(ctx context.Context, arg UpdateEventParams) (Event, error) {
	result, err := q.db.ExecContext(
		ctx,
		updateEventQuery,
		arg.CategoryID,
		arg.Title,
		arg.Slug,
		arg.Description,
		arg.VenueName,
		arg.VenueAddress,
		arg.City,
		arg.Country,
		arg.Latitude,
		arg.Longitude,
		arg.BannerURL,
		arg.PosterURL,
		arg.Status,
		arg.Currency,
		arg.IsFeatured,
		arg.ID,
	)
	if err != nil {
		return Event{}, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return Event{}, err
	}
	if rowsAffected == 0 {
		return q.GetEventByID(ctx, arg.ID)
	}

	return q.GetEventByID(ctx, arg.ID)
}

func (q *Queries) DeleteEvent(ctx context.Context, id string) (int64, error) {
	result, err := q.db.ExecContext(ctx, deleteEventQuery, id)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

func (q *Queries) GetEventByID(ctx context.Context, id string) (Event, error) {
	row := q.db.QueryRowContext(ctx, getEventByIDQuery, id)

	var event Event
	err := row.Scan(
		&event.ID,
		&event.OrganizerID,
		&event.CategoryID,
		&event.Title,
		&event.Slug,
		&event.Description,
		&event.VenueName,
		&event.VenueAddress,
		&event.City,
		&event.Country,
		&event.Latitude,
		&event.Longitude,
		&event.BannerURL,
		&event.PosterURL,
		&event.Status,
		&event.Currency,
		&event.IsFeatured,
		&event.CreatedAt,
		&event.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Event{}, sql.ErrNoRows
	}

	return event, err
}

func (q *Queries) ListEvents(ctx context.Context) ([]Event, error) {
	rows, err := q.db.QueryContext(ctx, listEventsQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Event
	for rows.Next() {
		var event Event
		if err := rows.Scan(
			&event.ID,
			&event.OrganizerID,
			&event.CategoryID,
			&event.Title,
			&event.Slug,
			&event.Description,
			&event.VenueName,
			&event.VenueAddress,
			&event.City,
			&event.Country,
			&event.Latitude,
			&event.Longitude,
			&event.BannerURL,
			&event.PosterURL,
			&event.Status,
			&event.Currency,
			&event.IsFeatured,
			&event.CreatedAt,
			&event.UpdatedAt,
		); err != nil {
			return nil, err
		}

		items = append(items, event)
	}

	return items, rows.Err()
}

func (q *Queries) ListEventsByOrganizerID(ctx context.Context, organizerID string) ([]Event, error) {
	rows, err := q.db.QueryContext(ctx, listEventsByOrganizerIDQuery, organizerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Event
	for rows.Next() {
		var event Event
		if err := rows.Scan(
			&event.ID,
			&event.OrganizerID,
			&event.CategoryID,
			&event.Title,
			&event.Slug,
			&event.Description,
			&event.VenueName,
			&event.VenueAddress,
			&event.City,
			&event.Country,
			&event.Latitude,
			&event.Longitude,
			&event.BannerURL,
			&event.PosterURL,
			&event.Status,
			&event.Currency,
			&event.IsFeatured,
			&event.CreatedAt,
			&event.UpdatedAt,
		); err != nil {
			return nil, err
		}

		items = append(items, event)
	}

	return items, rows.Err()
}

func (q *Queries) ListPublicEvents(ctx context.Context) ([]ListPublicEventsRow, error) {
	rows, err := q.db.QueryContext(ctx, listPublicEventsQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []ListPublicEventsRow
	for rows.Next() {
		var item ListPublicEventsRow
		if err := rows.Scan(
			&item.ID,
			&item.OrganizerID,
			&item.OrganizerName,
			&item.OrganizerSlug,
			&item.CategoryID,
			&item.CategoryName,
			&item.CategorySlug,
			&item.CategoryDescription,
			&item.CategoryImageURL,
			&item.Title,
			&item.Slug,
			&item.Description,
			&item.VenueName,
			&item.VenueAddress,
			&item.City,
			&item.Country,
			&item.Latitude,
			&item.Longitude,
			&item.BannerURL,
			&item.PosterURL,
			&item.Status,
			&item.Currency,
			&item.IsFeatured,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.NextSessionID,
			&item.NextSessionStartsAt,
			&item.NextSessionEndsAt,
			&item.NextSalesStartsAt,
			&item.NextSalesEndsAt,
			&item.PriceFrom,
			&item.TicketsLeft,
		); err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	return items, rows.Err()
}

func (q *Queries) GetPublicEventByID(ctx context.Context, id string) (GetPublicEventByIDRow, error) {
	row := q.db.QueryRowContext(ctx, getPublicEventByIDQuery, id)

	var item GetPublicEventByIDRow
	err := row.Scan(
		&item.ID,
		&item.OrganizerID,
		&item.OrganizerName,
		&item.OrganizerSlug,
		&item.CategoryID,
		&item.CategoryName,
		&item.CategorySlug,
		&item.CategoryDescription,
		&item.CategoryImageURL,
		&item.Title,
		&item.Slug,
		&item.Description,
		&item.VenueName,
		&item.VenueAddress,
		&item.City,
		&item.Country,
		&item.Latitude,
		&item.Longitude,
		&item.BannerURL,
		&item.PosterURL,
		&item.Status,
		&item.Currency,
		&item.IsFeatured,
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.NextSessionID,
		&item.NextSessionStartsAt,
		&item.NextSessionEndsAt,
		&item.NextSalesStartsAt,
		&item.NextSalesEndsAt,
		&item.PriceFrom,
		&item.TicketsLeft,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return GetPublicEventByIDRow{}, sql.ErrNoRows
	}

	return item, err
}

func (q *Queries) CreateEventSession(ctx context.Context, arg CreateEventSessionParams) (EventSession, error) {
	_, err := q.db.ExecContext(
		ctx,
		createEventSessionQuery,
		arg.ID,
		arg.EventID,
		arg.StartsAt,
		arg.EndsAt,
		arg.SalesStartsAt,
		arg.SalesEndsAt,
		arg.Status,
	)
	if err != nil {
		return EventSession{}, err
	}

	return q.GetEventSessionByID(ctx, arg.ID)
}

func (q *Queries) GetEventSessionByID(ctx context.Context, id string) (EventSession, error) {
	row := q.db.QueryRowContext(ctx, getEventSessionByIDQuery, id)

	var item EventSession
	err := row.Scan(
		&item.ID,
		&item.EventID,
		&item.StartsAt,
		&item.EndsAt,
		&item.SalesStartsAt,
		&item.SalesEndsAt,
		&item.Status,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return EventSession{}, sql.ErrNoRows
	}

	return item, err
}

func (q *Queries) ListEventSessionsByEventID(ctx context.Context, eventID string) ([]EventSession, error) {
	rows, err := q.db.QueryContext(ctx, listEventSessionsByEventIDQuery, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []EventSession
	for rows.Next() {
		var item EventSession
		if err := rows.Scan(
			&item.ID,
			&item.EventID,
			&item.StartsAt,
			&item.EndsAt,
			&item.SalesStartsAt,
			&item.SalesEndsAt,
			&item.Status,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	return items, rows.Err()
}

func (q *Queries) ListPublicEventSessionsByEventID(ctx context.Context, eventID string) ([]EventSession, error) {
	rows, err := q.db.QueryContext(ctx, listPublicEventSessionsByEventIDQuery, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []EventSession
	for rows.Next() {
		var item EventSession
		if err := rows.Scan(
			&item.ID,
			&item.EventID,
			&item.StartsAt,
			&item.EndsAt,
			&item.SalesStartsAt,
			&item.SalesEndsAt,
			&item.Status,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	return items, rows.Err()
}

func (q *Queries) UpdateEventSession(ctx context.Context, arg UpdateEventSessionParams) (EventSession, error) {
	result, err := q.db.ExecContext(
		ctx,
		updateEventSessionQuery,
		arg.StartsAt,
		arg.EndsAt,
		arg.SalesStartsAt,
		arg.SalesEndsAt,
		arg.Status,
		arg.ID,
	)
	if err != nil {
		return EventSession{}, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return EventSession{}, err
	}
	if rowsAffected == 0 {
		return q.GetEventSessionByID(ctx, arg.ID)
	}

	return q.GetEventSessionByID(ctx, arg.ID)
}

func (q *Queries) DeleteEventSession(ctx context.Context, id string) (int64, error) {
	result, err := q.db.ExecContext(ctx, deleteEventSessionQuery, id)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

func (q *Queries) CreateTicketType(ctx context.Context, arg CreateTicketTypeParams) (TicketType, error) {
	_, err := q.db.ExecContext(
		ctx,
		createTicketTypeQuery,
		arg.ID,
		arg.EventSessionID,
		arg.Name,
		arg.Description,
		arg.Price,
		arg.Quantity,
		arg.MaxPerOrder,
	)
	if err != nil {
		return TicketType{}, err
	}

	return q.GetTicketTypeByID(ctx, arg.ID)
}

func (q *Queries) GetTicketTypeByID(ctx context.Context, id string) (TicketType, error) {
	row := q.db.QueryRowContext(ctx, getTicketTypeByIDQuery, id)

	var item TicketType
	err := row.Scan(
		&item.ID,
		&item.EventSessionID,
		&item.Name,
		&item.Description,
		&item.Price,
		&item.Quantity,
		&item.MaxPerOrder,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return TicketType{}, sql.ErrNoRows
	}

	return item, err
}

func (q *Queries) ListTicketTypesBySessionID(ctx context.Context, sessionID string) ([]TicketType, error) {
	rows, err := q.db.QueryContext(ctx, listTicketTypesBySessionIDQuery, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []TicketType
	for rows.Next() {
		var item TicketType
		if err := rows.Scan(
			&item.ID,
			&item.EventSessionID,
			&item.Name,
			&item.Description,
			&item.Price,
			&item.Quantity,
			&item.MaxPerOrder,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	return items, rows.Err()
}

func (q *Queries) UpdateTicketType(ctx context.Context, arg UpdateTicketTypeParams) (TicketType, error) {
	result, err := q.db.ExecContext(
		ctx,
		updateTicketTypeQuery,
		arg.Name,
		arg.Description,
		arg.Price,
		arg.Quantity,
		arg.MaxPerOrder,
		arg.ID,
	)
	if err != nil {
		return TicketType{}, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return TicketType{}, err
	}
	if rowsAffected == 0 {
		return q.GetTicketTypeByID(ctx, arg.ID)
	}

	return q.GetTicketTypeByID(ctx, arg.ID)
}

func (q *Queries) DeleteTicketType(ctx context.Context, id string) (int64, error) {
	result, err := q.db.ExecContext(ctx, deleteTicketTypeQuery, id)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}
