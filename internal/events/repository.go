package events

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"eventy-api/internal/platform/db/sqlc"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v81"
	checkoutsession "github.com/stripe/stripe-go/v81/checkout/session"
	"github.com/stripe/stripe-go/v81/webhook"
)

type Repository struct {
	db      *sql.DB
	queries *sqlc.Queries
}

func NewRepository(db *sql.DB, queries *sqlc.Queries) *Repository {
	return &Repository{db: db, queries: queries}
}

func (r *Repository) Create(ctx context.Context, organizerID uuid.UUID, categoryID uuid.UUID, input CreateEventInput) (Event, error) {
	eventID := uuid.New()

	dbEvent, err := r.queries.CreateEvent(ctx, sqlc.CreateEventParams{
		ID:           eventID.String(),
		OrganizerID:  organizerID.String(),
		CategoryID:   categoryID.String(),
		Title:        strings.TrimSpace(input.Title),
		Slug:         strings.TrimSpace(input.Slug),
		Description:  strings.TrimSpace(input.Description),
		VenueName:    strings.TrimSpace(input.VenueName),
		VenueAddress: strings.TrimSpace(input.VenueAddress),
		City:         strings.TrimSpace(input.City),
		Country:      strings.TrimSpace(input.Country),
		Latitude:     nullableFloat64(input.Latitude),
		Longitude:    nullableFloat64(input.Longitude),
		BannerURL:    nullableString(strings.TrimSpace(input.BannerURL)),
		PosterURL:    nullableString(strings.TrimSpace(input.PosterURL)),
		Status:       strings.TrimSpace(input.Status),
		Currency:     strings.TrimSpace(input.Currency),
		IsFeatured:   input.IsFeatured,
	})
	if err != nil {
		switch {
		case isUniqueViolation(err):
			return Event{}, ErrEventSlugAlreadyExists
		case isForeignKeyViolation(err) && hasConstraint(err, "fk_events_category"):
			return Event{}, ErrCategoryNotFound
		case isForeignKeyViolation(err) && hasConstraint(err, "fk_events_organizer"):
			return Event{}, ErrOrganizerNotFound
		default:
			return Event{}, err
		}
	}

	return mapEvent(dbEvent)
}

func (r *Repository) Update(ctx context.Context, eventID uuid.UUID, categoryID uuid.UUID, input UpdateEventInput) (Event, error) {
	dbEvent, err := r.queries.UpdateEvent(ctx, sqlc.UpdateEventParams{
		CategoryID:   categoryID.String(),
		Title:        strings.TrimSpace(input.Title),
		Slug:         strings.TrimSpace(input.Slug),
		Description:  strings.TrimSpace(input.Description),
		VenueName:    strings.TrimSpace(input.VenueName),
		VenueAddress: strings.TrimSpace(input.VenueAddress),
		City:         strings.TrimSpace(input.City),
		Country:      strings.TrimSpace(input.Country),
		Latitude:     nullableFloat64(input.Latitude),
		Longitude:    nullableFloat64(input.Longitude),
		BannerURL:    nullableString(strings.TrimSpace(input.BannerURL)),
		PosterURL:    nullableString(strings.TrimSpace(input.PosterURL)),
		Status:       strings.TrimSpace(input.Status),
		Currency:     strings.TrimSpace(input.Currency),
		IsFeatured:   input.IsFeatured,
		ID:           eventID.String(),
	})
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return Event{}, ErrEventNotFound
		case isUniqueViolation(err):
			return Event{}, ErrEventSlugAlreadyExists
		case isForeignKeyViolation(err) && hasConstraint(err, "fk_events_category"):
			return Event{}, ErrCategoryNotFound
		default:
			return Event{}, err
		}
	}

	return mapEvent(dbEvent)
}

func (r *Repository) Delete(ctx context.Context, eventID uuid.UUID) error {
	rowsAffected, err := r.queries.DeleteEvent(ctx, eventID.String())
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrEventNotFound
	}

	return nil
}

func (r *Repository) List(ctx context.Context) ([]EventListItem, error) {
	return r.listItems(ctx, nil)
}

func (r *Repository) ListPublic(ctx context.Context) ([]PublicEvent, error) {
	rows, err := r.db.QueryContext(ctx, listPublicEventsQuery(true))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]PublicEvent, 0)
	for rows.Next() {
		item, err := scanPublicEvent(rows)
		if err != nil {
			return nil, err
		}

		sessions, err := r.ListPublicSessionsByEventID(ctx, item.ID)
		if err != nil {
			return nil, err
		}

		detailSessions := make([]EventSessionDetail, 0, len(sessions))
		for _, session := range sessions {
			ticketTypes, err := r.ListPublicTicketTypesBySessionID(ctx, session.ID)
			if err != nil {
				return nil, err
			}

			if len(ticketTypes) == 0 {
				continue
			}

			detailSessions = append(detailSessions, EventSessionDetail{
				EventSession: session,
				TicketTypes:  ticketTypes,
			})
		}

		item.Sessions = detailSessions

		if len(item.Sessions) == 0 {
			continue
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func (r *Repository) ListByOrganizerID(ctx context.Context, organizerID uuid.UUID) ([]EventListItem, error) {
	return r.listItems(ctx, &organizerID)
}

func (r *Repository) GetByID(ctx context.Context, eventID uuid.UUID) (Event, error) {
	dbEvent, err := r.queries.GetEventByID(ctx, eventID.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Event{}, ErrEventNotFound
		}

		return Event{}, err
	}

	return mapEvent(dbEvent)
}

func (r *Repository) listItems(ctx context.Context, organizerID *uuid.UUID) ([]EventListItem, error) {
	baseQuery := `
SELECT e.id, e.organizer_id, e.category_id, e.title, e.slug, e.description,
       e.venue_name, e.venue_address, e.city, e.country, e.latitude, e.longitude,
       e.banner_url, e.poster_url, e.status, e.currency, e.is_featured,
       e.created_at, e.updated_at,
       o.name AS organizer_name, o.slug AS organizer_slug,
       c.name AS category_name, c.slug AS category_slug,
       COUNT(DISTINCT es.id) AS session_count,
       COUNT(DISTINCT tt.id) AS ticket_type_count,
       MIN(CASE
           WHEN es.status = 'scheduled' AND es.starts_at >= UTC_TIMESTAMP() THEN es.starts_at
           ELSE NULL
       END) AS next_session_starts_at,
       MAX(CASE
           WHEN es.id IS NOT NULL AND tt.id IS NULL THEN 1
           ELSE 0
       END) AS has_sessions_without_tickets
FROM events e
JOIN organizers o ON o.id = e.organizer_id
JOIN categories c ON c.id = e.category_id
LEFT JOIN event_sessions es ON es.event_id = e.id
LEFT JOIN ticket_types tt ON tt.event_session_id = es.id
`

	args := []any{}
	if organizerID != nil {
		baseQuery += "WHERE e.organizer_id = ?\n"
		args = append(args, organizerID.String())
	}

	baseQuery += `
GROUP BY e.id, e.organizer_id, e.category_id, e.title, e.slug, e.description,
         e.venue_name, e.venue_address, e.city, e.country, e.latitude, e.longitude,
         e.banner_url, e.poster_url, e.status, e.currency, e.is_featured,
         e.created_at, e.updated_at, o.name, o.slug, c.name, c.slug
ORDER BY e.created_at DESC, e.title ASC
`

	rows, err := r.db.QueryContext(ctx, baseQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]EventListItem, 0)
	for rows.Next() {
		var item EventListItem
		var eventIDText string
		var organizerIDText string
		var categoryIDText string
		var latitude sql.NullFloat64
		var longitude sql.NullFloat64
		var bannerURL sql.NullString
		var posterURL sql.NullString
		var nextSessionStartsAt sql.NullTime
		var hasSessionsWithoutTickets int64

		if err := rows.Scan(
			&eventIDText,
			&organizerIDText,
			&categoryIDText,
			&item.Title,
			&item.Slug,
			&item.Description,
			&item.VenueName,
			&item.VenueAddress,
			&item.City,
			&item.Country,
			&latitude,
			&longitude,
			&bannerURL,
			&posterURL,
			&item.Status,
			&item.Currency,
			&item.IsFeatured,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.OrganizerName,
			&item.OrganizerSlug,
			&item.CategoryName,
			&item.CategorySlug,
			&item.SessionCount,
			&item.TicketTypeCount,
			&nextSessionStartsAt,
			&hasSessionsWithoutTickets,
		); err != nil {
			return nil, err
		}

		item.ID, err = uuid.Parse(eventIDText)
		if err != nil {
			return nil, err
		}

		item.OrganizerID, err = uuid.Parse(organizerIDText)
		if err != nil {
			return nil, err
		}

		item.CategoryID, err = uuid.Parse(categoryIDText)
		if err != nil {
			return nil, err
		}

		if latitude.Valid {
			latitudeValue := latitude.Float64
			item.Latitude = &latitudeValue
		}
		if longitude.Valid {
			longitudeValue := longitude.Float64
			item.Longitude = &longitudeValue
		}
		if bannerURL.Valid {
			item.BannerURL = bannerURL.String
		}
		if posterURL.Valid {
			item.PosterURL = posterURL.String
		}
		if nextSessionStartsAt.Valid {
			nextSessionStartsAtValue := nextSessionStartsAt.Time
			item.NextSessionStartsAt = &nextSessionStartsAtValue
		}

		item.HasSessionsWithoutTickets = hasSessionsWithoutTickets > 0

		items = append(items, item)
	}

	return items, rows.Err()
}

