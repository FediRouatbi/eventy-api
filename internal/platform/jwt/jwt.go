package jwt

import (
	"errors"
	"time"

	jwtv5 "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	UserID      uuid.UUID  `json:"user_id"`
	Email       string     `json:"email"`
	Role        string     `json:"role"`
	OrganizerID *uuid.UUID `json:"organizer_id,omitempty"`
	jwtv5.RegisteredClaims
}

type Manager struct {
	secret []byte
	issuer string
	ttl    time.Duration
}

func NewManager(secret string, issuer string, ttl time.Duration) *Manager {
	return &Manager{
		secret: []byte(secret),
		issuer: issuer,
		ttl:    ttl,
	}
}

func (m *Manager) Generate(userID uuid.UUID, email string, role string, organizerID *uuid.UUID) (string, time.Time, error) {
	expiresAt := time.Now().Add(m.ttl)

	claims := Claims{
		UserID:      userID,
		Email:       email,
		Role:        role,
		OrganizerID: organizerID,
		RegisteredClaims: jwtv5.RegisteredClaims{
			Subject:   userID.String(),
			Issuer:    m.issuer,
			IssuedAt:  jwtv5.NewNumericDate(time.Now()),
			ExpiresAt: jwtv5.NewNumericDate(expiresAt),
		},
	}

	token := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, err
	}

	return signedToken, expiresAt, nil
}

func (m *Manager) Parse(token string) (*Claims, error) {
	parsedToken, err := jwtv5.ParseWithClaims(token, &Claims{}, func(_ *jwtv5.Token) (any, error) {
		return m.secret, nil
	}, jwtv5.WithValidMethods([]string{jwtv5.SigningMethodHS256.Alg()}))
	if err != nil {
		return nil, err
	}

	claims, ok := parsedToken.Claims.(*Claims)
	if !ok || !parsedToken.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}
