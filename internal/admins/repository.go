package admins

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"eventy-api/internal/platform/roles"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateOrganizerAdmin(ctx context.Context, input CreateOrganizerAdminInput) (CreateOrganizerAdminResult, error) {
	passwordHash, err := hashPassword(input.AdminPassword)
	if err != nil {
		return CreateOrganizerAdminResult{}, err
	}

	organizerID := uuid.New()
	adminID := uuid.New()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return CreateOrganizerAdminResult{}, err
	}

	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	_, err = tx.ExecContext(ctx, `
INSERT INTO organizers (
    id,
    name,
    slug
) VALUES (
    ?,
    ?,
    ?
)`,
		organizerID.String(),
		input.OrganizerName,
		input.OrganizerSlug,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return CreateOrganizerAdminResult{}, ErrOrganizerSlugExists
		}

		return CreateOrganizerAdminResult{}, err
	}

	_, err = tx.ExecContext(ctx, `
INSERT INTO users (
    id,
    name,
    email,
    password_hash,
    role,
    organizer_id
) VALUES (
    ?,
    ?,
    ?,
    ?,
    ?,
    ?
)`,
		adminID.String(),
		input.AdminName,
		input.AdminEmail,
		passwordHash,
		roles.OrganizerAdmin,
		organizerID.String(),
	)
	if err != nil {
		if isUniqueViolation(err) {
			return CreateOrganizerAdminResult{}, ErrAdminEmailExists
		}

		return CreateOrganizerAdminResult{}, err
	}

	var result CreateOrganizerAdminResult
	row := tx.QueryRowContext(ctx, `
SELECT o.id, o.name, o.slug, o.created_at, o.updated_at,
       u.id, u.name, u.email, u.role, u.organizer_id, u.created_at, u.updated_at
FROM organizers o
JOIN users u ON u.organizer_id = o.id
WHERE o.id = ? AND u.id = ?
LIMIT 1
`, organizerID.String(), adminID.String())

	var organizerIDText string
	var adminIDText string
	var adminOrganizerID string
	err = row.Scan(
		&organizerIDText,
		&result.Organizer.Name,
		&result.Organizer.Slug,
		&result.Organizer.CreatedAt,
		&result.Organizer.UpdatedAt,
		&adminIDText,
		&result.Admin.Name,
		&result.Admin.Email,
		&result.Admin.Role,
		&adminOrganizerID,
		&result.Admin.CreatedAt,
		&result.Admin.UpdatedAt,
	)
	if err != nil {
		return CreateOrganizerAdminResult{}, err
	}

	result.Organizer.ID, err = uuid.Parse(organizerIDText)
	if err != nil {
		return CreateOrganizerAdminResult{}, err
	}

	result.Admin.ID, err = uuid.Parse(adminIDText)
	if err != nil {
		return CreateOrganizerAdminResult{}, err
	}

	result.Admin.OrganizerID, err = uuid.Parse(adminOrganizerID)
	if err != nil {
		return CreateOrganizerAdminResult{}, err
	}

	if err := tx.Commit(); err != nil {
		return CreateOrganizerAdminResult{}, err
	}

	tx = nil

	return result, nil
}

func (r *Repository) UpdateOrganizer(ctx context.Context, organizerID uuid.UUID, input UpdateOrganizerInput) (Organizer, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `
SELECT EXISTS(
    SELECT 1
    FROM organizers
    WHERE id = ?
)
`, organizerID.String()).Scan(&exists)
	if err != nil {
		return Organizer{}, err
	}
	if !exists {
		return Organizer{}, ErrOrganizerNotFound
	}

	_, err = r.db.ExecContext(ctx, `
UPDATE organizers
SET name = ?,
    slug = ?
WHERE id = ?
`,
		input.OrganizerName,
		input.OrganizerSlug,
		organizerID.String(),
	)
	if err != nil {
		if isUniqueViolation(err) {
			return Organizer{}, ErrOrganizerSlugExists
		}

		return Organizer{}, err
	}

	var item Organizer
	var organizerIDText string
	err = r.db.QueryRowContext(ctx, `
SELECT id, name, slug, created_at, updated_at
FROM organizers
WHERE id = ?
LIMIT 1
`, organizerID.String()).Scan(
		&organizerIDText,
		&item.Name,
		&item.Slug,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Organizer{}, ErrOrganizerNotFound
		}

		return Organizer{}, err
	}

	item.ID, err = uuid.Parse(organizerIDText)
	if err != nil {
		return Organizer{}, err
	}

	return item, nil
}

