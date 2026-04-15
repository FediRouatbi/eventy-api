package tickets

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"eventy-api/internal/platform/jwt"
	"eventy-api/internal/platform/roles"

	"github.com/google/uuid"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) ListByCustomerEmail(ctx context.Context, email string, limit int) ([]Ticket, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return nil, nil
	}

	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	rows, err := r.db.QueryContext(ctx, `
SELECT
    t.id,
    t.code,
    t.order_id,
    co.order_number,
    co.status,
    t.customer_name,
    t.customer_email,
    t.ticket_type_id,
    t.ticket_type_name,
    t.event_id,
    t.event_title,
    t.session_id,
    t.session_starts_at,
    t.session_ends_at,
    co.paid_at,
    t.checked_in_at,
    t.checked_in_by_user_id,
    t.created_at,
    t.updated_at
FROM tickets t
JOIN checkout_orders co ON co.id = t.order_id
WHERE t.customer_email = ?
ORDER BY co.paid_at DESC, t.created_at DESC
LIMIT ?
`, email, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tickets := make([]Ticket, 0)
	for rows.Next() {
		var (
			idText           string
			orderIDText      string
			ticketTypeIDText string
			eventIDText      string
			sessionIDText    string
			paidAt           sql.NullTime
			checkedInAt      sql.NullTime
			checkedInBy      sql.NullString
		)

		var ticket Ticket
		if err := rows.Scan(
			&idText,
			&ticket.Code,
			&orderIDText,
			&ticket.OrderNumber,
			&ticket.OrderStatus,
			&ticket.CustomerName,
			&ticket.CustomerEmail,
			&ticketTypeIDText,
			&ticket.TicketTypeName,
			&eventIDText,
			&ticket.EventTitle,
			&sessionIDText,
			&ticket.SessionStartsAt,
			&ticket.SessionEndsAt,
			&paidAt,
			&checkedInAt,
			&checkedInBy,
			&ticket.CreatedAt,
			&ticket.UpdatedAt,
		); err != nil {
			return nil, err
		}

		parsedID, err := uuid.Parse(idText)
		if err != nil {
			return nil, err
		}
		ticket.ID = parsedID

		parsedOrderID, err := uuid.Parse(orderIDText)
		if err != nil {
			return nil, err
		}
		ticket.OrderID = parsedOrderID

		parsedTicketTypeID, err := uuid.Parse(ticketTypeIDText)
		if err != nil {
			return nil, err
		}
		ticket.TicketTypeID = parsedTicketTypeID

		parsedEventID, err := uuid.Parse(eventIDText)
		if err != nil {
			return nil, err
		}
		ticket.EventID = parsedEventID

		parsedSessionID, err := uuid.Parse(sessionIDText)
		if err != nil {
			return nil, err
		}
		ticket.SessionID = parsedSessionID

		if paidAt.Valid {
			value := paidAt.Time
			ticket.PaidAt = &value
		}

		if checkedInAt.Valid {
			value := checkedInAt.Time
			ticket.CheckedInAt = &value
		}

		if checkedInBy.Valid && strings.TrimSpace(checkedInBy.String) != "" {
			value, err := uuid.Parse(checkedInBy.String)
			if err != nil {
				return nil, err
			}
			ticket.CheckedInBy = &value
		}

		tickets = append(tickets, ticket)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tickets, nil
}

func (r *Repository) ListByOrderID(ctx context.Context, orderID uuid.UUID) ([]Ticket, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT
    t.id,
    t.code,
    t.order_id,
    co.order_number,
    co.status,
    t.customer_name,
    t.customer_email,
    t.ticket_type_id,
    t.ticket_type_name,
    t.event_id,
    t.event_title,
    t.session_id,
    t.session_starts_at,
    t.session_ends_at,
    co.paid_at,
    t.checked_in_at,
    t.checked_in_by_user_id,
    t.created_at,
    t.updated_at
FROM tickets t
JOIN checkout_orders co ON co.id = t.order_id
WHERE t.order_id = ?
ORDER BY t.session_starts_at ASC, t.ticket_type_name ASC, t.created_at ASC
`, orderID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tickets := make([]Ticket, 0)
	for rows.Next() {
		var (
			idText           string
			orderIDText      string
			ticketTypeIDText string
			eventIDText      string
			sessionIDText    string
			paidAt           sql.NullTime
			checkedInAt      sql.NullTime
			checkedInBy      sql.NullString
		)

		var ticket Ticket
		if err := rows.Scan(
			&idText,
			&ticket.Code,
			&orderIDText,
			&ticket.OrderNumber,
			&ticket.OrderStatus,
			&ticket.CustomerName,
			&ticket.CustomerEmail,
			&ticketTypeIDText,
			&ticket.TicketTypeName,
			&eventIDText,
			&ticket.EventTitle,
			&sessionIDText,
			&ticket.SessionStartsAt,
			&ticket.SessionEndsAt,
			&paidAt,
			&checkedInAt,
			&checkedInBy,
			&ticket.CreatedAt,
			&ticket.UpdatedAt,
		); err != nil {
			return nil, err
		}

		parsedID, err := uuid.Parse(idText)
		if err != nil {
			return nil, err
		}
		ticket.ID = parsedID

		parsedOrderID, err := uuid.Parse(orderIDText)
		if err != nil {
			return nil, err
		}
		ticket.OrderID = parsedOrderID

		parsedTicketTypeID, err := uuid.Parse(ticketTypeIDText)
		if err != nil {
			return nil, err
		}
		ticket.TicketTypeID = parsedTicketTypeID

		parsedEventID, err := uuid.Parse(eventIDText)
		if err != nil {
			return nil, err
		}
		ticket.EventID = parsedEventID

		parsedSessionID, err := uuid.Parse(sessionIDText)
		if err != nil {
			return nil, err
		}
		ticket.SessionID = parsedSessionID

		if paidAt.Valid {
			value := paidAt.Time
			ticket.PaidAt = &value
		}

		if checkedInAt.Valid {
			value := checkedInAt.Time
			ticket.CheckedInAt = &value
		}

		if checkedInBy.Valid && strings.TrimSpace(checkedInBy.String) != "" {
			value, err := uuid.Parse(checkedInBy.String)
			if err != nil {
				return nil, err
			}
			ticket.CheckedInBy = &value
		}

		tickets = append(tickets, ticket)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tickets, nil
}

func (r *Repository) GetByCode(ctx context.Context, claims *jwt.Claims, code string) (Ticket, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return Ticket{}, ErrInvalidTicketCode
	}

	var (
		ticketIDText      string
		orderIDText       string
		ticketTypeIDText  string
		eventIDText       string
		sessionIDText     string
		organizerIDText   string
		paidAt            sql.NullTime
		checkedInAt       sql.NullTime
		checkedInByUserID sql.NullString
	)

	var ticket Ticket
	if err := r.db.QueryRowContext(ctx, `
SELECT
    t.id,
    t.code,
    t.order_id,
    co.order_number,
    co.status,
    t.customer_name,
    t.customer_email,
    t.ticket_type_id,
    t.ticket_type_name,
    t.event_id,
    t.event_title,
    t.session_id,
    t.session_starts_at,
    t.session_ends_at,
    co.paid_at,
    t.checked_in_at,
    t.checked_in_by_user_id,
    e.organizer_id,
    t.created_at,
    t.updated_at
FROM tickets t
JOIN checkout_orders co ON co.id = t.order_id
JOIN events e ON e.id = t.event_id
WHERE t.code = ?
LIMIT 1
`, code).Scan(
		&ticketIDText,
		&ticket.Code,
		&orderIDText,
		&ticket.OrderNumber,
		&ticket.OrderStatus,
		&ticket.CustomerName,
		&ticket.CustomerEmail,
		&ticketTypeIDText,
		&ticket.TicketTypeName,
		&eventIDText,
		&ticket.EventTitle,
		&sessionIDText,
		&ticket.SessionStartsAt,
		&ticket.SessionEndsAt,
		&paidAt,
		&checkedInAt,
		&checkedInByUserID,
		&organizerIDText,
		&ticket.CreatedAt,
		&ticket.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Ticket{}, ErrTicketNotFound
		}
		return Ticket{}, err
	}

	parsedTicketID, err := uuid.Parse(ticketIDText)
	if err != nil {
		return Ticket{}, err
	}
	ticket.ID = parsedTicketID

	parsedOrderID, err := uuid.Parse(orderIDText)
	if err != nil {
		return Ticket{}, err
	}
	ticket.OrderID = parsedOrderID

	parsedTicketTypeID, err := uuid.Parse(ticketTypeIDText)
	if err != nil {
		return Ticket{}, err
	}
	ticket.TicketTypeID = parsedTicketTypeID

	parsedEventID, err := uuid.Parse(eventIDText)
	if err != nil {
		return Ticket{}, err
	}
	ticket.EventID = parsedEventID

	parsedSessionID, err := uuid.Parse(sessionIDText)
	if err != nil {
		return Ticket{}, err
	}
	ticket.SessionID = parsedSessionID

	var organizerID uuid.UUID
	if organizerIDText != "" {
		parsedOrganizerID, err := uuid.Parse(organizerIDText)
		if err != nil {
			return Ticket{}, err
		}
		organizerID = parsedOrganizerID
	}

	if claims.Role == roles.OrganizerAdmin {
		if claims.OrganizerID == nil {
			return Ticket{}, ErrOrganizerScopeMissing
		}

		if organizerID != *claims.OrganizerID {
			return Ticket{}, ErrTicketNotFound
		}
	} else if claims.Role != roles.SuperAdmin {
		return Ticket{}, ErrForbidden
	}

	if paidAt.Valid {
		value := paidAt.Time
		ticket.PaidAt = &value
	}

	if checkedInAt.Valid {
		value := checkedInAt.Time
		ticket.CheckedInAt = &value
	}

	if checkedInByUserID.Valid && strings.TrimSpace(checkedInByUserID.String) != "" {
		value, err := uuid.Parse(checkedInByUserID.String)
		if err != nil {
			return Ticket{}, err
		}
		ticket.CheckedInBy = &value
	}

	return ticket, nil
}

func (r *Repository) CheckInByCode(ctx context.Context, claims *jwt.Claims, code string, expectedEventID *uuid.UUID, expectedSessionID *uuid.UUID) (CheckInResult, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return CheckInResult{}, ErrInvalidTicketCode
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return CheckInResult{}, err
	}
	defer tx.Rollback()

	var (
		ticketIDText      string
		orderIDText       string
		ticketTypeIDText  string
		eventIDText       string
		sessionIDText     string
		organizerIDText   string
		paidAt            sql.NullTime
		checkedInAt       sql.NullTime
		checkedInByUserID sql.NullString
	)

	var ticket Ticket
	if err := tx.QueryRowContext(ctx, `
SELECT
    t.id,
    t.code,
    t.order_id,
    co.order_number,
    co.status,
    t.customer_name,
    t.customer_email,
    t.ticket_type_id,
    t.ticket_type_name,
    t.event_id,
    t.event_title,
    t.session_id,
    t.session_starts_at,
    t.session_ends_at,
    co.paid_at,
    t.checked_in_at,
    t.checked_in_by_user_id,
    e.organizer_id,
    t.created_at,
    t.updated_at
FROM tickets t
JOIN checkout_orders co ON co.id = t.order_id
JOIN events e ON e.id = t.event_id
WHERE t.code = ?
LIMIT 1
FOR UPDATE
`, code).Scan(
		&ticketIDText,
		&ticket.Code,
		&orderIDText,
		&ticket.OrderNumber,
		&ticket.OrderStatus,
		&ticket.CustomerName,
		&ticket.CustomerEmail,
		&ticketTypeIDText,
		&ticket.TicketTypeName,
		&eventIDText,
		&ticket.EventTitle,
		&sessionIDText,
		&ticket.SessionStartsAt,
		&ticket.SessionEndsAt,
		&paidAt,
		&checkedInAt,
		&checkedInByUserID,
		&organizerIDText,
		&ticket.CreatedAt,
		&ticket.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return CheckInResult{}, ErrTicketNotFound
		}
		return CheckInResult{}, err
	}

	parsedTicketID, err := uuid.Parse(ticketIDText)
	if err != nil {
		return CheckInResult{}, err
	}
	ticket.ID = parsedTicketID

	parsedOrderID, err := uuid.Parse(orderIDText)
	if err != nil {
		return CheckInResult{}, err
	}
	ticket.OrderID = parsedOrderID

	parsedTicketTypeID, err := uuid.Parse(ticketTypeIDText)
	if err != nil {
		return CheckInResult{}, err
	}
	ticket.TicketTypeID = parsedTicketTypeID

	parsedEventID, err := uuid.Parse(eventIDText)
	if err != nil {
		return CheckInResult{}, err
	}
	ticket.EventID = parsedEventID

	parsedSessionID, err := uuid.Parse(sessionIDText)
	if err != nil {
		return CheckInResult{}, err
	}
	ticket.SessionID = parsedSessionID

	var organizerID uuid.UUID
	if organizerIDText != "" {
		parsedOrganizerID, err := uuid.Parse(organizerIDText)
		if err != nil {
			return CheckInResult{}, err
		}
		organizerID = parsedOrganizerID
	}

	if claims.Role == roles.OrganizerAdmin {
		if claims.OrganizerID == nil {
			return CheckInResult{}, ErrOrganizerScopeMissing
		}

		if organizerID != *claims.OrganizerID {
			// Do not leak ticket existence outside of organizer scope.
			return CheckInResult{}, ErrTicketNotFound
		}
	} else if claims.Role != roles.SuperAdmin {
		return CheckInResult{}, ErrForbidden
	}

	if expectedEventID != nil && ticket.EventID != *expectedEventID {
		return CheckInResult{}, ErrTicketEventMismatch
	}

	if expectedSessionID != nil && ticket.SessionID != *expectedSessionID {
		return CheckInResult{}, ErrTicketSessionMismatch
	}

	if ticket.OrderStatus != "paid" {
		return CheckInResult{}, ErrTicketNotPaid
	}

	if paidAt.Valid {
		value := paidAt.Time
		ticket.PaidAt = &value
	}

	if checkedInAt.Valid {
		value := checkedInAt.Time
		ticket.CheckedInAt = &value
	}

	if checkedInByUserID.Valid && strings.TrimSpace(checkedInByUserID.String) != "" {
		value, err := uuid.Parse(checkedInByUserID.String)
		if err != nil {
			return CheckInResult{}, err
		}
		ticket.CheckedInBy = &value
	}

	if ticket.CheckedInAt != nil {
		return CheckInResult{
			Ticket:         ticket,
			AlreadyChecked: true,
		}, tx.Commit()
	}

	if _, err := tx.ExecContext(ctx, `
UPDATE tickets
SET checked_in_at = UTC_TIMESTAMP(),
    checked_in_by_user_id = ?
WHERE id = ?
  AND checked_in_at IS NULL
`, claims.UserID.String(), ticket.ID.String()); err != nil {
		return CheckInResult{}, err
	}

	var refreshedCheckedInAt sql.NullTime
	var refreshedCheckedInBy sql.NullString
	var refreshedUpdatedAt time.Time
	if err := tx.QueryRowContext(ctx, `
SELECT checked_in_at, checked_in_by_user_id, updated_at
FROM tickets
WHERE id = ?
LIMIT 1
`, ticket.ID.String()).Scan(&refreshedCheckedInAt, &refreshedCheckedInBy, &refreshedUpdatedAt); err != nil {
		return CheckInResult{}, err
	}

	if refreshedCheckedInAt.Valid {
		value := refreshedCheckedInAt.Time
		ticket.CheckedInAt = &value
	}

	if refreshedCheckedInBy.Valid && strings.TrimSpace(refreshedCheckedInBy.String) != "" {
		value, err := uuid.Parse(refreshedCheckedInBy.String)
		if err != nil {
			return CheckInResult{}, err
		}
		ticket.CheckedInBy = &value
	}

	ticket.UpdatedAt = refreshedUpdatedAt

	return CheckInResult{
		Ticket:         ticket,
		AlreadyChecked: false,
	}, tx.Commit()
}