func (r *Repository) GetDetailByID(ctx context.Context, eventID uuid.UUID) (EventDetail, error) {
	event, err := r.GetByID(ctx, eventID)
	if err != nil {
		return EventDetail{}, err
	}

	sessions, err := r.ListSessionsByEventID(ctx, eventID)
	if err != nil {
		return EventDetail{}, err
	}

	detailSessions := make([]EventSessionDetail, 0, len(sessions))
	for _, session := range sessions {
		ticketTypes, err := r.ListTicketTypesBySessionID(ctx, session.ID)
		if err != nil {
			return EventDetail{}, err
		}

		detailSessions = append(detailSessions, EventSessionDetail{
			EventSession: session,
			TicketTypes:  ticketTypes,
		})
	}

	return EventDetail{
		Event:    event,
		Sessions: detailSessions,
	}, nil
}

func (r *Repository) GetPublicDetailByID(ctx context.Context, eventID uuid.UUID) (PublicEventDetail, error) {
	row := r.db.QueryRowContext(ctx, listPublicEventsQuery(false), eventID.String())
	event, err := scanPublicEvent(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PublicEventDetail{}, ErrEventNotFound
		}

		return PublicEventDetail{}, err
	}

	sessions, err := r.ListPublicSessionsByEventID(ctx, eventID)
	if err != nil {
		return PublicEventDetail{}, err
	}

	detailSessions := make([]EventSessionDetail, 0, len(sessions))
	for _, session := range sessions {
		ticketTypes, err := r.ListPublicTicketTypesBySessionID(ctx, session.ID)
		if err != nil {
			return PublicEventDetail{}, err
		}

		if len(ticketTypes) == 0 {
			continue
		}

		detailSessions = append(detailSessions, EventSessionDetail{
			EventSession: session,
			TicketTypes:  ticketTypes,
		})
	}

	if len(detailSessions) == 0 {
		return PublicEventDetail{}, ErrEventNotFound
	}

	event.Sessions = detailSessions

	return PublicEventDetail{
		PublicEvent: event,
	}, nil
}

func (r *Repository) ListPublicTicketTypesBySessionID(ctx context.Context, sessionID uuid.UUID) ([]TicketType, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT
    tt.id,
    tt.event_session_id,
    tt.name,
    tt.description,
    tt.price,
    GREATEST(tt.quantity - COALESCE(rs.reserved_quantity, 0), 0) AS quantity,
    tt.max_per_order,
    tt.created_at,
    tt.updated_at
FROM ticket_types tt
JOIN event_sessions es ON es.id = tt.event_session_id
JOIN events e ON e.id = es.event_id
LEFT JOIN (
    SELECT
        tri.ticket_type_id,
        SUM(tri.quantity) AS reserved_quantity
    FROM ticket_reservation_items tri
    JOIN ticket_reservations tr ON tr.id = tri.reservation_id
    WHERE tr.expires_at > UTC_TIMESTAMP()
    GROUP BY tri.ticket_type_id
) rs ON rs.ticket_type_id = tt.id
WHERE tt.event_session_id = ?
  AND es.status = 'scheduled'
  AND e.status = 'published'
ORDER BY tt.created_at ASC, tt.name ASC
`, sessionID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]TicketType, 0)
	for rows.Next() {
		var dbTicketType sqlc.TicketType
		if err := rows.Scan(
			&dbTicketType.ID,
			&dbTicketType.EventSessionID,
			&dbTicketType.Name,
			&dbTicketType.Description,
			&dbTicketType.Price,
			&dbTicketType.Quantity,
			&dbTicketType.MaxPerOrder,
			&dbTicketType.CreatedAt,
			&dbTicketType.UpdatedAt,
		); err != nil {
			return nil, err
		}

		item, err := mapTicketType(dbTicketType)
		if err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	return items, rows.Err()
}

func (r *Repository) CreateSession(ctx context.Context, eventID uuid.UUID, input CreateEventSessionInput) (EventSession, error) {
	sessionID := uuid.New()

	dbSession, err := r.queries.CreateEventSession(ctx, sqlc.CreateEventSessionParams{
		ID:            sessionID.String(),
		EventID:       eventID.String(),
		StartsAt:      input.StartsAt,
		EndsAt:        input.EndsAt,
		SalesStartsAt: nullableTime(input.SalesStartsAt),
		SalesEndsAt:   nullableTime(input.SalesEndsAt),
		Status:        strings.TrimSpace(input.Status),
	})
	if err != nil {
		switch {
		case isForeignKeyViolation(err) && hasConstraint(err, "fk_event_sessions_event"):
			return EventSession{}, ErrEventNotFound
		default:
			return EventSession{}, err
		}
	}

	return mapEventSession(dbSession)
}

func (r *Repository) ListSessionsByEventID(ctx context.Context, eventID uuid.UUID) ([]EventSession, error) {
	dbSessions, err := r.queries.ListEventSessionsByEventID(ctx, eventID.String())
	if err != nil {
		return nil, err
	}

	return mapEventSessions(dbSessions)
}

func (r *Repository) ListPublicSessionsByEventID(ctx context.Context, eventID uuid.UUID) ([]EventSession, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, event_id, starts_at, ends_at, sales_starts_at, sales_ends_at, status, created_at, updated_at
FROM event_sessions es
WHERE es.event_id = ?
  AND es.status = 'scheduled'
  AND es.ends_at >= UTC_TIMESTAMP()
  AND (es.sales_starts_at IS NULL OR es.sales_starts_at <= UTC_TIMESTAMP())
  AND (es.sales_ends_at IS NULL OR es.sales_ends_at >= UTC_TIMESTAMP())
  AND EXISTS (
      SELECT 1
      FROM ticket_types tt
      WHERE tt.event_session_id = es.id
  )
ORDER BY es.starts_at ASC, es.created_at ASC
`, eventID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	dbSessions := make([]sqlc.EventSession, 0)
	for rows.Next() {
		var session sqlc.EventSession
		if err := rows.Scan(
			&session.ID,
			&session.EventID,
			&session.StartsAt,
			&session.EndsAt,
			&session.SalesStartsAt,
			&session.SalesEndsAt,
			&session.Status,
			&session.CreatedAt,
			&session.UpdatedAt,
		); err != nil {
			return nil, err
		}
		dbSessions = append(dbSessions, session)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return mapEventSessions(dbSessions)
}

func (r *Repository) GetSessionByID(ctx context.Context, sessionID uuid.UUID) (EventSession, error) {
	dbSession, err := r.queries.GetEventSessionByID(ctx, sessionID.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return EventSession{}, ErrEventSessionNotFound
		}

		return EventSession{}, err
	}

	return mapEventSession(dbSession)
}

func (r *Repository) UpdateSession(ctx context.Context, sessionID uuid.UUID, input UpdateEventSessionInput) (EventSession, error) {
	dbSession, err := r.queries.UpdateEventSession(ctx, sqlc.UpdateEventSessionParams{
		StartsAt:      input.StartsAt,
		EndsAt:        input.EndsAt,
		SalesStartsAt: nullableTime(input.SalesStartsAt),
		SalesEndsAt:   nullableTime(input.SalesEndsAt),
		Status:        strings.TrimSpace(input.Status),
		ID:            sessionID.String(),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return EventSession{}, ErrEventSessionNotFound
		}

		return EventSession{}, err
	}

	return mapEventSession(dbSession)
}

func (r *Repository) DeleteSession(ctx context.Context, sessionID uuid.UUID) error {
	rowsAffected, err := r.queries.DeleteEventSession(ctx, sessionID.String())
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrEventSessionNotFound
	}

	return nil
}

func (r *Repository) CreateTicketType(ctx context.Context, sessionID uuid.UUID, input CreateTicketTypeInput) (TicketType, error) {
	ticketTypeID := uuid.New()

	dbTicketType, err := r.queries.CreateTicketType(ctx, sqlc.CreateTicketTypeParams{
		ID:             ticketTypeID.String(),
		EventSessionID: sessionID.String(),
		Name:           strings.TrimSpace(input.Name),
		Description:    nullableString(strings.TrimSpace(input.Description)),
		Price:          input.Price,
		Quantity:       input.Quantity,
		MaxPerOrder:    input.MaxPerOrder,
	})
	if err != nil {
		switch {
		case isForeignKeyViolation(err) && hasConstraint(err, "fk_ticket_types_event_session"):
			return TicketType{}, ErrEventSessionNotFound
		default:
			return TicketType{}, err
		}
	}

	return mapTicketType(dbTicketType)
}

func (r *Repository) ListTicketTypesBySessionID(ctx context.Context, sessionID uuid.UUID) ([]TicketType, error) {
	dbTicketTypes, err := r.queries.ListTicketTypesBySessionID(ctx, sessionID.String())
	if err != nil {
		return nil, err
	}

	return mapTicketTypes(dbTicketTypes)
}

func (r *Repository) GetTicketTypeByID(ctx context.Context, ticketTypeID uuid.UUID) (TicketType, error) {
	dbTicketType, err := r.queries.GetTicketTypeByID(ctx, ticketTypeID.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return TicketType{}, ErrTicketTypeNotFound
		}

		return TicketType{}, err
	}

	return mapTicketType(dbTicketType)
}

func (r *Repository) UpdateTicketType(ctx context.Context, ticketTypeID uuid.UUID, input UpdateTicketTypeInput) (TicketType, error) {
	dbTicketType, err := r.queries.UpdateTicketType(ctx, sqlc.UpdateTicketTypeParams{
		Name:        strings.TrimSpace(input.Name),
		Description: nullableString(strings.TrimSpace(input.Description)),
		Price:       input.Price,
		Quantity:    input.Quantity,
		MaxPerOrder: input.MaxPerOrder,
		ID:          ticketTypeID.String(),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return TicketType{}, ErrTicketTypeNotFound
		}

		return TicketType{}, err
	}

	return mapTicketType(dbTicketType)
}

