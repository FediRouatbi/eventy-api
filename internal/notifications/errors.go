package notifications

import "errors"

var (
	ErrInvalidDeviceToken = errors.New("device token is required")
	ErrInvalidProvider    = errors.New("provider must be fcm or apns")
	ErrInvalidPlatform    = errors.New("platform must be android or ios")
)