func (r *Repository) GetOverview(ctx context.Context, organizerID *uuid.UUID, includePlatformMetrics bool) (AdminOverview, error) {
	overview := AdminOverview{
		Scope: "organizer_admin",
	}
	if includePlatformMetrics {
		overview.Scope = "super_admin"
	}

	scopeFilter := ""
	scopeAndFilter := ""
	scopeArgs := []any{}
	if organizerID != nil {
		scopeFilter = " WHERE e.organizer_id = ?"
		scopeAndFilter = " AND e.organizer_id = ?"
		scopeArgs = append(scopeArgs, organizerID.String())
	}

	if err := r.db.QueryRowContext(ctx, `
SELECT COUNT(*) AS events,
       COALESCE(SUM(CASE WHEN e.status = 'published' THEN 1 ELSE 0 END), 0) AS published_events,
       COALESCE(SUM(CASE WHEN e.status = 'draft' THEN 1 ELSE 0 END), 0) AS draft_events
FROM events e`+scopeFilter, scopeArgs...).Scan(
		&overview.Stats.Events,
		&overview.Stats.PublishedEvents,
		&overview.Stats.DraftEvents,
	); err != nil {
		return AdminOverview{}, err
	}

	if err := r.db.QueryRowContext(ctx, `
SELECT COUNT(*) AS sessions,
       COALESCE(SUM(CASE WHEN es.status = 'scheduled' THEN 1 ELSE 0 END), 0) AS scheduled_sessions
FROM event_sessions es
JOIN events e ON e.id = es.event_id`+scopeFilter, scopeArgs...).Scan(
		&overview.Stats.Sessions,
		&overview.Stats.ScheduledSessions,
	); err != nil {
		return AdminOverview{}, err
	}

	if err := r.db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM ticket_types tt
JOIN event_sessions es ON es.id = tt.event_session_id
JOIN events e ON e.id = es.event_id`+scopeFilter, scopeArgs...).Scan(
		&overview.Stats.TicketTypes,
	); err != nil {
		return AdminOverview{}, err
	}

	if includePlatformMetrics {
		if err := r.db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM categories
`).Scan(&overview.Stats.Categories); err != nil {
			return AdminOverview{}, err
		}
		if err := r.db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM organizers
`).Scan(&overview.Stats.Organizers); err != nil {
			return AdminOverview{}, err
		}
	}

	if err := r.db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM (
    SELECT e.id
    FROM events e
    LEFT JOIN event_sessions es ON es.event_id = e.id`+scopeFilter+`
    GROUP BY e.id
    HAVING COUNT(es.id) = 0
) AS events_without_sessions`, scopeArgs...).Scan(
		&overview.NeedsAttention.EventsWithoutSessionsCount,
	); err != nil {
		return AdminOverview{}, err
	}

	if err := r.db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM (
    SELECT es.id
    FROM event_sessions es
    JOIN events e ON e.id = es.event_id
    LEFT JOIN ticket_types tt ON tt.event_session_id = es.id`+scopeFilter+`
    GROUP BY es.id
    HAVING COUNT(tt.id) = 0
) AS sessions_without_ticket_types`, scopeArgs...).Scan(
		&overview.NeedsAttention.SessionsWithoutTicketTypesCount,
	); err != nil {
		return AdminOverview{}, err
	}

	overview.NeedsAttention.DraftEventsCount = overview.Stats.DraftEvents

	recentEventRows, err := r.db.QueryContext(ctx, `
