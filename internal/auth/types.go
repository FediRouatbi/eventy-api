package auth

import (
	"time"

	"github.com/google/uuid"
)

type CreateUserInput struct {
	Name        string `json:"name"`
	Email       string `json:"email"`
	FirebaseUID string `json:"firebase_uid"`
}

type FirebaseLoginInput struct {
	IDToken  string `json:"id_token"`
	Platform string `json:"platform"`
}

type EmailAvailabilityInput struct {
	Email string `json:"email"`
}

type EmailAvailabilityResult struct {
	Available bool `json:"available"`
}

type RefreshTokenInput struct {
	RefreshToken string `json:"refresh_token"`
}

type LogoutInput struct {
	RefreshToken string `json:"refresh_token"`
}

type User struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	Email       string     `json:"email"`
	FirebaseUID *string    `json:"firebase_uid,omitempty"`
	Role        string     `json:"role"`
	OrganizerID *uuid.UUID `json:"organizer_id,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type AuthResult struct {
	AccessToken      string    `json:"access_token"`
	AccessExpiresAt  time.Time `json:"access_expires_at"`
	RefreshToken     string    `json:"refresh_token"`
	RefreshExpiresAt time.Time `json:"refresh_expires_at"`
	User             User      `json:"user"`
}

type MessageResponse struct {
	Message string `json:"message"`
}
