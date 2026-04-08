package auth

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

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var input RegisterInput
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&input); err != nil {
		logger.RequestError(r, "auth.register.decode", err)
		responses.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.service.Register(r.Context(), input)
	if err != nil {
		h.writeAuthError(w, r, err)
		return
	}

	responses.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) VerifyRegisterOTP(w http.ResponseWriter, r *http.Request) {
	var input VerifyRegisterOTPInput
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&input); err != nil {
		logger.RequestError(r, "auth.verify_register_otp.decode", err)
		responses.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.service.VerifyRegisterOTP(r.Context(), input)
	if err != nil {
		h.writeAuthError(w, r, err)
		return
	}

	responses.WriteJSON(w, http.StatusCreated, result)
}

func (h *Handler) ResendRegisterOTP(w http.ResponseWriter, r *http.Request) {
	var input ResendRegisterOTPInput
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&input); err != nil {
		logger.RequestError(r, "auth.resend_register_otp.decode", err)
		responses.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.service.ResendRegisterOTP(r.Context(), input)
	if err != nil {
		h.writeAuthError(w, r, err)
		return
	}

	responses.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var input LoginInput
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&input); err != nil {
		logger.RequestError(r, "auth.login.decode", err)
		responses.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.service.Login(r.Context(), input)
	if err != nil {
		h.writeAuthError(w, r, err)
		return
	}

	responses.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) RefreshSession(w http.ResponseWriter, r *http.Request) {
	var input RefreshTokenInput
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&input); err != nil {
		logger.RequestError(r, "auth.refresh.decode", err)
		responses.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.service.RefreshSession(r.Context(), input)
	if err != nil {
		h.writeAuthError(w, r, err)
		return
	}

	responses.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var input ForgotPasswordInput
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&input); err != nil {
		logger.RequestError(r, "auth.forgot_password.decode", err)
		responses.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.service.ForgotPassword(r.Context(), input)
	if err != nil {
		h.writeAuthError(w, r, err)
		return
	}

	responses.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var input ResetPasswordInput
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&input); err != nil {
		logger.RequestError(r, "auth.reset_password.decode", err)
		responses.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.service.ResetPassword(r.Context(), input)
	if err != nil {
		h.writeAuthError(w, r, err)
		return
	}

	responses.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	claims, ok := httpmiddleware.ClaimsFromContext(r.Context())
	if !ok {
		responses.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var input ChangePasswordInput
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&input); err != nil {
		logger.RequestError(r, "auth.change_password.decode", err)
		responses.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.service.ChangePassword(r.Context(), claims.UserID, input)
	if err != nil {
		h.writeAuthError(w, r, err)
		return
	}

	responses.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if _, ok := httpmiddleware.ClaimsFromContext(r.Context()); !ok {
		responses.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var input LogoutInput
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&input); err != nil {
		logger.RequestError(r, "auth.logout.decode", err)
		responses.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.service.Logout(r.Context(), input)
	if err != nil {
		h.writeAuthError(w, r, err)
		return
	}

	responses.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) writeAuthError(w http.ResponseWriter, r *http.Request, err error) {
	logger.RequestError(r, "auth", err)

	switch {
	case errors.Is(err, ErrInvalidName),
		errors.Is(err, ErrInvalidEmail),
		errors.Is(err, ErrInvalidPassword),
		errors.Is(err, ErrInvalidOTP):
		responses.WriteError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrEmailAlreadyExists):
		responses.WriteError(w, http.StatusConflict, err.Error())
	case errors.Is(err, ErrInvalidCredentials),
		errors.Is(err, ErrInvalidRefreshToken),
		errors.Is(err, ErrSessionExpired),
		errors.Is(err, ErrSessionRevoked):
		responses.WriteError(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, ErrPendingOTPNotFound),
		errors.Is(err, ErrOTPExpired),
		errors.Is(err, ErrOTPDoesNotMatch):
		responses.WriteError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrPasswordResetNotFound),
		errors.Is(err, ErrPasswordResetExpired),
		errors.Is(err, ErrPasswordResetInvalid):
		responses.WriteError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrCurrentPasswordWrong):
		responses.WriteError(w, http.StatusUnauthorized, err.Error())
	default:
		responses.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}
