package events

import (
	"context"
	"database/sql"
	"errors"
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

func (r *Repository) List(ctx context.Context) ([]Event, error) {
	dbEvents, err := r.queries.ListEvents(ctx)
	if err != nil {
		return nil, err
	}

	return mapEvents(dbEvents)
}

func (r *Repository) ListByOrganizerID(ctx context.Context, organizerID uuid.UUID) ([]Event, error) {
	dbEvents, err := r.queries.ListEventsByOrganizerID(ctx, organizerID.String())
	if err != nil {
		return nil, err
	}

	return mapEvents(dbEvents)
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
		BannerURL:    dbEvent.BannerURL.String,
		PosterURL:    dbEvent.PosterURL.String,
		Status:       dbEvent.Status,
		Currency:     dbEvent.Currency,
		IsFeatured:   dbEvent.IsFeatured,
		CreatedAt:    dbEvent.CreatedAt,
		UpdatedAt:    dbEvent.UpdatedAt,
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
