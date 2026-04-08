package categories

import (
	"encoding/json"
	"errors"
	"net/http"

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
		case errors.Is(err, ErrCategoryExists), errors.Is(err, ErrCategoryNameUsed), errors.Is(err, ErrCategorySlugUsed):
			responses.WriteError(w, http.StatusConflict, err.Error())
		default:
			responses.WriteError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	responses.WriteJSON(w, http.StatusCreated, category)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	categoryID, err := uuid.Parse(chi.URLParam(r, "categoryID"))
	if err != nil {
		responses.WriteError(w, http.StatusBadRequest, "category_id must be a valid uuid")
		return
	}

	var input UpdateCategoryInput
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&input); err != nil {
		logger.RequestError(r, "categories.update.decode", err)
		responses.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	category, err := h.service.Update(r.Context(), categoryID, input)
	if err != nil {
		logger.RequestError(r, "categories.update", err)

		switch {
		case errors.Is(err, ErrInvalidName), errors.Is(err, ErrInvalidSlug):
			responses.WriteError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrCategoryNotFound):
			responses.WriteError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, ErrCategoryExists), errors.Is(err, ErrCategoryNameUsed), errors.Is(err, ErrCategorySlugUsed):
			responses.WriteError(w, http.StatusConflict, err.Error())
		default:
			responses.WriteError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	responses.WriteJSON(w, http.StatusOK, category)
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

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	categoryID, err := uuid.Parse(chi.URLParam(r, "categoryID"))
	if err != nil {
		responses.WriteError(w, http.StatusBadRequest, "category_id must be a valid uuid")
		return
	}

	err = h.service.Delete(r.Context(), categoryID)
	if err != nil {
		logger.RequestError(r, "categories.delete", err)

		switch {
		case errors.Is(err, ErrCategoryNotFound):
			responses.WriteError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, ErrCategoryHasEvents):
			responses.WriteError(w, http.StatusConflict, err.Error())
		default:
			responses.WriteError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	responses.WriteJSON(w, http.StatusOK, map[string]string{"message": "category deleted successfully"})
}

func (h *Handler) GetPublicBySlug(w http.ResponseWriter, r *http.Request) {
	categorySlug := chi.URLParam(r, "categorySlug")

	category, err := h.service.GetPublicBySlug(r.Context(), categorySlug)
	if err != nil {
		logger.RequestError(r, "categories.get_public_by_slug", err)

		switch {
		case errors.Is(err, ErrCategoryNotFound):
			responses.WriteError(w, http.StatusNotFound, err.Error())
		default:
			responses.WriteError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	responses.WriteJSON(w, http.StatusOK, category)
}