func (r *Repository) DeleteTicketType(ctx context.Context, ticketTypeID uuid.UUID) error {
	rowsAffected, err := r.queries.DeleteTicketType(ctx, ticketTypeID.String())
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrTicketTypeNotFound
	}

	return nil
}

func mapEvents(dbEvents []sqlc.Event) ([]Event, error) {
	items := make([]Event, 0, len(dbEvents))
	for _, dbEvent := range dbEvents {
		event, err := mapEvent(dbEvent)
		if err != nil {
			return nil, err
		}

		items = append(items, event)
	}

	return items, nil
}

func mapEvent(dbEvent sqlc.Event) (Event, error) {
	eventID, err := uuid.Parse(dbEvent.ID)
	if err != nil {
		return Event{}, err
	}

	organizerID, err := uuid.Parse(dbEvent.OrganizerID)
	if err != nil {
		return Event{}, err
	}

	categoryID, err := uuid.Parse(dbEvent.CategoryID)
	if err != nil {
		return Event{}, err
	}

	return Event{
		ID:           eventID,
		OrganizerID:  organizerID,
		CategoryID:   categoryID,
		Title:        dbEvent.Title,
		Slug:         dbEvent.Slug,
		Description:  dbEvent.Description,
		VenueName:    dbEvent.VenueName,
		VenueAddress: dbEvent.VenueAddress,
		City:         dbEvent.City,
		Country:      dbEvent.Country,
		Latitude:     nullableFloat64Ptr(dbEvent.Latitude),
		Longitude:    nullableFloat64Ptr(dbEvent.Longitude),
		BannerURL:    dbEvent.BannerURL.String,
		PosterURL:    dbEvent.PosterURL.String,
		Status:       dbEvent.Status,
		Currency:     dbEvent.Currency,
		IsFeatured:   dbEvent.IsFeatured,
		CreatedAt:    dbEvent.CreatedAt,
		UpdatedAt:    dbEvent.UpdatedAt,
	}, nil
}

func mapPublicEventListRow(row sqlc.ListPublicEventsRow) (PublicEvent, error) {
	return buildPublicEvent(
		row.ID,
		row.OrganizerID,
		row.OrganizerName,
		row.OrganizerSlug,
		row.CategoryID,
		row.CategoryName,
		row.CategorySlug,
		row.CategoryDescription,
		row.CategoryImageURL,
		row.Title,
		row.Slug,
		row.Description,
		row.VenueName,
		row.VenueAddress,
		row.City,
		row.Country,
		row.Latitude,
		row.Longitude,
		row.BannerURL,
		row.PosterURL,
		row.Status,
		row.Currency,
		row.IsFeatured,
		row.NextSessionID,
		row.NextSessionStartsAt,
		row.NextSessionEndsAt,
		row.NextSalesStartsAt,
		row.NextSalesEndsAt,
		row.PriceFrom,
		row.TicketsLeft,
		row.CreatedAt,
		row.UpdatedAt,
	)
}

func mapPublicEventDetailRow(row sqlc.GetPublicEventByIDRow) (PublicEvent, error) {
	return buildPublicEvent(
		row.ID,
		row.OrganizerID,
		row.OrganizerName,
		row.OrganizerSlug,
		row.CategoryID,
		row.CategoryName,
		row.CategorySlug,
		row.CategoryDescription,
		row.CategoryImageURL,
		row.Title,
		row.Slug,
		row.Description,
		row.VenueName,
		row.VenueAddress,
		row.City,
		row.Country,
		row.Latitude,
		row.Longitude,
		row.BannerURL,
		row.PosterURL,
		row.Status,
		row.Currency,
		row.IsFeatured,
		row.NextSessionID,
		row.NextSessionStartsAt,
		row.NextSessionEndsAt,
		row.NextSalesStartsAt,
		row.NextSalesEndsAt,
		row.PriceFrom,
		row.TicketsLeft,
		row.CreatedAt,
		row.UpdatedAt,
	)
}

func buildPublicEvent(
	eventIDText string,
	organizerIDText string,
	organizerName string,
	organizerSlug string,
	categoryIDText string,
	categoryName string,
	categorySlug string,
	categoryDescription sql.NullString,
	categoryImageURL sql.NullString,
	title string,
	slug string,
	description string,
	venueName string,
	venueAddress string,
	city string,
	country string,
	latitude sql.NullFloat64,
	longitude sql.NullFloat64,
	bannerURL sql.NullString,
	posterURL sql.NullString,
	status string,
	currency string,
	isFeatured bool,
	nextSessionIDText string,
	nextSessionStartsAt time.Time,
	nextSessionEndsAt time.Time,
	nextSalesStartsAt sql.NullTime,
	nextSalesEndsAt sql.NullTime,
	priceFrom float64,
	ticketsLeft int64,
	createdAt time.Time,
	updatedAt time.Time,
) (PublicEvent, error) {
	eventID, err := uuid.Parse(eventIDText)
	if err != nil {
		return PublicEvent{}, err
	}

	organizerID, err := uuid.Parse(organizerIDText)
	if err != nil {
		return PublicEvent{}, err
	}

	categoryID, err := uuid.Parse(categoryIDText)
	if err != nil {
		return PublicEvent{}, err
	}

	nextSessionID, err := uuid.Parse(nextSessionIDText)
	if err != nil {
		return PublicEvent{}, err
	}

	nextSessionIDValue := nextSessionID
	nextSessionStartsAtValue := nextSessionStartsAt
	nextSessionEndsAtValue := nextSessionEndsAt

	return PublicEvent{
		ID:                  eventID,
		OrganizerID:         organizerID,
		OrganizerName:       organizerName,
		OrganizerSlug:       organizerSlug,
		CategoryID:          categoryID,
		CategoryName:        categoryName,
		CategorySlug:        categorySlug,
		CategoryDescription: categoryDescription.String,
		CategoryImageURL:    categoryImageURL.String,
		Title:               title,
		Slug:                slug,
		Summary:             summarizeDescription(description),
		Description:         description,
		VenueName:           venueName,
		VenueAddress:        venueAddress,
		City:                city,
		Country:             country,
		Latitude:            nullableFloat64Ptr(latitude),
		Longitude:           nullableFloat64Ptr(longitude),
		BannerURL:           bannerURL.String,
		PosterURL:           posterURL.String,
		Status:              status,
		Currency:            currency,
		IsFeatured:          isFeatured,
		NextSessionID:       &nextSessionIDValue,
		NextSessionStartsAt: &nextSessionStartsAtValue,
		NextSessionEndsAt:   &nextSessionEndsAtValue,
		NextSalesStartsAt:   nullableTimePtr(nextSalesStartsAt),
		NextSalesEndsAt:     nullableTimePtr(nextSalesEndsAt),
		PriceFrom:           priceFrom,
		TicketsLeft:         int32(ticketsLeft),
		CreatedAt:           createdAt,
		UpdatedAt:           updatedAt,
	}, nil
}

func summarizeDescription(description string) string {
	trimmed := strings.TrimSpace(description)
	if trimmed == "" {
		return ""
	}

	runes := []rune(trimmed)
	if len(runes) <= 140 {
		return trimmed
	}

	summary := string(runes[:140])
	lastSpace := strings.LastIndex(summary, " ")
	if lastSpace > 90 {
		summary = summary[:lastSpace]
	}

	return strings.TrimSpace(summary) + "..."
}

