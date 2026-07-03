package events

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"eventy-api/internal/platform/email"
	"eventy-api/internal/platform/jwt"
	"eventy-api/internal/platform/roles"
	"eventy-api/internal/tickets"

	"github.com/google/uuid"
)

type ticketsMailer interface {
	SendTicketsIssued(
		toEmail string,
		toName string,
		orderNumber string,
		viewTicketsURL string,
		tickets []email.TicketIssuedItem,
		attachment *email.EmailAttachment,
	) error
}

type ticketsRepository interface {
	ListByOrderID(ctx context.Context, orderID uuid.UUID) ([]tickets.Ticket, error)
}

// pushNotifier delivers push notifications. It is satisfied by
// notifications.Dispatcher and is optional (nil when Firebase messaging is not
// configured). All methods are best-effort and must not affect business flow.
type pushNotifier interface {
	NotifyUserByEmail(ctx context.Context, email, title, body string, data map[string]string)
	NotifyEventAudience(ctx context.Context, eventID uuid.UUID, title, body string, data map[string]string)
	NotifyAllUsers(ctx context.Context, title, body string, data map[string]string)
}

type Service struct {
	repository *Repository
	stripeCfg  StripeConfig
	webBaseURL string
	mailer     ticketsMailer
	tickets    ticketsRepository
	notifier   pushNotifier
}

type StripeConfig struct {
	SecretKey     string
	WebhookSecret string
	SuccessURL    string
	CancelURL     string
}

func NewService(repository *Repository, stripeCfg StripeConfig, webBaseURL string, mailer ticketsMailer, ticketsRepo ticketsRepository) *Service {
	return &Service{
		repository: repository,
		stripeCfg:  stripeCfg,
		webBaseURL: strings.TrimRight(strings.TrimSpace(webBaseURL), "/"),
		mailer:     mailer,
		tickets:    ticketsRepo,
	}
}

// SetPushNotifier wires an optional push-notification dispatcher. Safe to leave
// unset (e.g. when Firebase messaging is not configured).
func (s *Service) SetPushNotifier(notifier pushNotifier) {
	s.notifier = notifier
}

// eventIsBookable reports whether an event is worth notifying about: it must be
// published and have at least one session and one ticket type. Best-effort — on
// a query error it returns false so we simply skip the notification.
func (s *Service) eventIsBookable(ctx context.Context, eventID uuid.UUID) bool {
	status, sessions, ticketTypes, err := s.repository.EventBookingReadiness(ctx, eventID)
	if err != nil {
		return false
	}
	return status == "published" && sessions > 0 && ticketTypes > 0
}

// announceEventChange centralizes the "available event" notification rule. When
// the event just became bookable it broadcasts to everyone; when it was already
// bookable (a genuine edit) it notifies only the event's ticket holders. Callers
// must only invoke this when the event is currently bookable.
func (s *Service) announceEventChange(
	ctx context.Context,
	eventID uuid.UUID,
	title string,
	audienceType string,
	audienceHeading string,
	audienceBody string,
	justBecameBookable bool,
) {
	if s.notifier == nil {
		return
	}

	if justBecameBookable {
		s.notifier.NotifyAllUsers(
			ctx,
			"New event on Eventy",
			fmt.Sprintf("%s is now live with tickets available. Book now!", title),
			map[string]string{"type": "event_available", "event_id": eventID.String()},
		)
		return
	}

	s.notifier.NotifyEventAudience(
		ctx,
		eventID,
		audienceHeading,
		audienceBody,
		map[string]string{"type": audienceType, "event_id": eventID.String()},
	)
}

