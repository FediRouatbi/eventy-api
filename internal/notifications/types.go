package notifications

import (
	"time"

	"github.com/google/uuid"
)

type RegisterDeviceTokenInput struct {
	Token      string `json:"token"`
	Provider   string `json:"provider"`
	Platform   string `json:"platform"`
	DeviceName string `json:"device_name"`
}

type DeviceToken struct {
	ID         uuid.UUID  `json:"id"`
	UserID     uuid.UUID  `json:"user_id"`
	Token      string     `json:"token"`
	Provider   string     `json:"provider"`
	Platform   string     `json:"platform"`
	DeviceName string     `json:"device_name,omitempty"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type MessageResponse struct {
	Message string `json:"message"`
}