func mapEventSessions(dbSessions []sqlc.EventSession) ([]EventSession, error) {
	items := make([]EventSession, 0, len(dbSessions))
	for _, dbSession := range dbSessions {
		item, err := mapEventSession(dbSession)
		if err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	return items, nil
}

func mapEventSession(dbSession sqlc.EventSession) (EventSession, error) {
	sessionID, err := uuid.Parse(dbSession.ID)
	if err != nil {
		return EventSession{}, err
	}

	eventID, err := uuid.Parse(dbSession.EventID)
	if err != nil {
		return EventSession{}, err
	}

	return EventSession{
		ID:            sessionID,
		EventID:       eventID,
		StartsAt:      dbSession.StartsAt,
		EndsAt:        dbSession.EndsAt,
		SalesStartsAt: nullableTimePtr(dbSession.SalesStartsAt),
		SalesEndsAt:   nullableTimePtr(dbSession.SalesEndsAt),
		Status:        dbSession.Status,
		CreatedAt:     dbSession.CreatedAt,
		UpdatedAt:     dbSession.UpdatedAt,
	}, nil
}

func mapTicketTypes(dbTicketTypes []sqlc.TicketType) ([]TicketType, error) {
	items := make([]TicketType, 0, len(dbTicketTypes))
	for _, dbTicketType := range dbTicketTypes {
		item, err := mapTicketType(dbTicketType)
		if err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	return items, nil
}

func mapTicketType(dbTicketType sqlc.TicketType) (TicketType, error) {
	ticketTypeID, err := uuid.Parse(dbTicketType.ID)
	if err != nil {
		return TicketType{}, err
	}

	sessionID, err := uuid.Parse(dbTicketType.EventSessionID)
	if err != nil {
		return TicketType{}, err
	}

	return TicketType{
		ID:             ticketTypeID,
		EventSessionID: sessionID,
		Name:           dbTicketType.Name,
		Description:    dbTicketType.Description.String,
		Price:          dbTicketType.Price,
		Quantity:       dbTicketType.Quantity,
		MaxPerOrder:    dbTicketType.MaxPerOrder,
		CreatedAt:      dbTicketType.CreatedAt,
		UpdatedAt:      dbTicketType.UpdatedAt,
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

func nullableTime(value *time.Time) sql.NullTime {
	if value == nil {
		return sql.NullTime{}
	}

	return sql.NullTime{
		Time:  *value,
		Valid: true,
	}
}

func nullableTimePtr(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}

	timeValue := value.Time
	return &timeValue
}

func nullableFloat64(value *float64) sql.NullFloat64 {
	if value == nil {
		return sql.NullFloat64{}
	}

	return sql.NullFloat64{
		Float64: *value,
		Valid:   true,
	}
}

func nullableFloat64Ptr(value sql.NullFloat64) *float64 {
	if !value.Valid {
		return nil
	}

	floatValue := value.Float64
	return &floatValue
}

type publicEventScanner interface {
	Scan(dest ...any) error
}

func listPublicEventsQuery(filterByEventID bool) string {
	query := `
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
      AND es.ends_at >= UTC_TIMESTAMP()`

	query += `
      AND EXISTS (
          SELECT 1
          FROM ticket_types tt
          WHERE tt.event_session_id = es.id
      )`

	if filterByEventID {
		query += `
      AND es.ends_at >= UTC_TIMESTAMP()`
	}

	query += `
      AND (es.sales_starts_at IS NULL OR es.sales_starts_at <= UTC_TIMESTAMP())
      AND (es.sales_ends_at IS NULL OR es.sales_ends_at >= UTC_TIMESTAMP())`

	query += `
),
reservation_summaries AS (
    SELECT
        tri.ticket_type_id,
        SUM(tri.quantity) AS reserved_quantity
    FROM ticket_reservation_items tri
    JOIN ticket_reservations tr ON tr.id = tri.reservation_id
    WHERE tr.expires_at > UTC_TIMESTAMP()
    GROUP BY tri.ticket_type_id
),
ticket_summaries AS (
    SELECT
        tt.event_session_id,
        MIN(tt.price) AS price_from,
        COALESCE(SUM(GREATEST(tt.quantity - COALESCE(rs.reserved_quantity, 0), 0)), 0) AS tickets_left
    FROM ticket_types tt
    LEFT JOIN reservation_summaries rs ON rs.ticket_type_id = tt.id
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
WHERE e.status = 'published'`

	if !filterByEventID {
		query += `
  AND e.id = ?`
	}

	query += `
`
	if filterByEventID {
		query += `ORDER BY e.is_featured DESC, ns.starts_at ASC, e.title ASC`
	} else {
		query += `LIMIT 1`
	}

	return query
}

func scanPublicEvent(scanner publicEventScanner) (PublicEvent, error) {
	var (
		eventIDText         string
		organizerIDText     string
		organizerName       string
		organizerSlug       string
		categoryIDText      string
		categoryName        string
		categorySlug        string
		categoryDescription sql.NullString
		categoryImageURL    sql.NullString
		title               string
		slug                string
		description         string
		venueName           string
		venueAddress        string
		city                string
		country             string
		latitude            sql.NullFloat64
		longitude           sql.NullFloat64
		bannerURL           sql.NullString
		posterURL           sql.NullString
		status              string
		currency            string
		isFeatured          bool
		createdAt           time.Time
		updatedAt           time.Time
		nextSessionIDText   string
		nextSessionStartsAt time.Time
		nextSessionEndsAt   time.Time
		nextSalesStartsAt   sql.NullTime
		nextSalesEndsAt     sql.NullTime
		priceFrom           float64
		ticketsLeft         int64
	)

	if err := scanner.Scan(
		&eventIDText,
		&organizerIDText,
		&organizerName,
		&organizerSlug,
		&categoryIDText,
		&categoryName,
		&categorySlug,
		&categoryDescription,
		&categoryImageURL,
		&title,
		&slug,
		&description,
		&venueName,
		&venueAddress,
		&city,
		&country,
		&latitude,
		&longitude,
		&bannerURL,
		&posterURL,
		&status,
		&currency,
		&isFeatured,
		&createdAt,
		&updatedAt,
		&nextSessionIDText,
		&nextSessionStartsAt,
		&nextSessionEndsAt,
		&nextSalesStartsAt,
		&nextSalesEndsAt,
		&priceFrom,
		&ticketsLeft,
	); err != nil {
		return PublicEvent{}, err
	}

	return buildPublicEvent(
		eventIDText,
		organizerIDText,
		organizerName,
		organizerSlug,
		categoryIDText,
		categoryName,
		categorySlug,
		categoryDescription,
		categoryImageURL,
		title,
		slug,
		description,
		venueName,
		venueAddress,
		city,
		country,
		latitude,
		longitude,
		bannerURL,
		posterURL,
		status,
		currency,
		isFeatured,
		nextSessionIDText,
		nextSessionStartsAt,
		nextSessionEndsAt,
		nextSalesStartsAt,
		nextSalesEndsAt,
		priceFrom,
		ticketsLeft,
		createdAt,
		updatedAt,
	)
}

func buildInClause(count int) string {
	if count <= 0 {
		return ""
	}

	return strings.TrimRight(strings.Repeat("?,", count), ",")
}

type reservationTicketSnapshot struct {
	TicketTypeID      string
	TicketTypeName    string
	UnitPrice         float64
	TotalQuantity     int32
	MaxPerOrder       int32
	EventID           string
	EventTitle        string
	SessionID         string
	SessionStartsAt   time.Time
	SessionEndsAt     time.Time
	EventStatus       string
	SessionStatus     string
	Currency          string
	SalesStartsAt     sql.NullTime
	SalesEndsAt       sql.NullTime
	AvailableQuantity int32
}

func (r *Repository) CompletePastEventSessions(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `
UPDATE event_sessions
SET status = 'completed'
WHERE status = 'scheduled'
  AND ends_at <= UTC_TIMESTAMP()
`)
	return err
}

func (r *Repository) CleanupExpiredReservations(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `
DELETE FROM ticket_reservations
WHERE expires_at <= UTC_TIMESTAMP()
  AND id NOT IN (
      SELECT reservation_id
      FROM checkout_orders
  )
`)
	return err
}

func (r *Repository) UpsertReservation(ctx context.Context, input UpsertTicketReservationInput) (TicketReservation, error) {
	if len(input.Items) == 0 {
		return TicketReservation{}, ErrInvalidReservationItems
	}

	if err := r.CleanupExpiredReservations(ctx); err != nil {
		return TicketReservation{}, err
	}

	normalizedItems := make(map[string]int32)
	for _, item := range input.Items {
		ticketTypeID := strings.TrimSpace(item.TicketTypeID)
		if _, err := uuid.Parse(ticketTypeID); err != nil {
			return TicketReservation{}, ErrInvalidTicketTypeID
		}
		if item.Quantity <= 0 {
			return TicketReservation{}, ErrInvalidTicketQuantity
		}

		normalizedItems[ticketTypeID] += item.Quantity
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return TicketReservation{}, err
	}
	defer tx.Rollback()

	reservationID := uuid.New().String()
	reservationToken := uuid.New().String()
	currentReservationID := ""

	if strings.TrimSpace(input.ReservationID) != "" || strings.TrimSpace(input.ReservationToken) != "" {
		if strings.TrimSpace(input.ReservationToken) == "" {
			return TicketReservation{}, ErrInvalidReservationToken
		}
		if _, err := uuid.Parse(strings.TrimSpace(input.ReservationID)); err != nil {
			return TicketReservation{}, ErrInvalidReservationID
		}

		var existingID string
		if err := tx.QueryRowContext(ctx, `
SELECT id
FROM ticket_reservations
WHERE id = ?
  AND token = ?
  AND expires_at > UTC_TIMESTAMP()
FOR UPDATE
`, strings.TrimSpace(input.ReservationID), strings.TrimSpace(input.ReservationToken)).Scan(&existingID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return TicketReservation{}, ErrReservationNotFound
			}

			return TicketReservation{}, err
		}

		reservationID = existingID
		reservationToken = strings.TrimSpace(input.ReservationToken)
		currentReservationID = existingID
	}

	ticketTypeIDs := make([]string, 0, len(normalizedItems))
	args := make([]any, 0, len(normalizedItems))
	for ticketTypeID := range normalizedItems {
		ticketTypeIDs = append(ticketTypeIDs, ticketTypeID)
		args = append(args, ticketTypeID)
	}

	ticketRows, err := tx.QueryContext(ctx, `
SELECT
    tt.id,
    tt.name,
    tt.price,
    tt.quantity,
    tt.max_per_order,
    e.id,
    e.title,
    es.id,
    es.starts_at,
    es.ends_at,
    e.status,
    es.status,
    e.currency,
    es.sales_starts_at,
    es.sales_ends_at
FROM ticket_types tt
JOIN event_sessions es ON es.id = tt.event_session_id
JOIN events e ON e.id = es.event_id
WHERE tt.id IN (`+buildInClause(len(ticketTypeIDs))+`)
FOR UPDATE
`, args...)
	if err != nil {
		return TicketReservation{}, err
	}
	defer ticketRows.Close()

	ticketSnapshots := make(map[string]reservationTicketSnapshot, len(ticketTypeIDs))
	for ticketRows.Next() {
		var snapshot reservationTicketSnapshot
		if err := ticketRows.Scan(
			&snapshot.TicketTypeID,
			&snapshot.TicketTypeName,
			&snapshot.UnitPrice,
			&snapshot.TotalQuantity,
			&snapshot.MaxPerOrder,
			&snapshot.EventID,
			&snapshot.EventTitle,
			&snapshot.SessionID,
			&snapshot.SessionStartsAt,
			&snapshot.SessionEndsAt,
			&snapshot.EventStatus,
			&snapshot.SessionStatus,
			&snapshot.Currency,
			&snapshot.SalesStartsAt,
			&snapshot.SalesEndsAt,
		); err != nil {
			return TicketReservation{}, err
		}

		ticketSnapshots[snapshot.TicketTypeID] = snapshot
	}
	if err := ticketRows.Err(); err != nil {
		return TicketReservation{}, err
	}
	if len(ticketSnapshots) != len(ticketTypeIDs) {
		return TicketReservation{}, ErrTicketTypeNotFound
	}

	reservationArgs := make([]any, 0, len(ticketTypeIDs))
	for _, ticketTypeID := range ticketTypeIDs {
		reservationArgs = append(reservationArgs, ticketTypeID)
	}
	reservedByOthersQuery := `
SELECT
    tri.ticket_type_id,
    COALESCE(SUM(tri.quantity), 0) AS reserved_quantity
FROM ticket_reservation_items tri
JOIN ticket_reservations tr ON tr.id = tri.reservation_id
WHERE tri.ticket_type_id IN (` + buildInClause(len(ticketTypeIDs)) + `)
  AND tr.expires_at > UTC_TIMESTAMP()`
	if currentReservationID != "" {
		reservedByOthersQuery += `
  AND tr.id <> ?`
		reservationArgs = append(reservationArgs, currentReservationID)
	}
	reservedByOthersQuery += `
GROUP BY tri.ticket_type_id
`

	reservedRows, err := tx.QueryContext(ctx, reservedByOthersQuery, reservationArgs...)
	if err != nil {
		return TicketReservation{}, err
	}
	defer reservedRows.Close()

	reservedByOthers := make(map[string]int32, len(ticketTypeIDs))
	for reservedRows.Next() {
		var ticketTypeID string
		var reservedQuantity int32
		if err := reservedRows.Scan(&ticketTypeID, &reservedQuantity); err != nil {
			return TicketReservation{}, err
		}
		reservedByOthers[ticketTypeID] = reservedQuantity
	}
	if err := reservedRows.Err(); err != nil {
		return TicketReservation{}, err
	}

	now := time.Now().UTC()
	for ticketTypeID, requestedQuantity := range normalizedItems {
		snapshot := ticketSnapshots[ticketTypeID]
		if snapshot.EventStatus != "published" || snapshot.SessionStatus != "scheduled" {
			return TicketReservation{}, ErrTicketSalesUnavailable
		}
		if snapshot.SalesStartsAt.Valid && now.Before(snapshot.SalesStartsAt.Time) {
			return TicketReservation{}, ErrTicketSalesUnavailable
		}
		if snapshot.SalesEndsAt.Valid && now.After(snapshot.SalesEndsAt.Time) {
			return TicketReservation{}, ErrTicketSalesUnavailable
		}
		if requestedQuantity > snapshot.MaxPerOrder {
			return TicketReservation{}, ErrReservationMaxPerOrder
		}

		availableQuantity := snapshot.TotalQuantity - reservedByOthers[ticketTypeID]
		if availableQuantity < 0 {
			availableQuantity = 0
		}
		if requestedQuantity > availableQuantity {
			return TicketReservation{}, ErrReservationUnavailable
		}
	}

	if currentReservationID == "" {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO ticket_reservations (
    id,
    token,
    expires_at
) VALUES (
    ?,
    ?,
    DATE_ADD(UTC_TIMESTAMP(), INTERVAL 10 MINUTE)
)
`, reservationID, reservationToken); err != nil {
			return TicketReservation{}, err
		}
	} else {
		if _, err := tx.ExecContext(ctx, `
UPDATE ticket_reservations
SET expires_at = DATE_ADD(UTC_TIMESTAMP(), INTERVAL 10 MINUTE)
WHERE id = ?
`, reservationID); err != nil {
			return TicketReservation{}, err
		}
	}

	if _, err := tx.ExecContext(ctx, `
DELETE FROM ticket_reservation_items
WHERE reservation_id = ?
`, reservationID); err != nil {
		return TicketReservation{}, err
	}

	for ticketTypeID, requestedQuantity := range normalizedItems {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO ticket_reservation_items (
    id,
    reservation_id,
    ticket_type_id,
    quantity
) VALUES (
    ?,
    ?,
    ?,
    ?
)
`, uuid.New().String(), reservationID, ticketTypeID, requestedQuantity); err != nil {
			return TicketReservation{}, err
		}
	}

	reservation, err := r.getReservationTx(ctx, tx, reservationID, reservationToken)
	if err != nil {
		return TicketReservation{}, err
	}

	if err := tx.Commit(); err != nil {
		return TicketReservation{}, err
	}

	return reservation, nil
}