func (s *Service) Create(ctx context.Context, claims *jwt.Claims, input CreateEventInput) (Event, error) {
	input.OrganizerID = strings.TrimSpace(input.OrganizerID)
	input.CategoryID = strings.TrimSpace(input.CategoryID)
	input.Title = strings.TrimSpace(input.Title)
	input.Slug = strings.TrimSpace(input.Slug)
	input.Description = strings.TrimSpace(input.Description)
	input.VenueName = strings.TrimSpace(input.VenueName)
	input.VenueAddress = strings.TrimSpace(input.VenueAddress)
	input.City = strings.TrimSpace(input.City)
	input.Country = strings.TrimSpace(input.Country)
	input.BannerURL = strings.TrimSpace(input.BannerURL)
	input.PosterURL = strings.TrimSpace(input.PosterURL)
	input.Status = strings.TrimSpace(input.Status)
	input.Currency = "EUR"

	if err := validateCreateEventInput(input); err != nil {
		return Event{}, err
	}

	organizerID, err := resolveOrganizerScopeForCreate(claims, input.OrganizerID)
	if err != nil {
		return Event{}, err
	}

	categoryID, err := uuid.Parse(input.CategoryID)
	if err != nil {
		return Event{}, ErrInvalidCategoryID
	}

	created, err := s.repository.Create(ctx, organizerID, categoryID, input)
	if err != nil {
		return Event{}, err
	}

	// No notification on create: a brand-new event has no sessions or ticket
	// types yet, so it is never bookable. The "available" broadcast fires later
	// (from AddTicketType / publish) once it actually has a session + ticket type.

	return created, nil
}

func (s *Service) Update(ctx context.Context, claims *jwt.Claims, eventID uuid.UUID, input UpdateEventInput) (Event, error) {
	input.CategoryID = strings.TrimSpace(input.CategoryID)
	input.Title = strings.TrimSpace(input.Title)
	input.Slug = strings.TrimSpace(input.Slug)
	input.Description = strings.TrimSpace(input.Description)
	input.VenueName = strings.TrimSpace(input.VenueName)
	input.VenueAddress = strings.TrimSpace(input.VenueAddress)
	input.City = strings.TrimSpace(input.City)
	input.Country = strings.TrimSpace(input.Country)
	input.BannerURL = strings.TrimSpace(input.BannerURL)
	input.PosterURL = strings.TrimSpace(input.PosterURL)
	input.Status = strings.TrimSpace(input.Status)
	input.Currency = "EUR"

	if err := validateUpdateEventInput(input); err != nil {
		return Event{}, err
	}

	event, err := s.getAccessibleEvent(ctx, claims, eventID)
	if err != nil {
		return Event{}, err
	}

	categoryID, err := uuid.Parse(input.CategoryID)
	if err != nil {
		return Event{}, ErrInvalidCategoryID
	}

	updated, err := s.repository.Update(ctx, event.ID, categoryID, input)
	if err != nil {
		return Event{}, err
	}

	if s.notifier != nil {
		if updated.Status == "cancelled" && event.Status != "cancelled" {
			// Cancellation always warns ticket holders, regardless of bookability.
			s.notifier.NotifyEventAudience(
				ctx,
				updated.ID,
				"Event cancelled",
				fmt.Sprintf("%s has been cancelled. Tap for details.", updated.Title),
				map[string]string{"type": "event_cancelled", "event_id": updated.ID.String()},
			)
		} else {
			// An event update leaves sessions/ticket types untouched, so the only
			// thing that can flip bookability is the status change.
			_, sessions, ticketTypes, readErr := s.repository.EventBookingReadiness(ctx, updated.ID)
			if readErr == nil {
				hasPieces := sessions > 0 && ticketTypes > 0
				bookableAfter := updated.Status == "published" && hasPieces
				bookableBefore := event.Status == "published" && hasPieces
				if bookableAfter {
					s.announceEventChange(
						ctx,
						updated.ID,
						updated.Title,
						"event_updated",
						"Event updated",
						fmt.Sprintf("%s has been updated. Check the latest details.", updated.Title),
						!bookableBefore,
					)
				}
			}
		}
	}

	return updated, nil
}

func (s *Service) Delete(ctx context.Context, claims *jwt.Claims, eventID uuid.UUID) error {
	if _, err := s.getAccessibleEvent(ctx, claims, eventID); err != nil {
		return err
	}

	return s.repository.Delete(ctx, eventID)
}

func (s *Service) List(ctx context.Context, claims *jwt.Claims) ([]EventListItem, error) {
	switch claims.Role {
	case roles.SuperAdmin:
		return s.repository.List(ctx)
	case roles.OrganizerAdmin:
		if claims.OrganizerID == nil {
			return nil, ErrOrganizerScopeRequired
		}

		return s.repository.ListByOrganizerID(ctx, *claims.OrganizerID)
	default:
		return nil, ErrUnsupportedRole
	}
}

