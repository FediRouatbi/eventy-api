package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"eventy-api/internal/platform/db"
	eventyfirebase "eventy-api/internal/platform/firebase"
	"eventy-api/internal/platform/roles"
	firebaseauth "firebase.google.com/go/v4/auth"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using system env vars")
	}

	var (
		email    = flag.String("email", strings.TrimSpace(os.Getenv("ADMIN_EMAIL")), "admin email")
		password = flag.String("password", strings.TrimSpace(os.Getenv("ADMIN_PASSWORD")), "admin Firebase password")
		name     = flag.String("name", strings.TrimSpace(os.Getenv("ADMIN_NAME")), "admin display name")
	)
	flag.Parse()

	*email = normalizeEmail(*email)
	*name = strings.TrimSpace(*name)
	if *name == "" {
		*name = *email
	}

	if *email == "" {
		log.Fatal("admin email is required: pass -email or set ADMIN_EMAIL")
	}
	if strings.TrimSpace(*password) == "" {
		log.Fatal("admin password is required: pass -password or set ADMIN_PASSWORD")
	}

	ctx := context.Background()
	authClient, err := eventyfirebase.NewAuthClient(ctx, eventyfirebase.Config{
		ProjectID:       strings.TrimSpace(os.Getenv("FIREBASE_PROJECT_ID")),
		CredentialsFile: strings.TrimSpace(os.Getenv("FIREBASE_CREDENTIALS_FILE")),
		CredentialsJSON: strings.TrimSpace(os.Getenv("FIREBASE_CREDENTIALS_JSON")),
	})
	if err != nil {
		log.Fatalf("connect firebase auth: %v", err)
	}
	if authClient == nil {
		log.Fatal("firebase auth is not configured: set FIREBASE_PROJECT_ID and service account credentials")
	}

	firebaseUser, err := ensureFirebaseUser(ctx, authClient, *email, *password, *name)
	if err != nil {
		log.Fatalf("seed firebase user: %v", err)
	}

	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	database, err := db.Open(databaseURL)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer database.Close()

	if err := ensureSuperAdmin(ctx, database, *name, *email, firebaseUser.UID); err != nil {
		log.Fatalf("seed database admin: %v", err)
	}

	fmt.Printf("Seeded super admin %s with Firebase UID %s\n", *email, firebaseUser.UID)
}

func ensureFirebaseUser(ctx context.Context, client *firebaseauth.Client, email string, password string, name string) (*firebaseauth.UserRecord, error) {
	user, err := client.GetUserByEmail(ctx, email)
	if err != nil {
		if !firebaseauth.IsUserNotFound(err) {
			return nil, err
		}

		params := (&firebaseauth.UserToCreate{}).
			Email(email).
			Password(password).
			DisplayName(name).
			EmailVerified(true)
		return client.CreateUser(ctx, params)
	}

	params := (&firebaseauth.UserToUpdate{}).
		Password(password).
		DisplayName(name).
		EmailVerified(true)
	return client.UpdateUser(ctx, user.UID, params)
}

func ensureSuperAdmin(ctx context.Context, database *sql.DB, name string, email string, firebaseUID string) error {
	var existingID string
	err := database.QueryRowContext(ctx, `
		SELECT id
		FROM users
		WHERE email = ?
		LIMIT 1
	`, email).Scan(&existingID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	if err == nil {
		_, err = database.ExecContext(ctx, `
			UPDATE users
			SET name = ?,
				firebase_uid = ?,
				role = ?,
				organizer_id = NULL
			WHERE id = ?
		`, name, firebaseUID, roles.SuperAdmin, existingID)
		return err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(uuid.NewString()), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	_, err = database.ExecContext(ctx, `
		INSERT INTO users (
			id,
			name,
			email,
			firebase_uid,
			password_hash,
			role,
			organizer_id
		) VALUES (?, ?, ?, ?, ?, ?, NULL)
	`, uuid.NewString(), name, email, firebaseUID, string(passwordHash), roles.SuperAdmin)
	return err
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