func (r *Repository) GetReservation(ctx context.Context, reservationID uuid.UUID, reservationToken string) (TicketReservation, error) {
	if err := r.CleanupExpiredReservations(ctx); err != nil {
		return TicketReservation{}, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return TicketReservation{}, err
	}
	defer tx.Rollback()

	reservation, err := r.getReservationTx(ctx, tx, reservationID.String(), reservationToken)
	if err != nil {
		return TicketReservation{}, err
	}

	if err := tx.Commit(); err != nil {
		return TicketReservation{}, err
	}

	return reservation, nil
}

func (r *Repository) DeleteReservation(ctx context.Context, reservationID uuid.UUID, reservationToken string) error {
	reservationToken = strings.TrimSpace(reservationToken)

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var existingID string
	if err := tx.QueryRowContext(ctx, `
SELECT id
FROM ticket_reservations
WHERE id = ?
  AND token = ?
LIMIT 1
FOR UPDATE
`, reservationID.String(), reservationToken).Scan(&existingID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrReservationNotFound
		}
		return err
	}

	// If a checkout order exists for this reservation, keep it.
	// Deleting the reservation would cascade-delete the checkout order.
	var checkoutOrderID string
	if err := tx.QueryRowContext(ctx, `
SELECT id
FROM checkout_orders
WHERE reservation_id = ?
LIMIT 1
`, reservationID.String()).Scan(&checkoutOrderID); err == nil {
		return tx.Commit()
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
DELETE FROM ticket_reservations
WHERE id = ?
  AND token = ?
`, reservationID.String(), reservationToken); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *Repository) CreateCheckoutOrder(ctx context.Context, input CreateCheckoutOrderInput) (CheckoutOrder, error) {
	if err := r.CleanupExpiredReservations(ctx); err != nil {
		return CheckoutOrder{}, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return CheckoutOrder{}, err
	}
	defer tx.Rollback()

	var reservationIDText string
	var reservationExpiresAt time.Time
	if err := tx.QueryRowContext(ctx, `
SELECT id, expires_at
FROM ticket_reservations
WHERE id = ?
  AND token = ?
  AND expires_at > UTC_TIMESTAMP()
FOR UPDATE
`, input.ReservationID, input.ReservationToken).Scan(&reservationIDText, &reservationExpiresAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return CheckoutOrder{}, ErrReservationNotFound
		}

		return CheckoutOrder{}, err
	}

	var existingOrderID string
	var existingOrderToken string
	if err := tx.QueryRowContext(ctx, `
SELECT id, token
FROM checkout_orders
WHERE reservation_id = ?
LIMIT 1
`, reservationIDText).Scan(&existingOrderID, &existingOrderToken); err == nil {
		order, err := r.getCheckoutOrderTx(ctx, tx, existingOrderID, existingOrderToken)
		if err != nil {
			return CheckoutOrder{}, err
		}

		if err := tx.Commit(); err != nil {
			return CheckoutOrder{}, err
		}

		return order, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return CheckoutOrder{}, err
	}

	reservation, err := r.getReservationTx(ctx, tx, reservationIDText, input.ReservationToken)
	if err != nil {
		return CheckoutOrder{}, err
	}
	if len(reservation.Items) == 0 {
		return CheckoutOrder{}, ErrInvalidReservationItems
	}

	orderID := uuid.New().String()
	orderToken := uuid.New().String()
	orderNumber := fmt.Sprintf("EVT-%s", strings.ToUpper(strings.ReplaceAll(uuid.New().String()[:8], "-", "")))
	currency := reservation.Items[0].Currency
	subtotal := 0.0
	for _, item := range reservation.Items {
		subtotal += item.UnitPrice * float64(item.Quantity)
	}

	if _, err := tx.ExecContext(ctx, `
INSERT INTO checkout_orders (
    id,
    token,
    reservation_id,
    order_number,
    status,
    customer_name,
    customer_email,
    currency,
    subtotal,
    expires_at
) VALUES (?, ?, ?, ?, 'pending_payment', ?, ?, ?, ?, ?)
`, orderID, orderToken, reservationIDText, orderNumber, strings.TrimSpace(input.CustomerName), strings.TrimSpace(strings.ToLower(input.CustomerEmail)), currency, subtotal, reservationExpiresAt); err != nil {
		return CheckoutOrder{}, err
	}

	for _, item := range reservation.Items {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO checkout_order_items (
    id,
    order_id,
    ticket_type_id,
    ticket_type_name,
    quantity,
    unit_price,
    currency,
    event_id,
    event_title,
    session_id,
    session_starts_at,
    session_ends_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`, uuid.New().String(), orderID, item.TicketTypeID.String(), item.TicketTypeName, item.Quantity, item.UnitPrice, item.Currency, item.EventID.String(), item.EventTitle, item.SessionID.String(), item.SessionStartsAt, item.SessionEndsAt); err != nil {
			return CheckoutOrder{}, err
		}
	}

	order, err := r.getCheckoutOrderTx(ctx, tx, orderID, orderToken)
	if err != nil {
		return CheckoutOrder{}, err
	}

	if err := tx.Commit(); err != nil {
		return CheckoutOrder{}, err
	}

	return order, nil
}

