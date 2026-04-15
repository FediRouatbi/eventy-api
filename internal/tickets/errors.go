package tickets

import "errors"

var (
	ErrInvalidTicketCode     = errors.New("invalid ticket code")
	ErrInvalidEventID        = errors.New("event_id must be a valid uuid")
	ErrInvalidSessionID      = errors.New("session_id must be a valid uuid")
	ErrTicketNotFound        = errors.New("ticket not found")
	ErrTicketNotPaid         = errors.New("ticket is not paid")
	ErrTicketEventMismatch   = errors.New("ticket does not belong to this event")
	ErrTicketSessionMismatch = errors.New("ticket does not belong to this session")
	ErrOrganizerScopeMissing = errors.New("organizer scope required")
	ErrForbidden             = errors.New("forbidden")
)
