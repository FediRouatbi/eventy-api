package firebase

import (
	"context"
	"errors"
	"strings"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/idtoken"
	"google.golang.org/api/option"
)

type Config struct {
	ProjectID       string
	CredentialsFile string
	CredentialsJSON string
	GoogleClientIDs []string
}

type Services struct {
	Messaging        *messaging.Client
	GoogleVerifier   *GoogleVerifier
	FirebaseVerifier *FirebaseVerifier
}

func NewServices(ctx context.Context, cfg Config) (*Services, error) {
	services := &Services{
		GoogleVerifier: NewGoogleVerifier(cfg.GoogleClientIDs),
	}

	messagingClient, err := NewMessagingClient(ctx, cfg)
	if err != nil {
		return nil, err
	}
	services.Messaging = messagingClient
	services.FirebaseVerifier = NewFirebaseVerifier(nil)

	authClient, err := NewAuthClient(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if authClient != nil {
		services.FirebaseVerifier = NewFirebaseVerifier(authClient)
	}

	return services, nil
}

func newApp(ctx context.Context, cfg Config) (*firebase.App, error) {
	options := make([]option.ClientOption, 0, 1)
	if strings.TrimSpace(cfg.CredentialsFile) != "" {
		options = append(options, option.WithCredentialsFile(strings.TrimSpace(cfg.CredentialsFile)))
	} else if strings.TrimSpace(cfg.CredentialsJSON) != "" {
		options = append(options, option.WithCredentialsJSON([]byte(cfg.CredentialsJSON)))
	}

	if len(options) == 0 && strings.TrimSpace(cfg.ProjectID) == "" {
		return nil, nil
	}

	return firebase.NewApp(ctx, &firebase.Config{
		ProjectID: strings.TrimSpace(cfg.ProjectID),
	}, options...)
}

func NewMessagingClient(ctx context.Context, cfg Config) (*messaging.Client, error) {
	app, err := newApp(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if app == nil {
		return nil, nil
	}

	return app.Messaging(ctx)
}

func NewAuthClient(ctx context.Context, cfg Config) (*auth.Client, error) {
	app, err := newApp(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if app == nil {
		return nil, nil
	}

	return app.Auth(ctx)
}

type GoogleVerifier struct {
	clientIDs []string
}

func NewGoogleVerifier(clientIDs []string) *GoogleVerifier {
	clean := make([]string, 0, len(clientIDs))
	for _, clientID := range clientIDs {
		trimmed := strings.TrimSpace(clientID)
		if trimmed != "" {
			clean = append(clean, trimmed)
		}
	}

	return &GoogleVerifier{clientIDs: clean}
}

func (v *GoogleVerifier) VerifyIDToken(ctx context.Context, rawIDToken string) (FirebaseIdentity, error) {
	if v == nil || len(v.clientIDs) == 0 {
		return FirebaseIdentity{}, errors.New("google client ids are not configured")
	}

	rawIDToken = strings.TrimSpace(rawIDToken)
	if rawIDToken == "" {
		return FirebaseIdentity{}, errors.New("google id token is required")
	}

	var lastErr error
	for _, clientID := range v.clientIDs {
		payload, err := idtoken.Validate(ctx, rawIDToken, clientID)
		if err != nil {
			lastErr = err
			continue
		}

		email, _ := payload.Claims["email"].(string)
		name, _ := payload.Claims["name"].(string)
		emailVerified, _ := payload.Claims["email_verified"].(bool)

		return FirebaseIdentity{
			Email:         email,
			Name:          name,
			EmailVerified: emailVerified,
		}, nil
	}

	if lastErr != nil {
		return FirebaseIdentity{}, lastErr
	}

	return FirebaseIdentity{}, errors.New("google id token audience is not allowed")
}

type FirebaseVerifier struct {
	client *auth.Client
}

func NewFirebaseVerifier(client *auth.Client) *FirebaseVerifier {
	return &FirebaseVerifier{client: client}
}

type FirebaseIdentity struct {
	UID           string
	Email         string
	Name          string
	EmailVerified bool
}

func (i FirebaseIdentity) GetUID() string {
	return i.UID
}

func (i FirebaseIdentity) GetEmail() string {
	return i.Email
}

func (i FirebaseIdentity) GetName() string {
	return i.Name
}

func (i FirebaseIdentity) IsEmailVerified() bool {
	return i.EmailVerified
}

func (v *FirebaseVerifier) VerifyIDToken(ctx context.Context, rawIDToken string) (FirebaseIdentity, error) {
	if v == nil || v.client == nil {
		return FirebaseIdentity{}, errors.New("firebase auth is not configured")
	}

	rawIDToken = strings.TrimSpace(rawIDToken)
	if rawIDToken == "" {
		return FirebaseIdentity{}, errors.New("firebase id token is required")
	}

	token, err := v.client.VerifyIDToken(ctx, rawIDToken)
	if err != nil {
		return FirebaseIdentity{}, err
	}

	email, _ := token.Claims["email"].(string)
	name, _ := token.Claims["name"].(string)
	emailVerified, _ := token.Claims["email_verified"].(bool)
	if token.Firebase.SignInProvider == "password" && email != "" {
		emailVerified = true
	}

	return FirebaseIdentity{
		UID:           token.UID,
		Email:         email,
		Name:          name,
		EmailVerified: emailVerified,
	}, nil
}

func (v *FirebaseVerifier) DeleteUser(ctx context.Context, uid string) error {
	if v == nil || v.client == nil {
		return errors.New("firebase auth is not configured")
	}

	uid = strings.TrimSpace(uid)
	if uid == "" {
		return errors.New("firebase uid is required")
	}

	if err := v.client.DeleteUser(ctx, uid); err != nil {
		if auth.IsUserNotFound(err) {
			return nil
		}

		return err
	}

	return nil
}
