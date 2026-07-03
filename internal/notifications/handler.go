package notifications

import (
	"encoding/json"
	"errors"
	"net/http"

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

func (h *Handler) RegisterDeviceToken(w http.ResponseWriter, r *http.Request) {
	claims, ok := httpmiddleware.ClaimsFromContext(r.Context())
	if !ok {
		responses.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var input RegisterDeviceTokenInput
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&input); err != nil {
		logger.RequestError(r, "notifications.device_tokens.decode", err)
		responses.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.service.RegisterDeviceToken(r.Context(), claims.UserID, input)
	if err != nil {
		h.writeNotificationError(w, r, err)
		return
	}

	responses.WriteJSON(w, http.StatusOK, result)
}

type unregisterDeviceTokenInput struct {
	Token string `json:"token"`
}

func (h *Handler) UnregisterDeviceToken(w http.ResponseWriter, r *http.Request) {
	claims, ok := httpmiddleware.ClaimsFromContext(r.Context())
	if !ok {
		responses.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var input unregisterDeviceTokenInput
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&input); err != nil {
		logger.RequestError(r, "notifications.device_tokens.unregister.decode", err)
		responses.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.service.UnregisterDeviceToken(r.Context(), claims.UserID, input.Token)
	if err != nil {
		h.writeNotificationError(w, r, err)
		return
	}

	responses.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) writeNotificationError(w http.ResponseWriter, r *http.Request, err error) {
	logger.RequestError(r, "notifications", err)

	switch {
	case errors.Is(err, ErrInvalidDeviceToken),
		errors.Is(err, ErrInvalidProvider),
		errors.Is(err, ErrInvalidPlatform):
		responses.WriteError(w, http.StatusBadRequest, err.Error())
	default:
		responses.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}
