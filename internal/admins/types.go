package admins

import (
	"time"

	"github.com/google/uuid"
)

type CreateOrganizerAdminInput struct {
	OrganizerName string `json:"organizer_name"`
	OrganizerSlug string `json:"organizer_slug"`
	AdminName     string `json:"admin_name"`
	AdminEmail    string `json:"admin_email"`
	AdminPassword string `json:"admin_password"`
}

type UpdateOrganizerInput struct {
	OrganizerName string `json:"organizer_name"`
	OrganizerSlug string `json:"organizer_slug"`
}

type UpdateOrganizerAdminInput struct {
	AdminName  string `json:"admin_name"`
	AdminEmail string `json:"admin_email"`
}

type AddOrganizerAdminInput struct {
	AdminName     string `json:"admin_name"`
	AdminEmail    string `json:"admin_email"`
	AdminPassword string `json:"admin_password"`
}

type ResetOrganizerAdminPasswordInput struct {
	Password string `json:"password"`
}

type Organizer struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type OrganizerSummary struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Slug         string    `json:"slug"`
	AdminCount   int       `json:"admin_count"`
	EventCount   int       `json:"event_count"`
	SessionCount int       `json:"session_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type OrganizerListItem struct {
	ID           uuid.UUID       `json:"id"`
	Name         string          `json:"name"`
	Slug         string          `json:"slug"`
	AdminCount   int             `json:"admin_count"`
	EventCount   int             `json:"event_count"`
	SessionCount int             `json:"session_count"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	Admin        *OrganizerAdmin `json:"admin"`
}

type OrganizerDetail struct {
	Organizer OrganizerListItem       `json:"organizer"`
	Admins    []OrganizerAdmin        `json:"admins"`
	Events    []OrganizerManagedEvent `json:"events"`
}

type OrganizerManagedEvent struct {
	ID                uuid.UUID  `json:"id"`
	Title             string     `json:"title"`
	Slug              string     `json:"slug"`
	Status            string     `json:"status"`
	Currency          string     `json:"currency"`
	City              string     `json:"city"`
	Country           string     `json:"country"`
	SessionCount      int        `json:"session_count"`
	TicketTypeCount   int        `json:"ticket_type_count"`
	NextSessionStarts *time.Time `json:"next_session_starts_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type AdminOverview struct {
	Scope              string                          `json:"scope"`
	Stats              AdminOverviewStats              `json:"stats"`
	NeedsAttention     AdminOverviewNeedsAttention     `json:"needs_attention"`
	RecentEvents       []AdminOverviewEvent            `json:"recent_events"`
	UpcomingSessions   []AdminOverviewSession          `json:"upcoming_sessions"`
	OrganizerSummaries []AdminOverviewOrganizerSummary `json:"organizers"`
}

type AdminOverviewStats struct {
	Events            int `json:"events"`
	PublishedEvents   int `json:"published_events"`
	DraftEvents       int `json:"draft_events"`
	Sessions          int `json:"sessions"`
	ScheduledSessions int `json:"scheduled_sessions"`
	TicketTypes       int `json:"ticket_types"`
	Categories        int `json:"categories"`
	Organizers        int `json:"organizers"`
}

type AdminOverviewNeedsAttention struct {
	DraftEventsCount                int                             `json:"draft_events_count"`
	EventsWithoutSessionsCount      int                             `json:"events_without_sessions_count"`
	SessionsWithoutTicketTypesCount int                             `json:"sessions_without_ticket_types_count"`
	EventsWithoutSessions           []AdminOverviewAttentionEvent   `json:"events_without_sessions"`
	SessionsWithoutTicketTypes      []AdminOverviewAttentionSession `json:"sessions_without_ticket_types"`
}

type AdminOverviewEvent struct {
	ID            uuid.UUID `json:"id"`
	OrganizerID   uuid.UUID `json:"organizer_id"`
	CategoryID    uuid.UUID `json:"category_id"`
	Title         string    `json:"title"`
	Slug          string    `json:"slug"`
	Status        string    `json:"status"`
	VenueName     string    `json:"venue_name"`
	City          string    `json:"city"`
	Country       string    `json:"country"`
	OrganizerName string    `json:"organizer_name"`
	CategoryName  string    `json:"category_name"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type AdminOverviewSession struct {
	ID              uuid.UUID `json:"id"`
	EventID         uuid.UUID `json:"event_id"`
	EventTitle      string    `json:"event_title"`
	EventSlug       string    `json:"event_slug"`
	EventCurrency   string    `json:"event_currency"`
	Status          string    `json:"status"`
	StartsAt        time.Time `json:"starts_at"`
	TicketTypeCount int       `json:"ticket_type_count"`
}

type AdminOverviewAttentionEvent struct {
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"created_at"`
}

type AdminOverviewAttentionSession struct {
	ID         uuid.UUID `json:"id"`
	EventID    uuid.UUID `json:"event_id"`
	EventTitle string    `json:"event_title"`
	StartsAt   time.Time `json:"starts_at"`
	Status     string    `json:"status"`
}

type AdminOverviewOrganizerSummary struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Slug         string    `json:"slug"`
	EventCount   int       `json:"event_count"`
	SessionCount int       `json:"session_count"`
}

type AdminPayments struct {
	Scope   string               `json:"scope"`
	Summary AdminPaymentsSummary `json:"summary"`
	Trends  []AdminPaymentTrend  `json:"trends"`
	Items   []AdminPaymentItem   `json:"items"`
}

type AdminPaymentsSummary struct {
	Gross             float64 `json:"gross"`
	PaidOrders        int     `json:"paid_orders"`
	PendingOrders     int     `json:"pending_orders"`
	FailedOrExpired   int     `json:"failed_or_expired_orders"`
	AverageOrderValue float64 `json:"average_order_value"`
	Currency          string  `json:"currency"`
}

type AdminPaymentTrend struct {
	Day        string  `json:"day"`
	Gross      float64 `json:"gross"`
	PaidOrders int     `json:"paid_orders"`
}

type AdminPaymentItem struct {
	ID            uuid.UUID  `json:"id"`
	OrderNumber   string     `json:"order_number"`
	Status        string     `json:"status"`
	Amount        float64    `json:"amount"`
	Currency      string     `json:"currency"`
	CustomerName  string     `json:"customer_name"`
	CustomerEmail string     `json:"customer_email"`
	EventID       *uuid.UUID `json:"event_id,omitempty"`
	EventTitle    string     `json:"event_title,omitempty"`
	OrganizerID   *uuid.UUID `json:"organizer_id,omitempty"`
	OrganizerName string     `json:"organizer_name,omitempty"`
	PaidAt        *time.Time `json:"paid_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type AdminPaymentsExportFilters struct {
	FromDate    time.Time
	ToDate      time.Time
	OrganizerID *uuid.UUID
}

type AdminPaymentExportRow struct {
	OrderID       uuid.UUID
	OrderNumber   string
	Status        string
	Amount        float64
	Currency      string
	CustomerName  string
	CustomerEmail string
	EventTitles   string
	OrganizerID   *uuid.UUID
	OrganizerName string
	PaidAt        *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type OrganizerAdmin struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Email       string    `json:"email"`
	Role        string    `json:"role"`
	OrganizerID uuid.UUID `json:"organizer_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreateOrganizerAdminResult struct {
	Organizer Organizer      `json:"organizer"`
	Admin     OrganizerAdmin `json:"admin"`
}