func (r *Repository) GetCheckoutOrder(ctx context.Context, orderID uuid.UUID, orderToken string) (CheckoutOrder, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return CheckoutOrder{}, err
	}
	defer tx.Rollback()

	order, err := r.getCheckoutOrderTx(ctx, tx, orderID.String(), orderToken)
	if err != nil {
		return CheckoutOrder{}, err
	}

	if err := tx.Commit(); err != nil {
		return CheckoutOrder{}, err
	}

	return order, nil
}

func (r *Repository) getReservationTx(ctx context.Context, tx *sql.Tx, reservationID string, reservationToken string) (TicketReservation, error) {
	var reservation TicketReservation
	var reservationIDText string
	if err := tx.QueryRowContext(ctx, `
SELECT id, token, expires_at, created_at, updated_at
FROM ticket_reservations
WHERE id = ?
  AND token = ?
  AND expires_at > UTC_TIMESTAMP()
LIMIT 1
`, reservationID, reservationToken).Scan(
		&reservationIDText,
		&reservation.Token,
		&reservation.ExpiresAt,
		&reservation.CreatedAt,
		&reservation.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return TicketReservation{}, ErrReservationNotFound
		}
		return TicketReservation{}, err
	}

	parsedReservationID, err := uuid.Parse(reservationIDText)
	if err != nil {
		return TicketReservation{}, err
	}
	reservation.ID = parsedReservationID

	itemRows, err := tx.QueryContext(ctx, `
SELECT
    tri.ticket_type_id,
    tri.quantity,
    tt.name,
    tt.price,
    tt.max_per_order,
    tt.quantity,
    e.id,
    e.title,
    e.currency,
    es.id,
    es.starts_at,
    es.ends_at
FROM ticket_reservation_items tri
JOIN ticket_types tt ON tt.id = tri.ticket_type_id
JOIN event_sessions es ON es.id = tt.event_session_id
JOIN events e ON e.id = es.event_id
WHERE tri.reservation_id = ?
ORDER BY e.title ASC, es.starts_at ASC, tt.name ASC
`, reservationID)
	if err != nil {
		return TicketReservation{}, err
	}
	defer itemRows.Close()

	type itemRow struct {
		ticketTypeID    string
		quantity        int32
		ticketTypeName  string
		unitPrice       float64
		maxPerOrder     int32
		totalQuantity   int32
		eventID         string
		eventTitle      string
		currency        string
		sessionID       string
		sessionStartsAt time.Time
		sessionEndsAt   time.Time
	}

	rawItems := make([]itemRow, 0)
	ticketTypeIDs := make([]string, 0)
	for itemRows.Next() {
		var item itemRow
		if err := itemRows.Scan(
			&item.ticketTypeID,
			&item.quantity,
			&item.ticketTypeName,
			&item.unitPrice,
			&item.maxPerOrder,
			&item.totalQuantity,
			&item.eventID,
			&item.eventTitle,
			&item.currency,
			&item.sessionID,
			&item.sessionStartsAt,
			&item.sessionEndsAt,
		); err != nil {
			return TicketReservation{}, err
		}

		rawItems = append(rawItems, item)
		ticketTypeIDs = append(ticketTypeIDs, item.ticketTypeID)
	}
	if err := itemRows.Err(); err != nil {
		return TicketReservation{}, err
	}

	availableByTicketType := make(map[string]int32, len(ticketTypeIDs))
	if len(ticketTypeIDs) > 0 {
		reservedByOthersArgs := make([]any, 0, len(ticketTypeIDs)+1)
		for _, ticketTypeID := range ticketTypeIDs {
			reservedByOthersArgs = append(reservedByOthersArgs, ticketTypeID)
		}
		reservedByOthersArgs = append(reservedByOthersArgs, reservationID)

		rows, err := tx.QueryContext(ctx, `
SELECT
    tri.ticket_type_id,
    COALESCE(SUM(tri.quantity), 0) AS reserved_quantity
FROM ticket_reservation_items tri
JOIN ticket_reservations tr ON tr.id = tri.reservation_id
WHERE tri.ticket_type_id IN (`+buildInClause(len(ticketTypeIDs))+`)
  AND tr.expires_at > UTC_TIMESTAMP()
  AND tr.id <> ?
GROUP BY tri.ticket_type_id
`, reservedByOthersArgs...)
		if err != nil {
			return TicketReservation{}, err
		}
		defer rows.Close()

		reservedByOthers := make(map[string]int32, len(ticketTypeIDs))
		for rows.Next() {
			var ticketTypeID string
			var reservedQuantity int32
			if err := rows.Scan(&ticketTypeID, &reservedQuantity); err != nil {
				return TicketReservation{}, err
			}
			reservedByOthers[ticketTypeID] = reservedQuantity
		}
		if err := rows.Err(); err != nil {
			return TicketReservation{}, err
		}

		for _, item := range rawItems {
			availableQuantity := item.totalQuantity - reservedByOthers[item.ticketTypeID]
			if availableQuantity < 0 {
				availableQuantity = 0
			}
			availableByTicketType[item.ticketTypeID] = availableQuantity
		}
	}

	reservation.Items = make([]TicketReservationItem, 0, len(rawItems))
	for _, item := range rawItems {
		ticketTypeID, err := uuid.Parse(item.ticketTypeID)
		if err != nil {
			return TicketReservation{}, err
		}
		eventID, err := uuid.Parse(item.eventID)
		if err != nil {
			return TicketReservation{}, err
		}
		sessionID, err := uuid.Parse(item.sessionID)
		if err != nil {
			return TicketReservation{}, err
		}

		reservation.Items = append(reservation.Items, TicketReservationItem{
			TicketTypeID:      ticketTypeID,
			TicketTypeName:    item.ticketTypeName,
			Quantity:          item.quantity,
			UnitPrice:         item.unitPrice,
			Currency:          item.currency,
			MaxPerOrder:       item.maxPerOrder,
			AvailableQuantity: availableByTicketType[item.ticketTypeID],
			EventID:           eventID,
			EventTitle:        item.eventTitle,
			SessionID:         sessionID,
			SessionStartsAt:   item.sessionStartsAt,
			SessionEndsAt:     item.sessionEndsAt,
		})
	}

	return reservation, nil
}

func (r *Repository) getCheckoutOrderTx(ctx context.Context, tx *sql.Tx, orderID string, orderToken string) (CheckoutOrder, error) {
	var (
		order             CheckoutOrder
		orderIDText       string
		reservationIDText string
		stripeSessionID   sql.NullString
		paidAt            sql.NullTime
	)

	if err := tx.QueryRowContext(ctx, `
SELECT
    id,
    token,
    reservation_id,
    order_number,
    status,
    customer_name,
    customer_email,
    currency,
    subtotal,
    expires_at,
    created_at,
    updated_at,
    stripe_checkout_session_id,
    paid_at
FROM checkout_orders
WHERE id = ?
  AND token = ?
LIMIT 1
`, orderID, orderToken).Scan(
		&orderIDText,
		&order.Token,
		&reservationIDText,
		&order.OrderNumber,
		&order.Status,
		&order.CustomerName,
		&order.CustomerEmail,
		&order.Currency,
		&order.Subtotal,
		&order.ExpiresAt,
		&order.CreatedAt,
		&order.UpdatedAt,
		&stripeSessionID,
		&paidAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return CheckoutOrder{}, ErrCheckoutOrderNotFound
		}
		return CheckoutOrder{}, err
	}

	parsedOrderID, err := uuid.Parse(orderIDText)
	if err != nil {
		return CheckoutOrder{}, err
	}
	order.ID = parsedOrderID

	parsedReservationID, err := uuid.Parse(reservationIDText)
	if err != nil {
		return CheckoutOrder{}, err
	}
	order.ReservationID = parsedReservationID
	order.StripeSessionID = strings.TrimSpace(stripeSessionID.String)
	if paidAt.Valid {
		paidAtCopy := paidAt.Time
		order.PaidAt = &paidAtCopy
	}

	rows, err := tx.QueryContext(ctx, `
SELECT
    ticket_type_id,
    ticket_type_name,
    quantity,
    unit_price,
    currency,
    event_id,
    event_title,
    session_id,
    session_starts_at,
    session_ends_at
FROM checkout_order_items
WHERE order_id = ?
ORDER BY event_title ASC, session_starts_at ASC, ticket_type_name ASC
`, orderID)
	if err != nil {
		return CheckoutOrder{}, err
	}
	defer rows.Close()

	order.Items = make([]CheckoutOrderItem, 0)
	for rows.Next() {
		var (
			item             CheckoutOrderItem
			ticketTypeIDText string
			eventIDText      string
			sessionIDText    string
		)

		if err := rows.Scan(
			&ticketTypeIDText,
			&item.TicketTypeName,
			&item.Quantity,
			&item.UnitPrice,
			&item.Currency,
			&eventIDText,
			&item.EventTitle,
			&sessionIDText,
			&item.SessionStartsAt,
			&item.SessionEndsAt,
		); err != nil {
			return CheckoutOrder{}, err
		}

		item.TicketTypeID, err = uuid.Parse(ticketTypeIDText)
		if err != nil {
			return CheckoutOrder{}, err
		}
		item.EventID, err = uuid.Parse(eventIDText)
		if err != nil {
			return CheckoutOrder{}, err
		}
		item.SessionID, err = uuid.Parse(sessionIDText)
		if err != nil {
			return CheckoutOrder{}, err
		}

		order.Items = append(order.Items, item)
	}

	return order, rows.Err()
}