func (s *Service) ListPublic(ctx context.Context) ([]PublicEvent, error) {
	return s.repository.ListPublic(ctx)
}

func (s *Service) GetPublicByID(ctx context.Context, eventID uuid.UUID) (PublicEventDetail, error) {
	return s.repository.GetPublicDetailByID(ctx, eventID)
}

func (s *Service) UpsertReservation(ctx context.Context, input UpsertTicketReservationInput) (TicketReservation, error) {
	input.ReservationID = strings.TrimSpace(input.ReservationID)
	input.ReservationToken = strings.TrimSpace(input.ReservationToken)

	for index := range input.Items {
		input.Items[index].TicketTypeID = strings.TrimSpace(input.Items[index].TicketTypeID)
	}

	return s.repository.UpsertReservation(ctx, input)
}

func (s *Service) GetReservation(ctx context.Context, reservationID uuid.UUID, reservationToken string) (TicketReservation, error) {
	reservationToken = strings.TrimSpace(reservationToken)
	if reservationToken == "" {
		return TicketReservation{}, ErrInvalidReservationToken
	}

	return s.repository.GetReservation(ctx, reservationID, reservationToken)
}

func (s *Service) DeleteReservation(ctx context.Context, reservationID uuid.UUID, reservationToken string) error {
	reservationToken = strings.TrimSpace(reservationToken)
	if reservationToken == "" {
		return ErrInvalidReservationToken
	}

	return s.repository.DeleteReservation(ctx, reservationID, reservationToken)
}

func (s *Service) CreateCheckoutOrder(ctx context.Context, input CreateCheckoutOrderInput) (CheckoutOrder, error) {
	input.ReservationID = strings.TrimSpace(input.ReservationID)
	input.ReservationToken = strings.TrimSpace(input.ReservationToken)
	input.CustomerName = strings.TrimSpace(input.CustomerName)
	input.CustomerEmail = strings.TrimSpace(strings.ToLower(input.CustomerEmail))

	if err := validateCreateCheckoutOrderInput(input); err != nil {
		return CheckoutOrder{}, err
	}

	return s.repository.CreateCheckoutOrder(ctx, input)
}

func (s *Service) GetCheckoutOrder(ctx context.Context, orderID uuid.UUID, orderToken string) (CheckoutOrder, error) {
	orderToken = strings.TrimSpace(orderToken)
	if orderToken == "" {
		return CheckoutOrder{}, ErrInvalidOrderToken
	}

	return s.repository.GetCheckoutOrder(ctx, orderID, orderToken)
}

func (s *Service) CreateStripeCheckoutSession(ctx context.Context, orderID uuid.UUID, input CreateStripeCheckoutSessionInput) (StripeCheckoutSessionResponse, error) {
	if strings.TrimSpace(s.stripeCfg.SecretKey) == "" || strings.TrimSpace(s.stripeCfg.SuccessURL) == "" || strings.TrimSpace(s.stripeCfg.CancelURL) == "" {
		return StripeCheckoutSessionResponse{}, ErrStripeNotConfigured
	}

	input.OrderToken = strings.TrimSpace(input.OrderToken)
	input.SuccessURL = strings.TrimSpace(input.SuccessURL)
	input.CancelURL = strings.TrimSpace(input.CancelURL)

	successURL := s.stripeCfg.SuccessURL
	if input.SuccessURL != "" {
		if err := validateCheckoutRedirectURL(input.SuccessURL); err != nil {
			return StripeCheckoutSessionResponse{}, err
		}
		successURL = input.SuccessURL
	}

	cancelURL := s.stripeCfg.CancelURL
	if input.CancelURL != "" {
		if err := validateCheckoutRedirectURL(input.CancelURL); err != nil {
			return StripeCheckoutSessionResponse{}, err
		}
		cancelURL = input.CancelURL
	}

	order, err := s.repository.GetCheckoutOrder(ctx, orderID, input.OrderToken)
	if err != nil {
		return StripeCheckoutSessionResponse{}, err
	}

	if order.Status != "pending_payment" {
		return StripeCheckoutSessionResponse{}, ErrCheckoutOrderNotPayable
	}

	session, err := s.repository.CreateStripeCheckoutSession(ctx, s.stripeCfg.SecretKey, successURL, cancelURL, order)
	if err != nil {
		return StripeCheckoutSessionResponse{}, err
	}

	return StripeCheckoutSessionResponse{
		SessionID:   session.ID,
		CheckoutURL: session.URL,
		OrderID:     order.ID.String(),
		OrderNumber: order.OrderNumber,
		ExpiresAt:   order.ExpiresAt.Format(time.RFC3339),
	}, nil
}

