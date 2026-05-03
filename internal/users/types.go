package users

import (
	"time"

	"github.com/google/uuid"
)

type Profile struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	Email       string     `json:"email"`
	FirebaseUID *string    `json:"firebase_uid,omitempty"`
	Role        string     `json:"role"`
	OrganizerID *uuid.UUID `json:"organizer_id,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type UpdateProfileInput struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}