func (r *Repository) CreateStripeCheckoutSession(ctx context.Context, stripeSecretKey string, successURL string, cancelURL string, order CheckoutOrder) (*stripe.CheckoutSession, error) {
	stripeSecretKey = strings.TrimSpace(stripeSecretKey)
	if stripeSecretKey == "" {
		return nil, ErrStripeNotConfigured
	}

	successURL = strings.TrimSpace(successURL)
	cancelURL = strings.TrimSpace(cancelURL)
	if successURL == "" || cancelURL == "" {
		return nil, ErrStripeNotConfigured
	}

	stripe.Key = stripeSecretKey

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var (
		status            string
		existingSessionID sql.NullString
	)

	if err := tx.QueryRowContext(ctx, `
SELECT status, stripe_checkout_session_id
FROM checkout_orders
WHERE id = ?
  AND token = ?
LIMIT 1
FOR UPDATE
`, order.ID.String(), order.Token).Scan(&status, &existingSessionID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCheckoutOrderNotFound
		}
		return nil, err
	}

	if status != "pending_payment" {
		return nil, ErrCheckoutOrderNotPayable
	}

	if strings.TrimSpace(existingSessionID.String) != "" {
		session, err := checkoutsession.Get(existingSessionID.String, nil)
		if err != nil {
			return nil, err
		}

		if err := tx.Commit(); err != nil {
			return nil, err
		}

		return session, nil
	}

	lineItems := make([]*stripe.CheckoutSessionLineItemParams, 0, len(order.Items))
	for _, item := range order.Items {
		multiplier := stripeMinorUnitMultiplier(item.Currency)
		unitAmount := int64(math.Round(item.UnitPrice * float64(multiplier)))
		if unitAmount < 0 {
			unitAmount = 0
		}

		lineItems = append(lineItems, &stripe.CheckoutSessionLineItemParams{
			Quantity: stripe.Int64(int64(item.Quantity)),
			PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
				Currency: stripe.String(strings.ToLower(item.Currency)),
				ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
					Name: stripe.String(item.TicketTypeName),
					Metadata: map[string]string{
						"event_title": item.EventTitle,
						"session_id":  item.SessionID.String(),
					},
				},
				UnitAmount: stripe.Int64(unitAmount),
			},
		})
	}

	params := &stripe.CheckoutSessionParams{
		Mode:              stripe.String(string(stripe.CheckoutSessionModePayment)),
		ClientReferenceID: stripe.String(order.ID.String()),
		CustomerEmail:     stripe.String(order.CustomerEmail),
		SuccessURL:        stripe.String(successURL),
		CancelURL:         stripe.String(cancelURL),
		ExpiresAt: stripe.Int64(func() int64 {
			now := time.Now().UTC()
			minExpiry := now.Add(31 * time.Minute)
			maxExpiry := now.Add(23 * time.Hour)
			expiresAt := order.ExpiresAt.UTC()
			if expiresAt.Before(minExpiry) {
				expiresAt = minExpiry
			}
			if expiresAt.After(maxExpiry) {
				expiresAt = maxExpiry
			}
			return expiresAt.Unix()
		}()),
		Metadata: map[string]string{
			"order_id":     order.ID.String(),
			"order_number": order.OrderNumber,
		},
		LineItems: lineItems,
	}

	session, err := checkoutsession.New(params)
	if err != nil {
		if unsupportedCurrencyErr, ok := asUnsupportedCurrencyError(err); ok {
			return nil, unsupportedCurrencyErr
		}

		return nil, err
	}

	if _, err := tx.ExecContext(ctx, `
UPDATE checkout_orders
SET stripe_checkout_session_id = ?
WHERE id = ?
  AND token = ?
`, session.ID, order.ID.String(), order.Token); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return session, nil
}

func (r *Repository) GetCheckoutOrderByStripeSessionID(ctx context.Context, stripeSessionID string) (CheckoutOrderSummary, error) {
	stripeSessionID = strings.TrimSpace(stripeSessionID)
	if stripeSessionID == "" {
		return CheckoutOrderSummary{}, ErrInvalidStripeSessionID
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return CheckoutOrderSummary{}, err
	}
	defer tx.Rollback()

	order, err := r.getCheckoutOrderSummaryByStripeSessionTx(ctx, tx, stripeSessionID)
	if err != nil {
		return CheckoutOrderSummary{}, err
	}

	if err := tx.Commit(); err != nil {
		return CheckoutOrderSummary{}, err
	}

	return order, nil
}

func (r *Repository) GetCheckoutOrderSummaryByID(ctx context.Context, orderID uuid.UUID) (CheckoutOrderSummary, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return CheckoutOrderSummary{}, err
	}
	defer tx.Rollback()

	order, err := r.getCheckoutOrderSummaryByIDTx(ctx, tx, orderID.String())
	if err != nil {
		return CheckoutOrderSummary{}, err
	}

	if err := tx.Commit(); err != nil {
		return CheckoutOrderSummary{}, err
	}

	return order, nil
}

func (r *Repository) ListCheckoutOrdersByCustomerEmail(ctx context.Context, customerEmail string, limit int) ([]CheckoutOrderSummary, error) {
	customerEmail = strings.TrimSpace(strings.ToLower(customerEmail))
	if customerEmail == "" {
		return []CheckoutOrderSummary{}, nil
	}

	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, `
SELECT id
FROM checkout_orders
WHERE customer_email = ?
ORDER BY created_at DESC
LIMIT ?
`, customerEmail, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orderIDs := make([]string, 0)
	for rows.Next() {
		var orderID string
		if err := rows.Scan(&orderID); err != nil {
			return nil, err
		}
		orderIDs = append(orderIDs, orderID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	items := make([]CheckoutOrderSummary, 0, len(orderIDs))
	for _, orderID := range orderIDs {
		order, err := r.getCheckoutOrderSummaryByIDTx(ctx, tx, orderID)
		if err != nil {
			if errors.Is(err, ErrCheckoutOrderNotFound) {
				continue
			}
			return nil, err
		}
		items = append(items, order)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return items, nil
}

func (r *Repository) getCheckoutOrderSummaryByIDTx(ctx context.Context, tx *sql.Tx, orderID string) (CheckoutOrderSummary, error) {
	var (
		order             CheckoutOrderSummary
		orderIDText       string
		stripeSessionIDDB sql.NullString
		paidAt            sql.NullTime
	)

	if err := tx.QueryRowContext(ctx, `
SELECT
    id,
    order_number,
    status,
    customer_name,
    customer_email,
    currency,
    subtotal,
    expires_at,
    created_at,
    updated_at,
    stripe_checkout_session_id,
    paid_at
FROM checkout_orders
WHERE id = ?
LIMIT 1
`, orderID).Scan(
		&orderIDText,
		&order.OrderNumber,
		&order.Status,
		&order.CustomerName,
		&order.CustomerEmail,
		&order.Currency,
		&order.Subtotal,
		&order.ExpiresAt,
		&order.CreatedAt,
		&order.UpdatedAt,
		&stripeSessionIDDB,
		&paidAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return CheckoutOrderSummary{}, ErrCheckoutOrderNotFound
		}
		return CheckoutOrderSummary{}, err
	}

	parsedOrderID, err := uuid.Parse(orderIDText)
	if err != nil {
		return CheckoutOrderSummary{}, err
	}
	order.ID = parsedOrderID
	order.StripeSessionID = strings.TrimSpace(stripeSessionIDDB.String)
	if paidAt.Valid {
		paidAtCopy := paidAt.Time
		order.PaidAt = &paidAtCopy
	}

	rows, err := tx.QueryContext(ctx, `
SELECT
    ticket_type_id,
    ticket_type_name,
    quantity,
    unit_price,
    currency,
    event_id,
    event_title,
    session_id,
    session_starts_at,
    session_ends_at
FROM checkout_order_items
WHERE order_id = ?
ORDER BY event_title ASC, session_starts_at ASC, ticket_type_name ASC
`, orderIDText)
	if err != nil {
		return CheckoutOrderSummary{}, err
	}
	defer rows.Close()

	order.Items = make([]CheckoutOrderItem, 0)
	for rows.Next() {
		var (
			item             CheckoutOrderItem
			ticketTypeIDText string
			eventIDText      string
			sessionIDText    string
		)

		if err := rows.Scan(
			&ticketTypeIDText,
			&item.TicketTypeName,
			&item.Quantity,
			&item.UnitPrice,
			&item.Currency,
			&eventIDText,
			&item.EventTitle,
			&sessionIDText,
			&item.SessionStartsAt,
			&item.SessionEndsAt,
		); err != nil {
			return CheckoutOrderSummary{}, err
		}

		item.TicketTypeID, err = uuid.Parse(ticketTypeIDText)
		if err != nil {
			return CheckoutOrderSummary{}, err
		}
		item.EventID, err = uuid.Parse(eventIDText)
		if err != nil {
			return CheckoutOrderSummary{}, err
		}
		item.SessionID, err = uuid.Parse(sessionIDText)
		if err != nil {
			return CheckoutOrderSummary{}, err
		}

		order.Items = append(order.Items, item)
	}

	if err := rows.Err(); err != nil {
		return CheckoutOrderSummary{}, err
	}

	return order, nil
}

func (r *Repository) getCheckoutOrderSummaryByStripeSessionTx(ctx context.Context, tx *sql.Tx, stripeSessionID string) (CheckoutOrderSummary, error) {
	var (
		order             CheckoutOrderSummary
		orderIDText       string
		stripeSessionIDDB sql.NullString
		paidAt            sql.NullTime
	)

	if err := tx.QueryRowContext(ctx, `
SELECT
    id,
    order_number,
    status,
    customer_name,
    customer_email,
    currency,
    subtotal,
    expires_at,
    created_at,
    updated_at,
    stripe_checkout_session_id,
    paid_at
FROM checkout_orders
WHERE stripe_checkout_session_id = ?
LIMIT 1
`, stripeSessionID).Scan(
		&orderIDText,
		&order.OrderNumber,
		&order.Status,
		&order.CustomerName,
		&order.CustomerEmail,
		&order.Currency,
		&order.Subtotal,
		&order.ExpiresAt,
		&order.CreatedAt,
		&order.UpdatedAt,
		&stripeSessionIDDB,
		&paidAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return CheckoutOrderSummary{}, ErrCheckoutOrderNotFound
		}
		return CheckoutOrderSummary{}, err
	}

	parsedOrderID, err := uuid.Parse(orderIDText)
	if err != nil {
		return CheckoutOrderSummary{}, err
	}
	order.ID = parsedOrderID
	order.StripeSessionID = strings.TrimSpace(stripeSessionIDDB.String)
	if paidAt.Valid {
		paidAtCopy := paidAt.Time
		order.PaidAt = &paidAtCopy
	}

	rows, err := tx.QueryContext(ctx, `
SELECT
    ticket_type_id,
    ticket_type_name,
    quantity,
    unit_price,
    currency,
    event_id,
    event_title,
    session_id,
    session_starts_at,
    session_ends_at
FROM checkout_order_items
WHERE order_id = ?
ORDER BY event_title ASC, session_starts_at ASC, ticket_type_name ASC
`, orderIDText)
	if err != nil {
		return CheckoutOrderSummary{}, err
	}
	defer rows.Close()

	order.Items = make([]CheckoutOrderItem, 0)
	for rows.Next() {
		var (
			item             CheckoutOrderItem
			ticketTypeIDText string
			eventIDText      string
			sessionIDText    string
		)

		if err := rows.Scan(
			&ticketTypeIDText,
			&item.TicketTypeName,
			&item.Quantity,
			&item.UnitPrice,
			&item.Currency,
			&eventIDText,
			&item.EventTitle,
			&sessionIDText,
			&item.SessionStartsAt,
			&item.SessionEndsAt,
		); err != nil {
			return CheckoutOrderSummary{}, err
		}

		item.TicketTypeID, err = uuid.Parse(ticketTypeIDText)
		if err != nil {
			return CheckoutOrderSummary{}, err
		}
		item.EventID, err = uuid.Parse(eventIDText)
		if err != nil {
			return CheckoutOrderSummary{}, err
		}
		item.SessionID, err = uuid.Parse(sessionIDText)
		if err != nil {
			return CheckoutOrderSummary{}, err
		}

		order.Items = append(order.Items, item)
	}

	if err := rows.Err(); err != nil {
		return CheckoutOrderSummary{}, err
	}

	return order, nil
}

func (r *Repository) HandleStripeWebhook(ctx context.Context, stripeSecretKey string, webhookSecret string, payload []byte, signature string) error {
	stripe.Key = strings.TrimSpace(stripeSecretKey)

	webhookSecret = strings.TrimSpace(webhookSecret)
	if webhookSecret == "" {
		return ErrStripeNotConfigured
	}

	event, err := webhook.ConstructEventWithOptions(payload, signature, webhookSecret, webhook.ConstructEventOptions{
		IgnoreAPIVersionMismatch: true,
	})
	if err != nil {
		return err
	}

	switch event.Type {
	case "checkout.session.completed", "checkout.session.async_payment_succeeded":
		var session stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
			return err
		}

		paymentIntentID := ""
		if session.PaymentIntent != nil {
			paymentIntentID = session.PaymentIntent.ID
		}

		customerID := ""
		if session.Customer != nil {
			customerID = session.Customer.ID
		}

		return r.markCheckoutOrderPaidByStripeSession(ctx, session.ID, paymentIntentID, customerID)
	case "checkout.session.expired":
		var session stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
			return err
		}

		return r.markCheckoutOrderExpiredByStripeSession(ctx, session.ID)
	default:
		return nil
	}
}

