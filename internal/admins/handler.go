package admins

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

func (h *Handler) CreateOrganizerAdmin(w http.ResponseWriter, r *http.Request) {
	var input CreateOrganizerAdminInput
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&input); err != nil {
		logger.RequestError(r, "admins.create_organizer_admin.decode", err)
		responses.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.service.CreateOrganizerAdmin(r.Context(), input)
	if err != nil {
		logger.RequestError(r, "admins.create_organizer_admin", err)

		switch {
		case errors.Is(err, ErrInvalidOrganizerName),
			errors.Is(err, ErrInvalidOrganizerSlug),
			errors.Is(err, ErrInvalidAdminName),
			errors.Is(err, ErrInvalidAdminEmail),
			errors.Is(err, ErrInvalidAdminPassword):
			responses.WriteError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrOrganizerSlugExists), errors.Is(err, ErrAdminEmailExists):
			responses.WriteError(w, http.StatusConflict, err.Error())
		default:
			responses.WriteError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	responses.WriteJSON(w, http.StatusCreated, result)
}

func (h *Handler) GetOverview(w http.ResponseWriter, r *http.Request) {
	claims, ok := httpmiddleware.ClaimsFromContext(r.Context())
	if !ok {
		responses.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	item, err := h.service.GetOverview(r.Context(), claims)
	if err != nil {
		logger.RequestError(r, "admins.get_overview", err)

		switch {
		case errors.Is(err, ErrOrganizerScopeRequired):
			responses.WriteError(w, http.StatusForbidden, err.Error())
		default:
			responses.WriteError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	responses.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) ListOrganizers(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListOrganizers(r.Context())
	if err != nil {
		logger.RequestError(r, "admins.list_organizers", err)
		responses.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	responses.WriteJSON(w, http.StatusOK, items)
}

func (h *Handler) UpdateOrganizer(w http.ResponseWriter, r *http.Request) {
	organizerID, err := uuid.Parse(chi.URLParam(r, "organizerID"))
	if err != nil {
		responses.WriteError(w, http.StatusBadRequest, ErrInvalidOrganizerID.Error())
		return
	}

	var input UpdateOrganizerInput
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&input); err != nil {
		logger.RequestError(r, "admins.update_organizer.decode", err)
		responses.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	item, err := h.service.UpdateOrganizer(r.Context(), organizerID, input)
	if err != nil {
		logger.RequestError(r, "admins.update_organizer", err)

		switch {
		case errors.Is(err, ErrInvalidOrganizerName), errors.Is(err, ErrInvalidOrganizerSlug):
			responses.WriteError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrOrganizerSlugExists):
			responses.WriteError(w, http.StatusConflict, err.Error())
		case errors.Is(err, ErrOrganizerNotFound):
			responses.WriteError(w, http.StatusNotFound, err.Error())
		default:
			responses.WriteError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	responses.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) GetOrganizer(w http.ResponseWriter, r *http.Request) {
	organizerID, err := uuid.Parse(chi.URLParam(r, "organizerID"))
	if err != nil {
		responses.WriteError(w, http.StatusBadRequest, ErrInvalidOrganizerID.Error())
		return
	}

	item, err := h.service.GetOrganizer(r.Context(), organizerID)
	if err != nil {
		logger.RequestError(r, "admins.get_organizer", err)

		switch {
		case errors.Is(err, ErrOrganizerNotFound):
			responses.WriteError(w, http.StatusNotFound, err.Error())
		default:
			responses.WriteError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	responses.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) ListOrganizerAdmins(w http.ResponseWriter, r *http.Request) {
	organizerID, err := uuid.Parse(chi.URLParam(r, "organizerID"))
	if err != nil {
		responses.WriteError(w, http.StatusBadRequest, ErrInvalidOrganizerID.Error())
		return
	}

	items, err := h.service.ListOrganizerAdmins(r.Context(), organizerID)
	if err != nil {
		logger.RequestError(r, "admins.list_organizer_admins", err)

		switch {
		case errors.Is(err, ErrOrganizerNotFound):
			responses.WriteError(w, http.StatusNotFound, err.Error())
		default:
			responses.WriteError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	responses.WriteJSON(w, http.StatusOK, items)
}

func (h *Handler) AddOrganizerAdmin(w http.ResponseWriter, r *http.Request) {
	organizerID, err := uuid.Parse(chi.URLParam(r, "organizerID"))
	if err != nil {
		responses.WriteError(w, http.StatusBadRequest, ErrInvalidOrganizerID.Error())
		return
	}

	var input AddOrganizerAdminInput
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&input); err != nil {
		logger.RequestError(r, "admins.add_organizer_admin.decode", err)
		responses.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	item, err := h.service.AddOrganizerAdmin(r.Context(), organizerID, input)
	if err != nil {
		logger.RequestError(r, "admins.add_organizer_admin", err)

		switch {
		case errors.Is(err, ErrInvalidAdminName),
			errors.Is(err, ErrInvalidAdminEmail),
			errors.Is(err, ErrInvalidAdminPassword):
			responses.WriteError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrOrganizerNotFound):
			responses.WriteError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, ErrAdminEmailExists):
			responses.WriteError(w, http.StatusConflict, err.Error())
		default:
			responses.WriteError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	responses.WriteJSON(w, http.StatusCreated, item)
}

func (h *Handler) GetOrganizerAdmin(w http.ResponseWriter, r *http.Request) {
	organizerID, err := uuid.Parse(chi.URLParam(r, "organizerID"))
	if err != nil {
		responses.WriteError(w, http.StatusBadRequest, ErrInvalidOrganizerID.Error())
		return
	}

	adminID, err := uuid.Parse(chi.URLParam(r, "adminID"))
	if err != nil {
		responses.WriteError(w, http.StatusBadRequest, ErrInvalidAdminID.Error())
		return
	}

	item, err := h.service.GetOrganizerAdmin(r.Context(), organizerID, adminID)
	if err != nil {
		logger.RequestError(r, "admins.get_organizer_admin", err)

		switch {
		case errors.Is(err, ErrOrganizerNotFound), errors.Is(err, ErrOrganizerAdminNotFound):
			responses.WriteError(w, http.StatusNotFound, err.Error())
		default:
			responses.WriteError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	responses.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) UpdateOrganizerAdmin(w http.ResponseWriter, r *http.Request) {
	organizerID, err := uuid.Parse(chi.URLParam(r, "organizerID"))
	if err != nil {
		responses.WriteError(w, http.StatusBadRequest, ErrInvalidOrganizerID.Error())
		return
	}

	adminID, err := uuid.Parse(chi.URLParam(r, "adminID"))
	if err != nil {
		responses.WriteError(w, http.StatusBadRequest, ErrInvalidAdminID.Error())
		return
	}

	var input UpdateOrganizerAdminInput
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&input); err != nil {
		logger.RequestError(r, "admins.update_organizer_admin.decode", err)
		responses.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	item, err := h.service.UpdateOrganizerAdmin(r.Context(), organizerID, adminID, input)
	if err != nil {
		logger.RequestError(r, "admins.update_organizer_admin", err)

		switch {
		case errors.Is(err, ErrInvalidAdminName), errors.Is(err, ErrInvalidAdminEmail):
			responses.WriteError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrAdminEmailExists):
			responses.WriteError(w, http.StatusConflict, err.Error())
		case errors.Is(err, ErrOrganizerNotFound), errors.Is(err, ErrOrganizerAdminNotFound):
			responses.WriteError(w, http.StatusNotFound, err.Error())
		default:
			responses.WriteError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	responses.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) ResetOrganizerAdminPassword(w http.ResponseWriter, r *http.Request) {
	organizerID, err := uuid.Parse(chi.URLParam(r, "organizerID"))
	if err != nil {
		responses.WriteError(w, http.StatusBadRequest, ErrInvalidOrganizerID.Error())
		return
	}

	adminID, err := uuid.Parse(chi.URLParam(r, "adminID"))
	if err != nil {
		responses.WriteError(w, http.StatusBadRequest, ErrInvalidAdminID.Error())
		return
	}

	var input ResetOrganizerAdminPasswordInput
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&input); err != nil {
		logger.RequestError(r, "admins.reset_organizer_admin_password.decode", err)
		responses.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	err = h.service.ResetOrganizerAdminPassword(r.Context(), organizerID, adminID, input)
	if err != nil {
		logger.RequestError(r, "admins.reset_organizer_admin_password", err)

		switch {
		case errors.Is(err, ErrInvalidAdminPassword):
			responses.WriteError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrOrganizerNotFound), errors.Is(err, ErrOrganizerAdminNotFound):
			responses.WriteError(w, http.StatusNotFound, err.Error())
		default:
			responses.WriteError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	responses.WriteJSON(w, http.StatusOK, map[string]string{"message": "organizer admin password updated successfully"})
}

func (h *Handler) DeleteOrganizerAdmin(w http.ResponseWriter, r *http.Request) {
	organizerID, err := uuid.Parse(chi.URLParam(r, "organizerID"))
	if err != nil {
		responses.WriteError(w, http.StatusBadRequest, ErrInvalidOrganizerID.Error())
		return
	}

	adminID, err := uuid.Parse(chi.URLParam(r, "adminID"))
	if err != nil {
		responses.WriteError(w, http.StatusBadRequest, ErrInvalidAdminID.Error())
		return
	}

	err = h.service.DeleteOrganizerAdmin(r.Context(), organizerID, adminID)
	if err != nil {
		logger.RequestError(r, "admins.delete_organizer_admin", err)

		switch {
		case errors.Is(err, ErrLastOrganizerAdmin):
			responses.WriteError(w, http.StatusConflict, err.Error())
		case errors.Is(err, ErrOrganizerNotFound), errors.Is(err, ErrOrganizerAdminNotFound):
			responses.WriteError(w, http.StatusNotFound, err.Error())
		default:
			responses.WriteError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	responses.WriteJSON(w, http.StatusOK, map[string]string{"message": "organizer admin deleted successfully"})
}

func (h *Handler) DeleteOrganizer(w http.ResponseWriter, r *http.Request) {
	organizerID, err := uuid.Parse(chi.URLParam(r, "organizerID"))
	if err != nil {
		responses.WriteError(w, http.StatusBadRequest, ErrInvalidOrganizerID.Error())
		return
	}

	err = h.service.DeleteOrganizer(r.Context(), organizerID)
	if err != nil {
		logger.RequestError(r, "admins.delete_organizer", err)

		switch {
		case errors.Is(err, ErrOrganizerNotFound):
			responses.WriteError(w, http.StatusNotFound, err.Error())
		default:
			responses.WriteError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	responses.WriteJSON(w, http.StatusOK, map[string]string{"message": "organizer deleted successfully"})
}
