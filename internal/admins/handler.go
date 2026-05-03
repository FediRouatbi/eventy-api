package admins

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

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
			errors.Is(err, ErrInvalidAdminEmail):
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

func (h *Handler) GetPayments(w http.ResponseWriter, r *http.Request) {
	claims, ok := httpmiddleware.ClaimsFromContext(r.Context())
	if !ok {
		responses.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	limit := 20
	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed <= 0 {
			responses.WriteError(w, http.StatusBadRequest, ErrInvalidLimit.Error())
			return
		}
		limit = int(math.Min(float64(parsed), 100))
	}

	item, err := h.service.GetPayments(r.Context(), claims, limit)
	if err != nil {
		logger.RequestError(r, "admins.get_payments", err)

		switch {
		case errors.Is(err, ErrOrganizerScopeRequired):
			responses.WriteError(w, http.StatusForbidden, err.Error())
		case errors.Is(err, ErrInvalidLimit):
			responses.WriteError(w, http.StatusBadRequest, err.Error())
		default:
			responses.WriteError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	responses.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) ExportPaymentsCSV(w http.ResponseWriter, r *http.Request) {
	claims, ok := httpmiddleware.ClaimsFromContext(r.Context())
	if !ok {
		responses.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	nowUTC := time.Now().UTC()
	defaultFrom := time.Date(nowUTC.Year(), nowUTC.Month(), nowUTC.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -29)
	defaultTo := time.Date(nowUTC.Year(), nowUTC.Month(), nowUTC.Day(), 0, 0, 0, 0, time.UTC)

	fromDate := defaultFrom
	toDate := defaultTo

	fromRaw := strings.TrimSpace(r.URL.Query().Get("from"))
	if fromRaw != "" {
		parsed, err := time.Parse("2006-01-02", fromRaw)
		if err != nil {
			responses.WriteError(w, http.StatusBadRequest, ErrInvalidFromDate.Error())
			return
		}
		fromDate = parsed.UTC()
	}

	toRaw := strings.TrimSpace(r.URL.Query().Get("to"))
	if toRaw != "" {
		parsed, err := time.Parse("2006-01-02", toRaw)
		if err != nil {
			responses.WriteError(w, http.StatusBadRequest, ErrInvalidToDate.Error())
			return
		}
		toDate = parsed.UTC()
	}

	if toDate.Before(fromDate) {
		responses.WriteError(w, http.StatusBadRequest, ErrInvalidDateRange.Error())
		return
	}
	if toDate.Sub(fromDate) > 365*24*time.Hour {
		responses.WriteError(w, http.StatusBadRequest, ErrInvalidDateRangeWindow.Error())
		return
	}

	filters := AdminPaymentsExportFilters{
		FromDate: fromDate,
		ToDate:   toDate,
	}

	timezoneLabel := "UTC"
	timezoneRaw := strings.TrimSpace(r.URL.Query().Get("timezone"))
	if timezoneRaw != "" {
		if _, err := time.LoadLocation(timezoneRaw); err != nil {
			responses.WriteError(w, http.StatusBadRequest, ErrInvalidTimezone.Error())
			return
		}
		timezoneLabel = timezoneRaw
	}

	if organizerIDRaw := strings.TrimSpace(r.URL.Query().Get("organizer_id")); organizerIDRaw != "" {
		parsedOrganizerID, err := uuid.Parse(organizerIDRaw)
		if err != nil {
			responses.WriteError(w, http.StatusBadRequest, ErrInvalidOrganizerID.Error())
			return
		}
		filters.OrganizerID = &parsedOrganizerID
	}

	content, filename, err := h.service.ExportPaymentsCSV(r.Context(), claims, filters, timezoneLabel)
	if err != nil {
		logger.RequestError(r, "admins.export_payments_csv", err)

		switch {
		case errors.Is(err, ErrOrganizerScopeRequired):
			responses.WriteError(w, http.StatusForbidden, err.Error())
		default:
			responses.WriteError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set(
		"Content-Disposition",
		fmt.Sprintf(`attachment; filename="%s"`, strings.ReplaceAll(filename, `"`, "")),
	)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
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
			errors.Is(err, ErrInvalidAdminEmail):
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
