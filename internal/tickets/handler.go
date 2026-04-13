package tickets

import (
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"

	httpmiddleware "eventy-api/internal/http/middleware"
	"eventy-api/internal/http/responses"
	"eventy-api/internal/platform/logger"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ListMyTickets(w http.ResponseWriter, r *http.Request) {
	claims, ok := httpmiddleware.ClaimsFromContext(r.Context())
	if !ok {
		responses.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	limit := 50
	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed <= 0 {
			responses.WriteError(w, http.StatusBadRequest, "limit must be a positive integer")
			return
		}
		limit = int(math.Min(float64(parsed), 200))
	}

	tickets, err := h.service.ListMyTickets(r.Context(), claims, limit)
	if err != nil {
		logger.RequestError(r, "tickets.list_my_tickets", err)
		responses.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	responses.WriteJSON(w, http.StatusOK, tickets)
}

type checkInTicketRequest struct {
	Code string `json:"code"`
}

func (h *Handler) CheckInTicket(w http.ResponseWriter, r *http.Request) {
	claims, ok := httpmiddleware.ClaimsFromContext(r.Context())
	if !ok {
		responses.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var input checkInTicketRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&input); err != nil {
		logger.RequestError(r, "tickets.check_in.decode", err)
		responses.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.service.CheckInTicket(r.Context(), claims, input.Code)
	if err != nil {
		logger.RequestError(r, "tickets.check_in", err)

		switch {
		case errors.Is(err, ErrInvalidTicketCode):
			responses.WriteError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrTicketNotFound):
			responses.WriteError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, ErrTicketNotPaid):
			responses.WriteError(w, http.StatusConflict, err.Error())
		case errors.Is(err, ErrOrganizerScopeMissing), errors.Is(err, ErrForbidden):
			responses.WriteError(w, http.StatusForbidden, err.Error())
		default:
			responses.WriteError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	responses.WriteJSON(w, http.StatusOK, result)
}