func validateCheckoutRedirectURL(rawURL string) error {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return ErrInvalidCheckoutRedirectURL
	}

	switch parsed.Scheme {
	case "http", "https", "eventy-mobile":
	default:
		return ErrInvalidCheckoutRedirectURL
	}

	if (parsed.Scheme == "http" || parsed.Scheme == "https") && strings.TrimSpace(parsed.Host) == "" {
		return ErrInvalidCheckoutRedirectURL
	}

	return nil
}

func (s *Service) GetCheckoutOrderByStripeSessionID(ctx context.Context, stripeSessionID string) (CheckoutOrderSummary, error) {
	stripeSessionID = strings.TrimSpace(stripeSessionID)
	if stripeSessionID == "" {
		return CheckoutOrderSummary{}, ErrInvalidStripeSessionID
	}

	return s.repository.GetCheckoutOrderByStripeSessionID(ctx, stripeSessionID)
}

func (s *Service) GetCheckoutOrderSummaryByID(ctx context.Context, orderID uuid.UUID) (CheckoutOrderSummary, error) {
	return s.repository.GetCheckoutOrderSummaryByID(ctx, orderID)
}

func (s *Service) ListCheckoutOrdersByCustomerEmail(ctx context.Context, customerEmail string, limit int) ([]CheckoutOrderSummary, error) {
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

	return s.repository.ListCheckoutOrdersByCustomerEmail(ctx, customerEmail, limit)
}

func (s *Service) HandleStripeWebhook(ctx context.Context, payload []byte, signature string) error {
	if strings.TrimSpace(s.stripeCfg.WebhookSecret) == "" {
		return ErrStripeNotConfigured
	}

	outcome, err := s.repository.HandleStripeWebhook(ctx, s.stripeCfg.SecretKey, s.stripeCfg.WebhookSecret, payload, signature)
	if err != nil {
		return err
	}

	if !outcome.ShouldEmailTickets {
		return nil
	}

	if s.mailer == nil || s.tickets == nil || strings.TrimSpace(s.webBaseURL) == "" {
		return errors.New("ticket email is not configured")
	}

	order, err := s.repository.GetCheckoutOrderByStripeSessionID(ctx, outcome.StripeSessionID)
	if err != nil {
		return err
	}

	if order.Status != "paid" {
		return nil
	}

	if order.TicketsEmailedAt != nil {
		return nil
	}

	issued, err := s.tickets.ListByOrderID(ctx, order.ID)
	if err != nil {
		return err
	}
	if len(issued) == 0 {
		return fmt.Errorf("paid order has no tickets: order_id=%s", order.ID.String())
	}

	viewURL := s.webBaseURL + "/tickets"
	items := make([]email.TicketIssuedItem, 0, len(issued))
	for _, ticket := range issued {
		items = append(items, email.TicketIssuedItem{
			Code:            ticket.Code,
			QRPayload:       fmt.Sprintf("eventy:ticket?code=%s&event_id=%s&session_id=%s", ticket.Code, ticket.EventID.String(), ticket.SessionID.String()),
			EventTitle:      ticket.EventTitle,
			TicketTypeName:  ticket.TicketTypeName,
			SessionStartsAt: ticket.SessionStartsAt,
			SessionEndsAt:   ticket.SessionEndsAt,
		})
	}

	receiptItems := make([]email.ReceiptPDFItem, 0, len(order.Items))
	for _, item := range order.Items {
		receiptItems = append(receiptItems, email.ReceiptPDFItem{
			TicketTypeName:  item.TicketTypeName,
			EventTitle:      item.EventTitle,
			Quantity:        item.Quantity,
			UnitPrice:       item.UnitPrice,
			Currency:        item.Currency,
			SessionStartsAt: item.SessionStartsAt,
			SessionEndsAt:   item.SessionEndsAt,
		})
	}

	receiptData, err := email.BuildReceiptPDF(email.ReceiptPDFInput{
		OrderNumber:   order.OrderNumber,
		Status:        order.Status,
		CustomerName:  order.CustomerName,
		CustomerEmail: order.CustomerEmail,
		Currency:      order.Currency,
		Subtotal:      order.Subtotal,
		CreatedAt:     order.CreatedAt,
		UpdatedAt:     order.UpdatedAt,
		PaidAt:        order.PaidAt,
		Items:         receiptItems,
	})
	if err != nil {
		return err
	}

	attachment := &email.EmailAttachment{
		Filename:    fmt.Sprintf("receipt-%s.pdf", strings.TrimSpace(order.OrderNumber)),
		ContentType: "application/pdf",
		Data:        receiptData,
	}

	if err := s.mailer.SendTicketsIssued(order.CustomerEmail, order.CustomerName, order.OrderNumber, viewURL, items, attachment); err != nil {
		return err
	}

	if err := s.repository.MarkCheckoutOrderTicketsEmailed(ctx, order.ID); err != nil {
		return err
	}

	if s.notifier != nil {
		s.notifier.NotifyUserByEmail(
			ctx,
			order.CustomerEmail,
			"Payment confirmed",
			fmt.Sprintf("Your tickets for order %s are ready.", order.OrderNumber),
			map[string]string{"type": "order_paid", "order_id": order.ID.String()},
		)
	}

	return nil
}

