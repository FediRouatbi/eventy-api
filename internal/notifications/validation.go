package notifications

import "strings"

func validateRegisterDeviceTokenInput(input RegisterDeviceTokenInput) error {
	if strings.TrimSpace(input.Token) == "" {
		return ErrInvalidDeviceToken
	}

	switch strings.TrimSpace(input.Provider) {
	case "fcm", "apns":
	default:
		return ErrInvalidProvider
	}

	switch strings.TrimSpace(input.Platform) {
	case "android", "ios":
	default:
		return ErrInvalidPlatform
	}

	return nil
}
