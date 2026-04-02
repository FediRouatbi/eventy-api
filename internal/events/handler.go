package events

import (
	"encoding/json"
	"errors"
	"net/http"

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

	event, err := h.service.GetByID(r.Context(), claims, eventID)
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