func (s *Service) GetByID(ctx context.Context, claims *jwt.Claims, eventID uuid.UUID) (Event, error) {
	event, err := s.repository.GetByID(ctx, eventID)
	if err != nil {
		return Event{}, err
	}

	if !canAccessEvent(claims, event) {
		if claims.Role == roles.OrganizerAdmin && claims.OrganizerID == nil {
			return Event{}, ErrOrganizerScopeRequired
		}

		return Event{}, ErrForbidden
	}

	return event, nil
}

func (s *Service) GetDetailByID(ctx context.Context, claims *jwt.Claims, eventID uuid.UUID) (EventDetail, error) {
	_, err := s.getAccessibleEvent(ctx, claims, eventID)
	if err != nil {
		return EventDetail{}, err
	}

	return s.repository.GetDetailByID(ctx, eventID)
}

func (s *Service) CreateSession(ctx context.Context, claims *jwt.Claims, eventID uuid.UUID, input CreateEventSessionInput) (EventSession, error) {
	input.Status = strings.TrimSpace(input.Status)

	if err := validateEventSessionInput(input.StartsAt, input.EndsAt, input.SalesStartsAt, input.SalesEndsAt, input.Status); err != nil {
		return EventSession{}, err
	}

	event, err := s.getAccessibleEvent(ctx, claims, eventID)
	if err != nil {
		return EventSession{}, err
	}

	created, err := s.repository.CreateSession(ctx, eventID, input)
	if err != nil {
		return EventSession{}, err
	}

	// A new session carries no ticket types, so it can never make an event
	// bookable on its own — only notify ticket holders of an already-bookable
	// event that a new date was added.
	if s.eventIsBookable(ctx, eventID) {
		s.announceEventChange(
			ctx,
			eventID,
			event.Title,
			"session_added",
			"New date added",
			fmt.Sprintf("%s — a new date was added.", event.Title),
			false,
		)
	}

	return created, nil
}

func (s *Service) ListSessions(ctx context.Context, claims *jwt.Claims, eventID uuid.UUID) ([]EventSession, error) {
	if _, err := s.getAccessibleEvent(ctx, claims, eventID); err != nil {
		return nil, err
	}

	return s.repository.ListSessionsByEventID(ctx, eventID)
}

