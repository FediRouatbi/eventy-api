package auth

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	httpmiddleware "eventy-api/internal/http/middleware"
	"eventy-api/internal/http/responses"
	"eventy-api/internal/platform/logger"
)

type Handler struct {
	service        *Service
	cookieSettings CookieSettings
}

type CookieSettings struct {
	RefreshTokenName string
	Domain           string
	Secure           bool
}

func NewHandler(service *Service, cookieSettings CookieSettings) *Handler {
	if strings.TrimSpace(cookieSettings.RefreshTokenName) == "" {
		cookieSettings.RefreshTokenName = "eventy_refresh_token"
	}

	return &Handler{
		service:        service,
		cookieSettings: cookieSettings,
	}
}

func (h *Handler) LoginWithFirebase(w http.ResponseWriter, r *http.Request) {
	var input FirebaseLoginInput
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&input); err != nil {
		logger.RequestError(r, "auth.firebase.login.decode", err)
		responses.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.service.LoginWithFirebase(r.Context(), input)
	if err != nil {
		h.writeAuthError(w, r, err)
		return
	}

	h.setRefreshTokenCookie(w, result.RefreshToken, result.RefreshExpiresAt)
	responses.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) CheckFirebaseEmailAvailability(w http.ResponseWriter, r *http.Request) {
	var input EmailAvailabilityInput
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&input); err != nil {
		logger.RequestError(r, "auth.firebase.email_availability.decode", err)
		responses.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.service.CheckEmailAvailability(r.Context(), input)
	if err != nil {
		h.writeAuthError(w, r, err)
		return
	}

	responses.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) RefreshSession(w http.ResponseWriter, r *http.Request) {
	input, err := h.decodeRefreshTokenInput(r)
	if err != nil {
		logger.RequestError(r, "auth.refresh.decode", err)
		responses.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.service.RefreshSession(r.Context(), input)
	if err != nil {
		h.writeAuthError(w, r, err)
		return
	}

	h.setRefreshTokenCookie(w, result.RefreshToken, result.RefreshExpiresAt)
	responses.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	claims, ok := httpmiddleware.ClaimsFromContext(r.Context())
	if !ok {
		responses.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	input, err := h.decodeLogoutInput(r)
	if err != nil {
		logger.RequestError(r, "auth.logout.decode", err)
		responses.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.service.Logout(r.Context(), claims.UserID, input)
	if err != nil {
		h.writeAuthError(w, r, err)
		return
	}

	h.clearRefreshTokenCookie(w)
	responses.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) writeAuthError(w http.ResponseWriter, r *http.Request, err error) {
	logger.RequestError(r, "auth", err)

	switch {
	case errors.Is(err, ErrInvalidEmail),
		errors.Is(err, ErrInvalidFirebaseToken):
		responses.WriteError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrEmailAlreadyExists):
		responses.WriteError(w, http.StatusConflict, err.Error())
	case errors.Is(err, ErrInvalidCredentials),
		errors.Is(err, ErrFirebaseEmailNotVerified),
		errors.Is(err, ErrInvalidRefreshToken),
		errors.Is(err, ErrSessionExpired),
		errors.Is(err, ErrSessionRevoked):
		responses.WriteError(w, http.StatusUnauthorized, err.Error())
	default:
		responses.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}

func (h *Handler) decodeRefreshTokenInput(r *http.Request) (RefreshTokenInput, error) {
	var input RefreshTokenInput
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil && !errors.Is(err, io.EOF) {
		return RefreshTokenInput{}, err
	}

	if strings.TrimSpace(input.RefreshToken) == "" {
		input.RefreshToken = h.readRefreshTokenCookie(r)
	}

	return input, nil
}

func (h *Handler) decodeLogoutInput(r *http.Request) (LogoutInput, error) {
	var input LogoutInput
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil && !errors.Is(err, io.EOF) {
		return LogoutInput{}, err
	}

	if strings.TrimSpace(input.RefreshToken) == "" {
		input.RefreshToken = h.readRefreshTokenCookie(r)
	}

	return input, nil
}

func (h *Handler) readRefreshTokenCookie(r *http.Request) string {
	cookie, err := r.Cookie(h.cookieSettings.RefreshTokenName)
	if err != nil {
		return ""
	}

	return strings.TrimSpace(cookie.Value)
}

func (h *Handler) setRefreshTokenCookie(w http.ResponseWriter, refreshToken string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.cookieSettings.RefreshTokenName,
		Value:    refreshToken,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   h.cookieSettings.Secure,
		SameSite: http.SameSiteLaxMode,
		Domain:   h.cookieSettings.Domain,
	})
}

func (h *Handler) clearRefreshTokenCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.cookieSettings.RefreshTokenName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.cookieSettings.Secure,
		SameSite: http.SameSiteLaxMode,
		Domain:   h.cookieSettings.Domain,
	})
}
