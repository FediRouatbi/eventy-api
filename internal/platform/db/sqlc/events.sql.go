package sqlc

import (
	"context"
	"database/sql"
	"errors"
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
)
`

const getEventByIDQuery = `
SELECT id, organizer_id, category_id, title, slug, description, venue_name, venue_address, city, country, banner_url, poster_url, status, currency, is_featured, created_at, updated_at
FROM events
WHERE id = ?
LIMIT 1
`

const listEventsQuery = `
SELECT id, organizer_id, category_id, title, slug, description, venue_name, venue_address, city, country, banner_url, poster_url, status, currency, is_featured, created_at, updated_at
FROM events
ORDER BY created_at DESC, title ASC
`

const listEventsByOrganizerIDQuery = `
SELECT id, organizer_id, category_id, title, slug, description, venue_name, venue_address, city, country, banner_url, poster_url, status, currency, is_featured, created_at, updated_at
FROM events
WHERE organizer_id = ?
ORDER BY created_at DESC, title ASC
`

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
