package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv                  string
	HTTPPort                string
	WebBaseURL              string
	CORSAllowedOrigins      []string
	DatabaseURL             string
	JWTSecret               string
	JWTIssuer               string
	JWTTTL                  time.Duration
	RefreshTokenTTL         time.Duration
	RefreshCookieName       string
	RefreshCookieDomain     string
	RefreshCookieSecure     bool
	MailjetAPIKey           string
	MailjetSecretKey        string
	MailjetFromEmail        string
	MailjetFromName         string
	StripeSecretKey         string
	StripeWebhookKey        string
	StripeSuccessURL        string
	StripeCancelURL         string
	GoogleClientIDs         []string
	FirebaseProjectID       string
	FirebaseCredentialsFile string
	FirebaseCredentialsJSON string
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

	cfg := Config{
		AppEnv:                  getEnv("APP_ENV", "development"),
		HTTPPort:                getEnv("HTTP_PORT", "8080"),
		WebBaseURL:              getEnv("WEB_BASE_URL", "http://localhost:3000"),
		DatabaseURL:             strings.TrimSpace(os.Getenv("DATABASE_URL")),
		JWTSecret:               strings.TrimSpace(os.Getenv("JWT_SECRET")),
		JWTIssuer:               getEnv("JWT_ISSUER", "eventy-api"),
		JWTTTL:                  time.Duration(ttlMinutes) * time.Minute,
		RefreshTokenTTL:         time.Duration(refreshTokenTTLHours) * time.Hour,
		RefreshCookieName:       getEnv("REFRESH_COOKIE_NAME", "eventy_refresh_token"),
		RefreshCookieDomain:     strings.TrimSpace(os.Getenv("REFRESH_COOKIE_DOMAIN")),
		MailjetAPIKey:           strings.TrimSpace(os.Getenv("MAILJET_API_KEY")),
		MailjetSecretKey:        strings.TrimSpace(os.Getenv("MAILJET_SECRET_KEY")),
		MailjetFromEmail:        strings.TrimSpace(os.Getenv("MAILJET_FROM_EMAIL")),
		MailjetFromName:         getEnv("MAILJET_FROM_NAME", "Eventy"),
		StripeSecretKey:         strings.TrimSpace(os.Getenv("STRIPE_SECRET_KEY")),
		StripeWebhookKey:        strings.TrimSpace(os.Getenv("STRIPE_WEBHOOK_SECRET")),
		GoogleClientIDs:         parseListWithFallback(os.Getenv("GOOGLE_CLIENT_IDS"), nil),
		FirebaseProjectID:       strings.TrimSpace(os.Getenv("FIREBASE_PROJECT_ID")),
		FirebaseCredentialsFile: strings.TrimSpace(os.Getenv("FIREBASE_CREDENTIALS_FILE")),
		FirebaseCredentialsJSON: strings.TrimSpace(os.Getenv("FIREBASE_CREDENTIALS_JSON")),
	}

	cookieSecureFallback := cfg.AppEnv == "production"
	refreshCookieSecure, err := parseBoolWithFallback(os.Getenv("REFRESH_COOKIE_SECURE"), cookieSecureFallback)
	if err != nil {
		return Config{}, errors.New("REFRESH_COOKIE_SECURE must be a valid boolean")
	}
	cfg.RefreshCookieSecure = refreshCookieSecure

	cfg.StripeSuccessURL = getEnv("STRIPE_CHECKOUT_SUCCESS_URL", cfg.WebBaseURL+"/checkout/complete?session_id={CHECKOUT_SESSION_ID}")
	cfg.StripeCancelURL = getEnv("STRIPE_CHECKOUT_CANCEL_URL", cfg.WebBaseURL+"/checkout?cancelled=1")
	cfg.CORSAllowedOrigins = parseListWithFallback(os.Getenv("CORS_ALLOWED_ORIGINS"), []string{
		cfg.WebBaseURL,
		"http://localhost:3000",
		"http://127.0.0.1:3000",
		"http://localhost:5173",
		"http://127.0.0.1:5173",
	})

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

func parseBoolWithFallback(value string, fallback bool) (bool, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback, nil
	}

	parsed, err := strconv.ParseBool(trimmed)
	if err != nil {
		return false, err
	}

	return parsed, nil
}

func parseListWithFallback(value string, fallback []string) []string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}

	parts := strings.Split(trimmed, ",")
	result := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item == "" {
			continue
		}
		if _, exists := seen[item]; exists {
			continue
		}

		result = append(result, item)
		seen[item] = struct{}{}
	}

	if len(result) == 0 {
		return fallback
	}

	return result
}
