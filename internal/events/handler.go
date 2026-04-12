package events

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	httpmiddleware "eventy-api/internal/http/middleware"
	"eventy-api/internal/http/responses"
	"eventy-api/internal/platform/logger"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	claims, ok := httpmiddleware.ClaimsFromContext(r.Context())
	if !ok {
		responses.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var input CreateEventInput
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&input); err != nil {
		logger.RequestError(r, "events.create.decode", err)
		responses.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	event, err := h.service.Create(r.Context(), claims, input)
	if err != nil {
		logger.RequestError(r, "events.create", err)

		switch {
		case errors.Is(err, ErrInvalidOrganizerID),
			errors.Is(err, ErrInvalidCategoryID),
			errors.Is(err, ErrInvalidTitle),
			errors.Is(err, ErrInvalidSlug),
			errors.Is(err, ErrInvalidDescription),
			errors.Is(err, ErrInvalidVenueName),
			errors.Is(err, ErrInvalidVenueAddress),
			errors.Is(err, ErrInvalidCity),
			errors.Is(err, ErrInvalidCountry),
			errors.Is(err, ErrInvalidLatitude),
			errors.Is(err, ErrInvalidLongitude),
			errors.Is(err, ErrInvalidCoordinates),
			errors.Is(err, ErrInvalidCurrency),
			errors.Is(err, ErrInvalidStatus):
			responses.WriteError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrForbidden), errors.Is(err, ErrUnsupportedRole), errors.Is(err, ErrOrganizerScopeRequired):
			responses.WriteError(w, http.StatusForbidden, err.Error())
		case errors.Is(err, ErrEventSlugAlreadyExists):
			responses.WriteError(w, http.StatusConflict, err.Error())
		case errors.Is(err, ErrCategoryNotFound), errors.Is(err, ErrOrganizerNotFound):
			responses.WriteError(w, http.StatusBadRequest, err.Error())
		default:
			responses.WriteError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	responses.WriteJSON(w, http.StatusCreated, event)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	claims, ok := httpmiddleware.ClaimsFromContext(r.Context())
	if !ok {
		responses.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	eventID, err := uuid.Parse(chi.URLParam(r, "eventID"))
	if err != nil {
		responses.WriteError(w, http.StatusBadRequest, "invalid event id")
		return
	}

	var input UpdateEventInput
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&input); err != nil {
		logger.RequestError(r, "events.update.decode", err)
		responses.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	event, err := h.service.Update(r.Context(), claims, eventID, input)
	if err != nil {
		logger.RequestError(r, "events.update", err)

		switch {
		case errors.Is(err, ErrInvalidCategoryID),
			errors.Is(err, ErrInvalidTitle),
			errors.Is(err, ErrInvalidSlug),
			errors.Is(err, ErrInvalidDescription),
			errors.Is(err, ErrInvalidVenueName),
			errors.Is(err, ErrInvalidVenueAddress),
			errors.Is(err, ErrInvalidCity),
			errors.Is(err, ErrInvalidCountry),
			errors.Is(err, ErrInvalidLatitude),
			errors.Is(err, ErrInvalidLongitude),
			errors.Is(err, ErrInvalidCoordinates),
			errors.Is(err, ErrInvalidCurrency),
			errors.Is(err, ErrInvalidStatus):
			responses.WriteError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrForbidden), errors.Is(err, ErrUnsupportedRole), errors.Is(err, ErrOrganizerScopeRequired):
			responses.WriteError(w, http.StatusForbidden, err.Error())
		case errors.Is(err, ErrEventSlugAlreadyExists):
			responses.WriteError(w, http.StatusConflict, err.Error())
		case errors.Is(err, ErrCategoryNotFound):
			responses.WriteError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrEventNotFound):
			responses.WriteError(w, http.StatusNotFound, err.Error())
		default:
			responses.WriteError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	responses.WriteJSON(w, http.StatusOK, event)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	claims, ok := httpmiddleware.ClaimsFromContext(r.Context())
	if !ok {
		responses.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	eventID, err := uuid.Parse(chi.URLParam(r, "eventID"))
	if err != nil {
		responses.WriteError(w, http.StatusBadRequest, "invalid event id")
		return
	}

	if err := h.service.Delete(r.Context(), claims, eventID); err != nil {
		logger.RequestError(r, "events.delete", err)

		switch {
		case errors.Is(err, ErrEventNotFound):
			responses.WriteError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, ErrForbidden), errors.Is(err, ErrUnsupportedRole), errors.Is(err, ErrOrganizerScopeRequired):
			responses.WriteError(w, http.StatusForbidden, err.Error())
		default:
			responses.WriteError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	claims, ok := httpmiddleware.ClaimsFromContext(r.Context())
	if !ok {
		responses.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	events, err := h.service.List(r.Context(), claims)
	if err != nil {
		logger.RequestError(r, "events.list", err)

		switch {
		case errors.Is(err, ErrForbidden), errors.Is(err, ErrUnsupportedRole), errors.Is(err, ErrOrganizerScopeRequired):
			responses.WriteError(w, http.StatusForbidden, err.Error())
		default:
			responses.WriteError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	responses.WriteJSON(w, http.StatusOK, events)
}

func (h *Handler) ListPublic(w http.ResponseWriter, r *http.Request) {
	events, err := h.service.ListPublic(r.Context())
	if err != nil {
		logger.RequestError(r, "events.list_public", err)
		responses.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	responses.WriteJSON(w, http.StatusOK, events)
}

func (h *Handler) GetPublicByID(w http.ResponseWriter, r *http.Request) {
	eventID, err := uuid.Parse(chi.URLParam(r, "eventID"))
	if err != nil {
		responses.WriteError(w, http.StatusBadRequest, "invalid event id")
		return
	}

	event, err := h.service.GetPublicByID(r.Context(), eventID)
	if err != nil {
		logger.RequestError(r, "events.get_public_by_id", err)

		switch {
		case errors.Is(err, ErrEventNotFound):
			responses.WriteError(w, http.StatusNotFound, err.Error())
		default:
			responses.WriteError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	responses.WriteJSON(w, http.StatusOK, event)
}

func (h *Handler) UpsertReservation(w http.ResponseWriter, r *http.Request) {
	var input UpsertTicketReservationInput
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&input); err != nil {
		logger.RequestError(r, "events.upsert_reservation.decode", err)
		responses.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	reservation, err := h.service.UpsertReservation(r.Context(), input)
	if err != nil {
		h.writeReservationError(w, r, "events.upsert_reservation", err)
		return
	}

	responses.WriteJSON(w, http.StatusOK, reservation)
}

func (h *Handler) GetReservation(w http.ResponseWriter, r *http.Request) {
	reservationID, err := uuid.Parse(chi.URLParam(r, "reservationID"))
	if err != nil {
		responses.WriteError(w, http.StatusBadRequest, ErrInvalidReservationID.Error())
		return
	}

	reservation, err := h.service.GetReservation(r.Context(), reservationID, r.URL.Query().Get("token"))
	if err != nil {
		h.writeReservationError(w, r, "events.get_reservation", err)
		return
	}

	responses.WriteJSON(w, http.StatusOK, reservation)
}

func (h *Handler) DeleteReservation(w http.ResponseWriter, r *http.Request) {
	reservationID, err := uuid.Parse(chi.URLParam(r, "reservationID"))
	if err != nil {
		responses.WriteError(w, http.StatusBadRequest, ErrInvalidReservationID.Error())
		return
	}

	if err := h.service.DeleteReservation(r.Context(), reservationID, r.URL.Query().Get("token")); err != nil {
		h.writeReservationError(w, r, "events.delete_reservation", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) CreateCheckoutOrder(w http.ResponseWriter, r *http.Request) {
	var input CreateCheckoutOrderInput
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&input); err != nil {
		logger.RequestError(r, "events.create_checkout_order.decode", err)
		responses.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	order, err := h.service.CreateCheckoutOrder(r.Context(), input)
	if err != nil {
		h.writeCheckoutOrderError(w, r, "events.create_checkout_order", err)
		return
	}

	responses.WriteJSON(w, http.StatusCreated, order)
}

func (h *Handler) GetCheckoutOrder(w http.ResponseWriter, r *http.Request) {
	orderID, err := uuid.Parse(chi.URLParam(r, "orderID"))
	if err != nil {
		responses.WriteError(w, http.StatusBadRequest, "invalid order id")
		return
	}

	order, err := h.service.GetCheckoutOrder(r.Context(), orderID, r.URL.Query().Get("token"))
	if err != nil {
		h.writeCheckoutOrderError(w, r, "events.get_checkout_order", err)
		return
	}

	responses.WriteJSON(w, http.StatusOK, order)
}

func (h *Handler) CreateStripeCheckoutSession(w http.ResponseWriter, r *http.Request) {
	orderID, err := uuid.Parse(chi.URLParam(r, "orderID"))
	if err != nil {
		responses.WriteError(w, http.StatusBadRequest, "invalid order id")
		return
	}

	var input CreateStripeCheckoutSessionInput
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&input); err != nil {
		logger.RequestError(r, "events.create_stripe_checkout_session.decode", err)
		responses.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	input.OrderToken = strings.TrimSpace(input.OrderToken)
	if input.OrderToken == "" {
		responses.WriteError(w, http.StatusBadRequest, ErrInvalidOrderToken.Error())
		return
	}

	session, err := h.service.CreateStripeCheckoutSession(r.Context(), orderID, input.OrderToken)
	if err != nil {
		logger.RequestError(r, "events.create_stripe_checkout_session", err)

		switch {
		case errors.Is(err, ErrInvalidOrderToken), errors.Is(err, ErrCheckoutOrderNotFound):
			responses.WriteError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, ErrUnsupportedCurrency):
			responses.WriteError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrStripeNotConfigured):
			responses.WriteError(w, http.StatusNotImplemented, err.Error())
		case errors.Is(err, ErrCheckoutOrderNotPayable):
			responses.WriteError(w, http.StatusConflict, err.Error())
		default:
			responses.WriteError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	responses.WriteJSON(w, http.StatusCreated, session)
}

func (h *Handler) GetCheckoutOrderByStripeSession(w http.ResponseWriter, r *http.Request) {
	stripeSessionID := chi.URLParam(r, "stripeSessionID")
	order, err := h.service.GetCheckoutOrderByStripeSessionID(r.Context(), stripeSessionID)
	if err != nil {
		logger.RequestError(r, "events.get_checkout_order_by_stripe_session", err)

		switch {
		case errors.Is(err, ErrInvalidStripeSessionID):
			responses.WriteError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrCheckoutOrderNotFound):
			responses.WriteError(w, http.StatusNotFound, err.Error())
		default:
			responses.WriteError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	responses.WriteJSON(w, http.StatusOK, order)
}

func (h *Handler) StripeWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		logger.RequestError(r, "events.stripe_webhook.read", err)
		responses.WriteError(w, http.StatusBadRequest, "invalid payload")
		return
	}

	signature := r.Header.Get("Stripe-Signature")
	if signature == "" {
		responses.WriteError(w, http.StatusBadRequest, "missing stripe signature")
		return
	}

	if err := h.service.HandleStripeWebhook(r.Context(), body, signature); err != nil {
		logger.RequestError(r, "events.stripe_webhook", err)

		switch {
		case errors.Is(err, ErrStripeNotConfigured):
			responses.WriteError(w, http.StatusNotImplemented, err.Error())
		default:
			responses.WriteError(w, http.StatusBadRequest, "webhook error")
		}
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	claims, ok := httpmiddleware.ClaimsFromContext(r.Context())
	if !ok {
		responses.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	eventID, err := uuid.Parse(chi.URLParam(r, "eventID"))
	if err != nil {
		responses.WriteError(w, http.StatusBadRequest, "invalid event id")
		return
	}

	event, err := h.service.GetDetailByID(r.Context(), claims, eventID)
	if err != nil {
		logger.RequestError(r, "events.get_by_id", err)

		switch {
		case errors.Is(err, ErrEventNotFound):
			responses.WriteError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, ErrForbidden), errors.Is(err, ErrUnsupportedRole), errors.Is(err, ErrOrganizerScopeRequired):
			responses.WriteError(w, http.StatusForbidden, err.Error())
		default:
			responses.WriteError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	responses.WriteJSON(w, http.StatusOK, event)
}

func (h *Handler) CreateSession(w http.ResponseWriter, r *http.Request) {
	claims, ok := httpmiddleware.ClaimsFromContext(r.Context())
	if !ok {
		responses.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	eventID, err := uuid.Parse(chi.URLParam(r, "eventID"))
	if err != nil {
		responses.WriteError(w, http.StatusBadRequest, ErrInvalidEventID.Error())
		return
	}

	var input CreateEventSessionInput
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&input); err != nil {
		logger.RequestError(r, "events.create_session.decode", err)
		responses.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	session, err := h.service.CreateSession(r.Context(), claims, eventID, input)
	if err != nil {
		h.writeSessionError(w, r, "events.create_session", err)
		return
	}

	responses.WriteJSON(w, http.StatusCreated, session)
}

func (h *Handler) ListSessions(w http.ResponseWriter, r *http.Request) {
	claims, ok := httpmiddleware.ClaimsFromContext(r.Context())
	if !ok {
		responses.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	eventID, err := uuid.Parse(chi.URLParam(r, "eventID"))
	if err != nil {
		responses.WriteError(w, http.StatusBadRequest, ErrInvalidEventID.Error())
		return
	}

	sessions, err := h.service.ListSessions(r.Context(), claims, eventID)
	if err != nil {
		h.writeSessionError(w, r, "events.list_sessions", err)
		return
	}

	responses.WriteJSON(w, http.StatusOK, sessions)
}

func (h *Handler) UpdateSession(w http.ResponseWriter, r *http.Request) {
	claims, ok := httpmiddleware.ClaimsFromContext(r.Context())
	if !ok {
		responses.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	eventID, err := uuid.Parse(chi.URLParam(r, "eventID"))
	if err != nil {
		responses.WriteError(w, http.StatusBadRequest, ErrInvalidEventID.Error())
		return
	}

	sessionID, err := uuid.Parse(chi.URLParam(r, "sessionID"))
	if err != nil {
		responses.WriteError(w, http.StatusBadRequest, ErrInvalidSessionID.Error())
		return
	}

	var input UpdateEventSessionInput
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&input); err != nil {
		logger.RequestError(r, "events.update_session.decode", err)
		responses.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	session, err := h.service.UpdateSession(r.Context(), claims, eventID, sessionID, input)
	if err != nil {
		h.writeSessionError(w, r, "events.update_session", err)
		return
	}

	responses.WriteJSON(w, http.StatusOK, session)
}

func (h *Handler) DeleteSession(w http.ResponseWriter, r *http.Request) {
	claims, ok := httpmiddleware.ClaimsFromContext(r.Context())
	if !ok {
		responses.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	eventID, err := uuid.Parse(chi.URLParam(r, "eventID"))
	if err != nil {
		responses.WriteError(w, http.StatusBadRequest, ErrInvalidEventID.Error())
		return
	}

	sessionID, err := uuid.Parse(chi.URLParam(r, "sessionID"))
	if err != nil {
		responses.WriteError(w, http.StatusBadRequest, ErrInvalidSessionID.Error())
		return
	}

	if err := h.service.DeleteSession(r.Context(), claims, eventID, sessionID); err != nil {
		h.writeSessionError(w, r, "events.delete_session", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) CreateTicketType(w http.ResponseWriter, r *http.Request) {
	claims, ok := httpmiddleware.ClaimsFromContext(r.Context())
	if !ok {
		responses.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	eventID, err := uuid.Parse(chi.URLParam(r, "eventID"))
	if err != nil {
		responses.WriteError(w, http.StatusBadRequest, ErrInvalidEventID.Error())
		return
	}

	sessionID, err := uuid.Parse(chi.URLParam(r, "sessionID"))
	if err != nil {
		responses.WriteError(w, http.StatusBadRequest, ErrInvalidSessionID.Error())
		return
	}

	var input CreateTicketTypeInput
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&input); err != nil {
		logger.RequestError(r, "events.create_ticket_type.decode", err)
		responses.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ticketType, err := h.service.CreateTicketType(r.Context(), claims, eventID, sessionID, input)
	if err != nil {
		h.writeTicketTypeError(w, r, "events.create_ticket_type", err)
		return
	}

	responses.WriteJSON(w, http.StatusCreated, ticketType)
}

func (h *Handler) ListTicketTypes(w http.ResponseWriter, r *http.Request) {
	claims, ok := httpmiddleware.ClaimsFromContext(r.Context())
	if !ok {
		responses.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	eventID, err := uuid.Parse(chi.URLParam(r, "eventID"))
	if err != nil {
		responses.WriteError(w, http.StatusBadRequest, ErrInvalidEventID.Error())
		return
	}

	sessionID, err := uuid.Parse(chi.URLParam(r, "sessionID"))
	if err != nil {
		responses.WriteError(w, http.StatusBadRequest, ErrInvalidSessionID.Error())
		return
	}

	ticketTypes, err := h.service.ListTicketTypes(r.Context(), claims, eventID, sessionID)
	if err != nil {
		h.writeTicketTypeError(w, r, "events.list_ticket_types", err)
		return
	}

	responses.WriteJSON(w, http.StatusOK, ticketTypes)
}

func (h *Handler) UpdateTicketType(w http.ResponseWriter, r *http.Request) {
	claims, ok := httpmiddleware.ClaimsFromContext(r.Context())
	if !ok {
		responses.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	eventID, err := uuid.Parse(chi.URLParam(r, "eventID"))
	if err != nil {
		responses.WriteError(w, http.StatusBadRequest, ErrInvalidEventID.Error())
		return
	}

	sessionID, err := uuid.Parse(chi.URLParam(r, "sessionID"))
	if err != nil {
		responses.WriteError(w, http.StatusBadRequest, ErrInvalidSessionID.Error())
		return
	}

	ticketTypeID, err := uuid.Parse(chi.URLParam(r, "ticketTypeID"))
	if err != nil {
		responses.WriteError(w, http.StatusBadRequest, ErrInvalidTicketTypeID.Error())
		return
	}

	var input UpdateTicketTypeInput
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&input); err != nil {
		logger.RequestError(r, "events.update_ticket_type.decode", err)
		responses.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ticketType, err := h.service.UpdateTicketType(r.Context(), claims, eventID, sessionID, ticketTypeID, input)
	if err != nil {
		h.writeTicketTypeError(w, r, "events.update_ticket_type", err)
		return
	}

	responses.WriteJSON(w, http.StatusOK, ticketType)
}

func (h *Handler) DeleteTicketType(w http.ResponseWriter, r *http.Request) {
	claims, ok := httpmiddleware.ClaimsFromContext(r.Context())
	if !ok {
		responses.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	eventID, err := uuid.Parse(chi.URLParam(r, "eventID"))
	if err != nil {
		responses.WriteError(w, http.StatusBadRequest, ErrInvalidEventID.Error())
		return
	}

	sessionID, err := uuid.Parse(chi.URLParam(r, "sessionID"))
	if err != nil {
		responses.WriteError(w, http.StatusBadRequest, ErrInvalidSessionID.Error())
		return
	}

	ticketTypeID, err := uuid.Parse(chi.URLParam(r, "ticketTypeID"))
	if err != nil {
		responses.WriteError(w, http.StatusBadRequest, ErrInvalidTicketTypeID.Error())
		return
	}

	if err := h.service.DeleteTicketType(r.Context(), claims, eventID, sessionID, ticketTypeID); err != nil {
		h.writeTicketTypeError(w, r, "events.delete_ticket_type", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) writeSessionError(w http.ResponseWriter, r *http.Request, logKey string, err error) {
	logger.RequestError(r, logKey, err)

	switch {
	case errors.Is(err, ErrInvalidStartsAt),
		errors.Is(err, ErrInvalidEndsAt),
		errors.Is(err, ErrInvalidSalesWindow),
		errors.Is(err, ErrInvalidSessionStatus):
		responses.WriteError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrEventNotFound), errors.Is(err, ErrEventSessionNotFound):
		responses.WriteError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrForbidden), errors.Is(err, ErrUnsupportedRole), errors.Is(err, ErrOrganizerScopeRequired):
		responses.WriteError(w, http.StatusForbidden, err.Error())
	default:
		responses.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}

func (h *Handler) writeTicketTypeError(w http.ResponseWriter, r *http.Request, logKey string, err error) {
	logger.RequestError(r, logKey, err)

	switch {
	case errors.Is(err, ErrInvalidTicketName),
		errors.Is(err, ErrInvalidTicketPrice),
		errors.Is(err, ErrInvalidTicketQuantity),
		errors.Is(err, ErrInvalidMaxPerOrder):
		responses.WriteError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrEventNotFound), errors.Is(err, ErrEventSessionNotFound), errors.Is(err, ErrTicketTypeNotFound):
		responses.WriteError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrForbidden), errors.Is(err, ErrUnsupportedRole), errors.Is(err, ErrOrganizerScopeRequired):
		responses.WriteError(w, http.StatusForbidden, err.Error())
	default:
		responses.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}

func (h *Handler) writeReservationError(w http.ResponseWriter, r *http.Request, logKey string, err error) {
	logger.RequestError(r, logKey, err)

	switch {
	case errors.Is(err, ErrInvalidReservationID),
		errors.Is(err, ErrInvalidReservationToken),
		errors.Is(err, ErrInvalidReservationItems),
		errors.Is(err, ErrInvalidTicketTypeID),
		errors.Is(err, ErrInvalidTicketQuantity):
		responses.WriteError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrReservationUnavailable),
		errors.Is(err, ErrReservationMaxPerOrder),
		errors.Is(err, ErrTicketSalesUnavailable):
		responses.WriteError(w, http.StatusConflict, err.Error())
	case errors.Is(err, ErrReservationNotFound):
		responses.WriteError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrTicketTypeNotFound):
		responses.WriteError(w, http.StatusBadRequest, err.Error())
	default:
		responses.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}

func (h *Handler) writeCheckoutOrderError(w http.ResponseWriter, r *http.Request, logKey string, err error) {
	logger.RequestError(r, logKey, err)

	switch {
	case errors.Is(err, ErrInvalidReservationID),
		errors.Is(err, ErrInvalidReservationToken),
		errors.Is(err, ErrInvalidCustomerName),
		errors.Is(err, ErrInvalidCustomerEmail),
		errors.Is(err, ErrInvalidOrderToken):
		responses.WriteError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrReservationNotFound), errors.Is(err, ErrCheckoutOrderNotFound):
		responses.WriteError(w, http.StatusNotFound, err.Error())
	default:
		responses.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}