SELECT e.id, e.organizer_id, e.category_id, e.title, e.slug, e.status,
       e.venue_name, e.city, e.country, o.name AS organizer_name, c.name AS category_name,
       e.created_at, e.updated_at
FROM events e
JOIN organizers o ON o.id = e.organizer_id
JOIN categories c ON c.id = e.category_id`+scopeFilter+`
ORDER BY e.created_at DESC, e.title ASC
LIMIT 5`, scopeArgs...)
	if err != nil {
		return AdminOverview{}, err
	}
	defer recentEventRows.Close()

	for recentEventRows.Next() {
		var item AdminOverviewEvent
		var eventIDText string
		var organizerIDText string
		var categoryIDText string

		if err := recentEventRows.Scan(
			&eventIDText,
			&organizerIDText,
			&categoryIDText,
			&item.Title,
			&item.Slug,
			&item.Status,
			&item.VenueName,
			&item.City,
			&item.Country,
			&item.OrganizerName,
			&item.CategoryName,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return AdminOverview{}, err
		}

		item.ID, err = uuid.Parse(eventIDText)
		if err != nil {
			return AdminOverview{}, err
		}
		item.OrganizerID, err = uuid.Parse(organizerIDText)
		if err != nil {
			return AdminOverview{}, err
		}
		item.CategoryID, err = uuid.Parse(categoryIDText)
		if err != nil {
			return AdminOverview{}, err
		}

		overview.RecentEvents = append(overview.RecentEvents, item)
	}
	if err := recentEventRows.Err(); err != nil {
		return AdminOverview{}, err
	}

	upcomingSessionRows, err := r.db.QueryContext(ctx, `
SELECT es.id, e.id, e.title, e.slug, e.currency, es.status, es.starts_at,
       COUNT(tt.id) AS ticket_type_count
FROM event_sessions es
JOIN events e ON e.id = es.event_id
LEFT JOIN ticket_types tt ON tt.event_session_id = es.id
WHERE es.status = 'scheduled' AND es.starts_at >= UTC_TIMESTAMP()`+scopeAndFilter+`
GROUP BY es.id, e.id, e.title, e.slug, e.currency, es.status, es.starts_at
ORDER BY es.starts_at ASC
LIMIT 6`, scopeArgs...)
	if err != nil {
		return AdminOverview{}, err
	}
	defer upcomingSessionRows.Close()

	for upcomingSessionRows.Next() {
		var item AdminOverviewSession
		var sessionIDText string
		var eventIDText string

		if err := upcomingSessionRows.Scan(
			&sessionIDText,
			&eventIDText,
			&item.EventTitle,
			&item.EventSlug,
			&item.EventCurrency,
			&item.Status,
			&item.StartsAt,
			&item.TicketTypeCount,
		); err != nil {
			return AdminOverview{}, err
		}

		item.ID, err = uuid.Parse(sessionIDText)
		if err != nil {
			return AdminOverview{}, err
		}
		item.EventID, err = uuid.Parse(eventIDText)
		if err != nil {
			return AdminOverview{}, err
		}

		overview.UpcomingSessions = append(overview.UpcomingSessions, item)
	}
	if err := upcomingSessionRows.Err(); err != nil {
		return AdminOverview{}, err
	}

	attentionEventRows, err := r.db.QueryContext(ctx, `
SELECT e.id, e.title, e.slug, e.created_at
FROM events e
LEFT JOIN event_sessions es ON es.event_id = e.id`+scopeFilter+`
GROUP BY e.id, e.title, e.slug, e.created_at
HAVING COUNT(es.id) = 0
ORDER BY e.created_at DESC, e.title ASC
LIMIT 3`, scopeArgs...)
	if err != nil {
		return AdminOverview{}, err
	}
	defer attentionEventRows.Close()

	for attentionEventRows.Next() {
		var item AdminOverviewAttentionEvent
		var eventIDText string

		if err := attentionEventRows.Scan(
			&eventIDText,
			&item.Title,
			&item.Slug,
			&item.CreatedAt,
		); err != nil {
			return AdminOverview{}, err
		}

		item.ID, err = uuid.Parse(eventIDText)
		if err != nil {
			return AdminOverview{}, err
		}

		overview.NeedsAttention.EventsWithoutSessions = append(
			overview.NeedsAttention.EventsWithoutSessions,
			item,
		)
	}
	if err := attentionEventRows.Err(); err != nil {
		return AdminOverview{}, err
	}

	attentionSessionRows, err := r.db.QueryContext(ctx, `
