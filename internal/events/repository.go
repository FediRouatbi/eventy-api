package events

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"eventy-api/internal/platform/db/sqlc"

	"github.com/google/uuid"
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
	rows, err := r.queries.ListPublicEvents(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]PublicEvent, 0, len(rows))
	for _, row := range rows {
		item, err := mapPublicEventListRow(row)
		if err != nil {
			return nil, err
		}

		sessions, err := r.ListPublicSessionsByEventID(ctx, item.ID)
		if err != nil {
			return nil, err
		}

		detailSessions := make([]EventSessionDetail, 0, len(sessions))
		for _, session := range sessions {
			ticketTypes, err := r.ListTicketTypesBySessionID(ctx, session.ID)
			if err != nil {
				return nil, err
			}

			detailSessions = append(detailSessions, EventSessionDetail{
				EventSession: session,
				TicketTypes:  ticketTypes,
			})
		}

		item.Sessions = detailSessions

		items = append(items, item)
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
	row, err := r.queries.GetPublicEventByID(ctx, eventID.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PublicEventDetail{}, ErrEventNotFound
		}

		return PublicEventDetail{}, err
	}

	event, err := mapPublicEventDetailRow(row)
	if err != nil {
		return PublicEventDetail{}, err
	}

	sessions, err := r.ListPublicSessionsByEventID(ctx, eventID)
	if err != nil {
		return PublicEventDetail{}, err
	}

	detailSessions := make([]EventSessionDetail, 0, len(sessions))
	for _, session := range sessions {
		ticketTypes, err := r.ListTicketTypesBySessionID(ctx, session.ID)
		if err != nil {
			return PublicEventDetail{}, err
		}

		detailSessions = append(detailSessions, EventSessionDetail{
			EventSession: session,
			TicketTypes:  ticketTypes,
		})
	}

	event.Sessions = detailSessions

	return PublicEventDetail{
		PublicEvent: event,
	}, nil
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
	dbSessions, err := r.queries.ListPublicEventSessionsByEventID(ctx, eventID.String())
	if err != nil {
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