func (s *Service) UpdateSession(ctx context.Context, claims *jwt.Claims, eventID uuid.UUID, sessionID uuid.UUID, input UpdateEventSessionInput) (EventSession, error) {
	input.Status = strings.TrimSpace(input.Status)

	if err := validateEventSessionInput(input.StartsAt, input.EndsAt, input.SalesStartsAt, input.SalesEndsAt, input.Status); err != nil {
		return EventSession{}, err
	}

	event, err := s.getAccessibleEvent(ctx, claims, eventID)
	if err != nil {
		return EventSession{}, err
	}

	session, err := s.repository.GetSessionByID(ctx, sessionID)
	if err != nil {
		return EventSession{}, err
	}

	if session.EventID != eventID {
		return EventSession{}, ErrEventSessionNotFound
	}

	updated, err := s.repository.UpdateSession(ctx, sessionID, input)
	if err != nil {
		return EventSession{}, err
	}

	if s.eventIsBookable(ctx, eventID) {
		s.announceEventChange(
			ctx,
			eventID,
			event.Title,
			"session_updated",
			"Session updated",
			fmt.Sprintf("%s — a session was updated.", event.Title),
			false,
		)
	}

	return updated, nil
}

func (s *Service) DeleteSession(ctx context.Context, claims *jwt.Claims, eventID uuid.UUID, sessionID uuid.UUID) error {
	if _, err := s.getAccessibleEvent(ctx, claims, eventID); err != nil {
		return err
	}

	session, err := s.repository.GetSessionByID(ctx, sessionID)
	if err != nil {
		return err
	}

	if session.EventID != eventID {
		return ErrEventSessionNotFound
	}

	return s.repository.DeleteSession(ctx, sessionID)
}

func (s *Service) CreateTicketType(ctx context.Context, claims *jwt.Claims, eventID uuid.UUID, sessionID uuid.UUID, input CreateTicketTypeInput) (TicketType, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Description = strings.TrimSpace(input.Description)

	if err := validateTicketTypeInput(input.Name, input.Price, input.Quantity, input.MaxPerOrder); err != nil {
		return TicketType{}, err
	}

	event, err := s.getAccessibleEvent(ctx, claims, eventID)
	if err != nil {
		return TicketType{}, err
	}

	session, err := s.repository.GetSessionByID(ctx, sessionID)
	if err != nil {
		return TicketType{}, err
	}

	if session.EventID != eventID {
		return TicketType{}, ErrEventSessionNotFound
	}

	created, err := s.repository.CreateTicketType(ctx, sessionID, input)
	if err != nil {
		return TicketType{}, err
	}

	// Adding the first ticket type to a published event with a session is the
	// moment it becomes bookable → broadcast. Adding another to an already
	// bookable event → notify its ticket holders.
	if _, sessions, ticketTypes, readErr := s.repository.EventBookingReadiness(ctx, eventID); readErr == nil {
		if event.Status == "published" && sessions > 0 && ticketTypes > 0 {
			s.announceEventChange(
				ctx,
				eventID,
				event.Title,
				"ticket_added",
				"New tickets available",
				fmt.Sprintf("%s — new tickets are on sale: %s.", event.Title, created.Name),
				ticketTypes == 1,
			)
		}
	}

	return created, nil
}

func (s *Service) ListTicketTypes(ctx context.Context, claims *jwt.Claims, eventID uuid.UUID, sessionID uuid.UUID) ([]TicketType, error) {
	if _, err := s.getAccessibleEvent(ctx, claims, eventID); err != nil {
		return nil, err
	}

	session, err := s.repository.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	if session.EventID != eventID {
		return nil, ErrEventSessionNotFound
	}

	return s.repository.ListTicketTypesBySessionID(ctx, sessionID)
}