SELECT es.id, e.id, e.title, es.starts_at, es.status
FROM event_sessions es
JOIN events e ON e.id = es.event_id
LEFT JOIN ticket_types tt ON tt.event_session_id = es.id`+scopeFilter+`
GROUP BY es.id, e.id, e.title, es.starts_at, es.status
HAVING COUNT(tt.id) = 0
ORDER BY es.starts_at ASC, e.title ASC
LIMIT 2`, scopeArgs...)
	if err != nil {
		return AdminOverview{}, err
	}
	defer attentionSessionRows.Close()

	for attentionSessionRows.Next() {
		var item AdminOverviewAttentionSession
		var sessionIDText string
		var eventIDText string

		if err := attentionSessionRows.Scan(
			&sessionIDText,
			&eventIDText,
			&item.EventTitle,
			&item.StartsAt,
			&item.Status,
		); err != nil {
			return AdminOverview{}, err
		}

		item.ID, err = uuid.Parse(sessionIDText)
		if err != nil {
			return AdminOverview{}, err
		}
		item.EventID, err = uuid.Parse(eventIDText)
		if err != nil {
			return AdminOverview{}, err
		}

		overview.NeedsAttention.SessionsWithoutTicketTypes = append(
			overview.NeedsAttention.SessionsWithoutTicketTypes,
			item,
		)
	}
	if err := attentionSessionRows.Err(); err != nil {
		return AdminOverview{}, err
	}

	if includePlatformMetrics {
		organizerRows, err := r.db.QueryContext(ctx, `
SELECT o.id, o.name, o.slug,
       COALESCE(ec.event_count, 0) AS event_count,
       COALESCE(sc.session_count, 0) AS session_count
