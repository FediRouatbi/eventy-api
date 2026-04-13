package tickets

import (
	"time"

	"github.com/google/uuid"
)

type Ticket struct {
	ID              uuid.UUID  `json:"id"`
	Code            string     `json:"code"`
	OrderID         uuid.UUID  `json:"order_id"`
	OrderNumber     string     `json:"order_number"`
	OrderStatus     string     `json:"order_status"`
	CustomerName    string     `json:"customer_name"`
	CustomerEmail   string     `json:"customer_email"`
	TicketTypeID    uuid.UUID  `json:"ticket_type_id"`
	TicketTypeName  string     `json:"ticket_type_name"`
	EventID         uuid.UUID  `json:"event_id"`
	EventTitle      string     `json:"event_title"`
	SessionID       uuid.UUID  `json:"session_id"`
	SessionStartsAt time.Time  `json:"session_starts_at"`
	SessionEndsAt   time.Time  `json:"session_ends_at"`
	PaidAt          *time.Time `json:"paid_at,omitempty"`
	CheckedInAt     *time.Time `json:"checked_in_at,omitempty"`
	CheckedInBy     *uuid.UUID `json:"checked_in_by_user_id,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type CheckInResult struct {
	Ticket         Ticket `json:"ticket"`
	AlreadyChecked bool   `json:"already_checked"`
}