func (s *Service) UpdateTicketType(ctx context.Context, claims *jwt.Claims, eventID uuid.UUID, sessionID uuid.UUID, ticketTypeID uuid.UUID, input UpdateTicketTypeInput) (TicketType, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Description = strings.TrimSpace(input.Description)

	if err := validateTicketTypeInput(input.Name, input.Price, input.Quantity, input.MaxPerOrder); err != nil {
		return TicketType{}, err
	}

	event, err := s.getAccessibleEvent(ctx, claims, eventID)
	if err != nil {
		return TicketType{}, err
	}

	session, err := s.repository.GetSessionByID(ctx, sessionID)
	if err != nil {
		return TicketType{}, err
	}

	if session.EventID != eventID {
		return TicketType{}, ErrEventSessionNotFound
	}

	ticketType, err := s.repository.GetTicketTypeByID(ctx, ticketTypeID)
	if err != nil {
		return TicketType{}, err
	}

	if ticketType.EventSessionID != sessionID {
		return TicketType{}, ErrTicketTypeNotFound
	}

	updated, err := s.repository.UpdateTicketType(ctx, ticketTypeID, input)
	if err != nil {
		return TicketType{}, err
	}

	// Updating an existing ticket type never changes bookability, so this only
	// notifies ticket holders — and only while the event is bookable.
	if s.eventIsBookable(ctx, eventID) {
		if updated.Price != ticketType.Price {
			s.announceEventChange(
				ctx,
				eventID,
				event.Title,
				"price_changed",
				"Ticket price updated",
				fmt.Sprintf("%s — %s is now %.2f %s.", event.Title, updated.Name, updated.Price, event.Currency),
				false,
			)
		} else {
			s.announceEventChange(
				ctx,
				eventID,
				event.Title,
				"ticket_updated",
				"Tickets updated",
				fmt.Sprintf("%s — ticket \"%s\" was updated.", event.Title, updated.Name),
				false,
			)
		}
	}

	return updated, nil
}

func (s *Service) DeleteTicketType(ctx context.Context, claims *jwt.Claims, eventID uuid.UUID, sessionID uuid.UUID, ticketTypeID uuid.UUID) error {
	if _, err := s.getAccessibleEvent(ctx, claims, eventID); err != nil {
		return err
	}

	session, err := s.repository.GetSessionByID(ctx, sessionID)
	if err != nil {
		return err
	}

	if session.EventID != eventID {
		return ErrEventSessionNotFound
	}

	ticketType, err := s.repository.GetTicketTypeByID(ctx, ticketTypeID)
	if err != nil {
		return err
	}

	if ticketType.EventSessionID != sessionID {
		return ErrTicketTypeNotFound
	}

	return s.repository.DeleteTicketType(ctx, ticketTypeID)
}

func (s *Service) getAccessibleEvent(ctx context.Context, claims *jwt.Claims, eventID uuid.UUID) (Event, error) {
	event, err := s.repository.GetByID(ctx, eventID)
	if err != nil {
		return Event{}, err
	}

	if !canAccessEvent(claims, event) {
		if claims.Role == roles.OrganizerAdmin && claims.OrganizerID == nil {
			return Event{}, ErrOrganizerScopeRequired
		}

		return Event{}, ErrForbidden
	}

	return event, nil
}

func resolveOrganizerScopeForCreate(claims *jwt.Claims, requestedOrganizerID string) (uuid.UUID, error) {
	switch claims.Role {
	case roles.SuperAdmin:
		if requestedOrganizerID == "" {
			return uuid.UUID{}, ErrInvalidOrganizerID
		}

		organizerID, err := uuid.Parse(requestedOrganizerID)
		if err != nil {
			return uuid.UUID{}, ErrInvalidOrganizerID
		}

		return organizerID, nil
	case roles.OrganizerAdmin:
		if claims.OrganizerID == nil {
			return uuid.UUID{}, ErrOrganizerScopeRequired
		}

		if requestedOrganizerID != "" && requestedOrganizerID != claims.OrganizerID.String() {
			return uuid.UUID{}, ErrForbidden
		}

		return *claims.OrganizerID, nil
	default:
		return uuid.UUID{}, ErrUnsupportedRole
	}
}

func canAccessEvent(claims *jwt.Claims, event Event) bool {
	switch claims.Role {
	case roles.SuperAdmin:
		return true
	case roles.OrganizerAdmin:
		return claims.OrganizerID != nil && *claims.OrganizerID == event.OrganizerID
	default:
		return false
	}
}