FROM organizers o
LEFT JOIN (
    SELECT organizer_id, COUNT(*) AS event_count
    FROM events
    GROUP BY organizer_id
) ec ON ec.organizer_id = o.id
LEFT JOIN (
    SELECT e.organizer_id, COUNT(es.id) AS session_count
    FROM events e
    LEFT JOIN event_sessions es ON es.event_id = e.id
    GROUP BY e.organizer_id
) sc ON sc.organizer_id = o.id
ORDER BY o.name ASC
LIMIT 6
`)
		if err != nil {
			return AdminOverview{}, err
		}
		defer organizerRows.Close()

		for organizerRows.Next() {
			var item AdminOverviewOrganizerSummary
			var organizerIDText string

			if err := organizerRows.Scan(
				&organizerIDText,
				&item.Name,
				&item.Slug,
				&item.EventCount,
				&item.SessionCount,
			); err != nil {
				return AdminOverview{}, err
			}

			item.ID, err = uuid.Parse(organizerIDText)
			if err != nil {
				return AdminOverview{}, err
			}

			overview.OrganizerSummaries = append(overview.OrganizerSummaries, item)
		}
		if err := organizerRows.Err(); err != nil {
			return AdminOverview{}, err
		}
	}

	return overview, nil
}

func (r *Repository) ListOrganizers(ctx context.Context) ([]OrganizerListItem, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT o.id, o.name, o.slug,
       COALESCE(ac.admin_count, 0) AS admin_count,
       COALESCE(ec.event_count, 0) AS event_count,
       COALESCE(sc.session_count, 0) AS session_count,
       o.created_at, o.updated_at,
       u.id, u.name, u.email, u.role, u.organizer_id, u.created_at, u.updated_at
FROM organizers o
LEFT JOIN (
    SELECT organizer_id, COUNT(*) AS admin_count
    FROM users
    WHERE role = ?
    GROUP BY organizer_id
) ac ON ac.organizer_id = o.id
LEFT JOIN (
    SELECT organizer_id, COUNT(*) AS event_count
    FROM events
    GROUP BY organizer_id
) ec ON ec.organizer_id = o.id
LEFT JOIN (
    SELECT e.organizer_id, COUNT(es.id) AS session_count
    FROM events e
    LEFT JOIN event_sessions es ON es.event_id = e.id
    GROUP BY e.organizer_id
) sc ON sc.organizer_id = o.id
LEFT JOIN users u
    ON u.organizer_id = o.id
   AND u.role = ?
ORDER BY o.name ASC
`, roles.OrganizerAdmin, roles.OrganizerAdmin)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []OrganizerListItem
	for rows.Next() {
		var item OrganizerListItem
		var organizerID string
		var adminID sql.NullString
		var adminName sql.NullString
		var adminEmail sql.NullString
		var adminRole sql.NullString
		var adminOrganizerID sql.NullString
		var adminCreatedAt sql.NullTime
		var adminUpdatedAt sql.NullTime
		if err := rows.Scan(
			&organizerID,
			&item.Name,
			&item.Slug,
			&item.AdminCount,
			&item.EventCount,
			&item.SessionCount,
			&item.CreatedAt,
			&item.UpdatedAt,
			&adminID,
			&adminName,
			&adminEmail,
			&adminRole,
			&adminOrganizerID,
			&adminCreatedAt,
			&adminUpdatedAt,
		); err != nil {
			return nil, err
		}

		item.ID, err = uuid.Parse(organizerID)
		if err != nil {
			return nil, err
		}

		if adminID.Valid {
			admin := OrganizerAdmin{
				Name:  adminName.String,
				Email: adminEmail.String,
				Role:  adminRole.String,
			}

			admin.ID, err = uuid.Parse(adminID.String)
			if err != nil {
				return nil, err
			}

			admin.OrganizerID, err = uuid.Parse(adminOrganizerID.String)
			if err != nil {
				return nil, err
			}

			if adminCreatedAt.Valid {
				admin.CreatedAt = adminCreatedAt.Time
			}
			if adminUpdatedAt.Valid {
				admin.UpdatedAt = adminUpdatedAt.Time
			}

			item.Admin = &admin
		}

		items = append(items, item)
	}

	return items, rows.Err()
}

