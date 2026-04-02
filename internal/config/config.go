package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv           string
	HTTPPort         string
	DatabaseURL      string
	JWTSecret        string
	JWTIssuer        string
	JWTTTL           time.Duration
	RefreshTokenTTL  time.Duration
	RegisterOTPTTL   time.Duration
	MailjetAPIKey    string
	MailjetSecretKey string
	MailjetFromEmail string
	MailjetFromName  string
}

func Load() (Config, error) {
	ttlMinutes, err := parseInt(getEnv("JWT_TTL_MINUTES", "60"))
	if err != nil {
		return Config{}, errors.New("JWT_TTL_MINUTES must be a valid integer")
	}

	refreshTokenTTLHours, err := parseInt(getEnv("REFRESH_TOKEN_TTL_HOURS", "720"))
	if err != nil {
		return Config{}, errors.New("REFRESH_TOKEN_TTL_HOURS must be a valid integer")
	}

	registerOTPTTLMinutes, err := parseInt(getEnv("REGISTER_OTP_TTL_MINUTES", "10"))
	if err != nil {
		return Config{}, errors.New("REGISTER_OTP_TTL_MINUTES must be a valid integer")
	}

	cfg := Config{
		AppEnv:           getEnv("APP_ENV", "development"),
		HTTPPort:         getEnv("HTTP_PORT", "8080"),
		DatabaseURL:      strings.TrimSpace(os.Getenv("DATABASE_URL")),
		JWTSecret:        strings.TrimSpace(os.Getenv("JWT_SECRET")),
		JWTIssuer:        getEnv("JWT_ISSUER", "eventy-api"),
		JWTTTL:           time.Duration(ttlMinutes) * time.Minute,
		RefreshTokenTTL:  time.Duration(refreshTokenTTLHours) * time.Hour,
		RegisterOTPTTL:   time.Duration(registerOTPTTLMinutes) * time.Minute,
		MailjetAPIKey:    strings.TrimSpace(os.Getenv("MAILJET_API_KEY")),
		MailjetSecretKey: strings.TrimSpace(os.Getenv("MAILJET_SECRET_KEY")),
		MailjetFromEmail: strings.TrimSpace(os.Getenv("MAILJET_FROM_EMAIL")),
		MailjetFromName:  getEnv("MAILJET_FROM_NAME", "Eventy"),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}

	if cfg.JWTSecret == "" {
		return Config{}, errors.New("JWT_SECRET is required")
	}

	if cfg.MailjetAPIKey == "" {
		return Config{}, errors.New("MAILJET_API_KEY is required")
	}

	if cfg.MailjetSecretKey == "" {
		return Config{}, errors.New("MAILJET_SECRET_KEY is required")
	}

	if cfg.MailjetFromEmail == "" {
		return Config{}, errors.New("MAILJET_FROM_EMAIL is required")
	}

	return cfg, nil
}

func getEnv(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	return value
}

func parseInt(value string) (int, error) {
	return strconv.Atoi(strings.TrimSpace(value))
}
