package categories

import (
	"encoding/json"
	"errors"
	"net/http"

	"eventy-api/internal/http/responses"
	"eventy-api/internal/platform/logger"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var input CreateCategoryInput
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&input); err != nil {
		logger.RequestError(r, "categories.create.decode", err)
		responses.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	category, err := h.service.Create(r.Context(), input)
	if err != nil {
		logger.RequestError(r, "categories.create", err)

		switch {
		case errors.Is(err, ErrInvalidName), errors.Is(err, ErrInvalidSlug):
			responses.WriteError(w, http.StatusBadRequest, err.Error())
		default:
			responses.WriteError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	responses.WriteJSON(w, http.StatusCreated, category)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	categories, err := h.service.List(r.Context())
	if err != nil {
		logger.RequestError(r, "categories.list", err)
		responses.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	responses.WriteJSON(w, http.StatusOK, categories)
}