func (r *Repository) GetOrganizer(ctx context.Context, organizerID uuid.UUID) (OrganizerDetail, error) {
	var item OrganizerDetail
	var organizerIDText string
	var adminID sql.NullString
	var adminName sql.NullString
	var adminEmail sql.NullString
	var adminRole sql.NullString
	var adminOrganizerID sql.NullString
	var adminCreatedAt sql.NullTime
	var adminUpdatedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, `
SELECT o.id, o.name, o.slug,
       COALESCE(ac.admin_count, 0) AS admin_count,
       COALESCE(ec.event_count, 0) AS event_count,
       COALESCE(sc.session_count, 0) AS session_count,
       o.created_at, o.updated_at,
       u.id, u.name, u.email, u.role, u.organizer_id, u.created_at, u.updated_at
FROM organizers o
LEFT JOIN users u
    ON u.organizer_id = o.id
   AND u.role = ?
LEFT JOIN (
    SELECT organizer_id, COUNT(*) AS admin_count
    FROM users
    WHERE role = ?
    GROUP BY organizer_id
) ac ON ac.organizer_id = o.id
LEFT JOIN (
    SELECT organizer_id, COUNT(*) AS event_count
    FROM events
    GROUP BY organizer_id
) ec ON ec.organizer_id = o.id
LEFT JOIN (
    SELECT e.organizer_id, COUNT(es.id) AS session_count
    FROM events e
    LEFT JOIN event_sessions es ON es.event_id = e.id
    GROUP BY e.organizer_id
) sc ON sc.organizer_id = o.id
WHERE o.id = ?
LIMIT 1
`, roles.OrganizerAdmin, roles.OrganizerAdmin, organizerID.String()).Scan(
		&organizerIDText,
		&item.Organizer.Name,
		&item.Organizer.Slug,
		&item.Organizer.AdminCount,
		&item.Organizer.EventCount,
		&item.Organizer.SessionCount,
		&item.Organizer.CreatedAt,
		&item.Organizer.UpdatedAt,
		&adminID,
		&adminName,
		&adminEmail,
		&adminRole,
		&adminOrganizerID,
		&adminCreatedAt,
		&adminUpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return OrganizerDetail{}, ErrOrganizerNotFound
		}

		return OrganizerDetail{}, err
	}

	item.Organizer.ID, err = uuid.Parse(organizerIDText)
	if err != nil {
		return OrganizerDetail{}, err
	}

	if adminID.Valid {
		admin := OrganizerAdmin{
			Name:  adminName.String,
			Email: adminEmail.String,
			Role:  adminRole.String,
		}

		admin.ID, err = uuid.Parse(adminID.String)
		if err != nil {
			return OrganizerDetail{}, err
		}

		admin.OrganizerID, err = uuid.Parse(adminOrganizerID.String)
		if err != nil {
			return OrganizerDetail{}, err
		}

		if adminCreatedAt.Valid {
			admin.CreatedAt = adminCreatedAt.Time
		}
		if adminUpdatedAt.Valid {
			admin.UpdatedAt = adminUpdatedAt.Time
		}

		item.Organizer.Admin = &admin
	}

	rows, err := r.db.QueryContext(ctx, `
SELECT e.id, e.title, e.slug, e.status, e.currency, e.city, e.country,
       COUNT(DISTINCT es.id) AS session_count,
       COUNT(DISTINCT tt.id) AS ticket_type_count,
       MIN(CASE
           WHEN es.status = 'scheduled' AND es.starts_at >= UTC_TIMESTAMP() THEN es.starts_at
           ELSE NULL
       END) AS next_session_starts_at,
       e.created_at, e.updated_at
FROM events e
LEFT JOIN event_sessions es ON es.event_id = e.id
LEFT JOIN ticket_types tt ON tt.event_session_id = es.id
WHERE e.organizer_id = ?
GROUP BY e.id, e.title, e.slug, e.status, e.currency, e.city, e.country, e.created_at, e.updated_at
ORDER BY e.created_at DESC, e.title ASC
`, organizerID.String())
	if err != nil {
		return OrganizerDetail{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var event OrganizerManagedEvent
		var eventID string
		var nextSessionStartsAt sql.NullTime

		if err := rows.Scan(
			&eventID,
			&event.Title,
			&event.Slug,
			&event.Status,
			&event.Currency,
			&event.City,
			&event.Country,
			&event.SessionCount,
			&event.TicketTypeCount,
			&nextSessionStartsAt,
			&event.CreatedAt,
			&event.UpdatedAt,
		); err != nil {
			return OrganizerDetail{}, err
		}

		event.ID, err = uuid.Parse(eventID)
		if err != nil {
			return OrganizerDetail{}, err
		}

		if nextSessionStartsAt.Valid {
			nextSessionStartsAtValue := nextSessionStartsAt.Time
			event.NextSessionStarts = &nextSessionStartsAtValue
		}

		item.Events = append(item.Events, event)
	}
	if err := rows.Err(); err != nil {
		return OrganizerDetail{}, err
	}

	return item, nil
}

func (r *Repository) GetOrganizerAdmin(ctx context.Context, organizerID uuid.UUID) (OrganizerAdmin, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `
SELECT EXISTS(
    SELECT 1
    FROM organizers
    WHERE id = ?
)
`, organizerID.String()).Scan(&exists)
	if err != nil {
		return OrganizerAdmin{}, err
	}
	if !exists {
		return OrganizerAdmin{}, ErrOrganizerNotFound
	}

	row := r.db.QueryRowContext(ctx, `
SELECT id, name, email, role, organizer_id, created_at, updated_at
FROM users
WHERE organizer_id = ? AND role = ?
`, organizerID.String(), roles.OrganizerAdmin)

	var item OrganizerAdmin
	var adminID string
	var adminOrganizerID string
	if err := row.Scan(
		&adminID,
		&item.Name,
		&item.Email,
		&item.Role,
		&adminOrganizerID,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return OrganizerAdmin{}, ErrOrganizerAdminNotFound
		}
		return OrganizerAdmin{}, err
	}

	item.ID, err = uuid.Parse(adminID)
	if err != nil {
		return OrganizerAdmin{}, err
	}

	item.OrganizerID, err = uuid.Parse(adminOrganizerID)
	if err != nil {
		return OrganizerAdmin{}, err
	}

	return item, nil
}