func (r *Repository) markCheckoutOrderExpiredByStripeSession(ctx context.Context, stripeSessionID string) error {
	stripeSessionID = strings.TrimSpace(stripeSessionID)
	if stripeSessionID == "" {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var (
		reservationIDText string
		status            string
	)
	if err := tx.QueryRowContext(ctx, `
SELECT reservation_id, status
FROM checkout_orders
WHERE stripe_checkout_session_id = ?
LIMIT 1
FOR UPDATE
`, stripeSessionID).Scan(&reservationIDText, &status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}

	if status != "pending_payment" {
		return tx.Commit()
	}

	if _, err := tx.ExecContext(ctx, `
UPDATE checkout_orders
SET status = 'expired'
WHERE stripe_checkout_session_id = ?
  AND status = 'pending_payment'
`, stripeSessionID); err != nil {
		return err
	}

	// Release held inventory by expiring the reservation immediately.
	if _, err := tx.ExecContext(ctx, `
UPDATE ticket_reservations
SET expires_at = UTC_TIMESTAMP()
WHERE id = ?
`, reservationIDText); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *Repository) markCheckoutOrderPaidByStripeSession(ctx context.Context, stripeSessionID string, paymentIntentID string, customerID string) error {
	stripeSessionID = strings.TrimSpace(stripeSessionID)
	if stripeSessionID == "" {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var (
		orderIDText       string
		reservationIDText string
		status            string
		customerName      string
		customerEmail     string
	)

	if err := tx.QueryRowContext(ctx, `
SELECT id, reservation_id, status, customer_name, customer_email
FROM checkout_orders
WHERE stripe_checkout_session_id = ?
LIMIT 1
FOR UPDATE
`, stripeSessionID).Scan(&orderIDText, &reservationIDText, &status, &customerName, &customerEmail); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}

	if status == "paid" {
		return tx.Commit()
	}

	if status != "pending_payment" {
		return tx.Commit()
	}

	rows, err := tx.QueryContext(ctx, `
SELECT id, ticket_type_id, ticket_type_name, quantity, event_id, event_title, session_id, session_starts_at, session_ends_at
FROM checkout_order_items
WHERE order_id = ?
`, orderIDText)
	if err != nil {
		return err
	}
	defer rows.Close()

	type orderItem struct {
		id              string
		ticketTypeID    string
		ticketTypeName  string
		quantity        int32
		eventID         string
		eventTitle      string
		sessionID       string
		sessionStartsAt time.Time
		sessionEndsAt   time.Time
	}

	items := make([]orderItem, 0)
	for rows.Next() {
		var item orderItem
		if err := rows.Scan(
			&item.id,
			&item.ticketTypeID,
			&item.ticketTypeName,
			&item.quantity,
			&item.eventID,
			&item.eventTitle,
			&item.sessionID,
			&item.sessionStartsAt,
			&item.sessionEndsAt,
		); err != nil {
			return err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, item := range items {
		result, err := tx.ExecContext(ctx, `
UPDATE ticket_types
SET quantity = quantity - ?
WHERE id = ?
  AND quantity >= ?
`, item.quantity, item.ticketTypeID, item.quantity)
		if err != nil {
			return err
		}

		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return errors.New("insufficient ticket inventory for paid order")
		}
	}

	keepUntil := time.Now().UTC().AddDate(10, 0, 0)

	if _, err := tx.ExecContext(ctx, `
UPDATE ticket_reservations
SET expires_at = ?
WHERE id = ?
`, keepUntil, reservationIDText); err != nil {
		return err
	}

	// Reservation items are only meant to hold inventory pre-payment.
	// After the order is paid we keep the reservation row for auditing, but drop the items
	// so future holds don't double-count availability.
	if _, err := tx.ExecContext(ctx, `
DELETE FROM ticket_reservation_items
WHERE reservation_id = ?
`, reservationIDText); err != nil {
		return err
	}

	normalizedEmail := strings.ToLower(strings.TrimSpace(customerEmail))
	if normalizedEmail == "" {
		return errors.New("missing customer email for paid order")
	}

	for _, item := range items {
		for i := int32(0); i < item.quantity; i++ {
			ticketID := uuid.New()
			ticketCode := uuid.New()

			if _, err := tx.ExecContext(ctx, `
INSERT INTO tickets (
    id,
    code,
    order_id,
    order_item_id,
    customer_name,
    customer_email,
    ticket_type_id,
    ticket_type_name,
    event_id,
    event_title,
    session_id,
    session_starts_at,
    session_ends_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`, ticketID.String(),
				ticketCode.String(),
				orderIDText,
				item.id,
				strings.TrimSpace(customerName),
				normalizedEmail,
				item.ticketTypeID,
				item.ticketTypeName,
				item.eventID,
				item.eventTitle,
				item.sessionID,
				item.sessionStartsAt,
				item.sessionEndsAt,
			); err != nil {
				return err
			}
		}
	}

	if _, err := tx.ExecContext(ctx, `
UPDATE checkout_orders
SET status = 'paid',
    stripe_payment_intent_id = NULLIF(?, ''),
    stripe_customer_id = NULLIF(?, ''),
    paid_at = COALESCE(paid_at, UTC_TIMESTAMP()),
    expires_at = ?
WHERE id = ?
`, strings.TrimSpace(paymentIntentID), strings.TrimSpace(customerID), keepUntil, orderIDText); err != nil {
		return err
	}

	return tx.Commit()
}