func (r *Repository) UpdateOrganizerAdmin(ctx context.Context, organizerID uuid.UUID, input UpdateOrganizerAdminInput) (OrganizerAdmin, error) {
	result, err := r.db.ExecContext(ctx, `
UPDATE users
SET name = ?,
    email = ?
WHERE organizer_id = ? AND role = ?
`,
		input.AdminName,
		input.AdminEmail,
		organizerID.String(),
		roles.OrganizerAdmin,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return OrganizerAdmin{}, ErrAdminEmailExists
		}

		return OrganizerAdmin{}, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return OrganizerAdmin{}, err
	}

	if rowsAffected == 0 {
		var exists bool
		err = r.db.QueryRowContext(ctx, `
SELECT EXISTS(
    SELECT 1
    FROM organizers
    WHERE id = ?
)
`, organizerID.String()).Scan(&exists)
		if err != nil {
			return OrganizerAdmin{}, err
		}
		if !exists {
			return OrganizerAdmin{}, ErrOrganizerNotFound
		}

		return OrganizerAdmin{}, ErrOrganizerAdminNotFound
	}

	return r.GetOrganizerAdmin(ctx, organizerID)
}

func (r *Repository) ResetOrganizerAdminPassword(ctx context.Context, organizerID uuid.UUID, password string) error {
	passwordHash, err := hashPassword(password)
	if err != nil {
		return err
	}

	result, err := r.db.ExecContext(ctx, `
UPDATE users
SET password_hash = ?
WHERE organizer_id = ? AND role = ?
`, passwordHash, organizerID.String(), roles.OrganizerAdmin)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		var exists bool
		err = r.db.QueryRowContext(ctx, `
SELECT EXISTS(
    SELECT 1
    FROM organizers
    WHERE id = ?
)
`, organizerID.String()).Scan(&exists)
		if err != nil {
			return err
		}
		if !exists {
			return ErrOrganizerNotFound
		}

		return ErrOrganizerAdminNotFound
	}

	return nil
}

func (r *Repository) DeleteOrganizerAdmin(ctx context.Context, organizerID uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, `
DELETE FROM users
WHERE organizer_id = ? AND role = ?
`, organizerID.String(), roles.OrganizerAdmin)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		var exists bool
		err = r.db.QueryRowContext(ctx, `
SELECT EXISTS(
    SELECT 1
    FROM organizers
    WHERE id = ?
)
`, organizerID.String()).Scan(&exists)
		if err != nil {
			return err
		}
		if !exists {
			return ErrOrganizerNotFound
		}

		return ErrOrganizerAdminNotFound
	}

	return nil
}

func (r *Repository) DeleteOrganizer(ctx context.Context, organizerID uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	var exists bool
	err = tx.QueryRowContext(ctx, `
SELECT EXISTS(
    SELECT 1
    FROM organizers
    WHERE id = ?
)
`, organizerID.String()).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return ErrOrganizerNotFound
	}

	if _, err := tx.ExecContext(ctx, `
DELETE FROM users
WHERE organizer_id = ? AND role = ?
`, organizerID.String(), roles.OrganizerAdmin); err != nil {
		return err
	}

	result, err := tx.ExecContext(ctx, `
DELETE FROM organizers
WHERE id = ?
`, organizerID.String())
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrOrganizerNotFound
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	tx = nil

	return nil
}

func hashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(hashedPassword), nil
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}

	message := strings.ToLower(err.Error())
	return strings.Contains(message, "duplicate") || strings.Contains(message, "unique")
}
