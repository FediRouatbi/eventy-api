# eventy-api

Eventy API built with Go, Chi, MySQL, sqlc, and JWT.

## API docs

- `GET /openapi.json`
- `GET /docs`

Frontend and mobile can use `/openapi.json` to generate typed API clients.
`/docs` now serves Scalar interactive API documentation.

## Implemented module

- auth register
- auth resend register otp
- auth register otp verification
- auth forgot password
- auth reset password
- auth login
- auth refresh session
- auth logout
- role guards foundation
- organizer ownership foundation
- categories module
- events/session/ticket schema foundation
- events module

## Auth endpoints

- `POST /v1/auth/register`
- `POST /v1/auth/register/resend-otp`
- `POST /v1/auth/register/verify`
- `POST /v1/auth/forgot-password`
- `POST /v1/auth/reset-password`
- `POST /v1/auth/login`
- `POST /v1/auth/refresh`
- `POST /v1/auth/logout`

## User endpoint

- `GET /v1/users/me`

## Category endpoints

- `GET /v1/categories`
- `POST /v1/categories`

## Event endpoints

- `GET /v1/events`
- `POST /v1/events`
- `GET /v1/events/{eventID}`

## Request examples

### Register

```json
{
  "name": "John Doe",
  "email": "john@example.com",
  "password": "secret123"
}
```

This endpoint now sends an OTP email and does not create the user yet.

### Verify Register OTP

```json
{
  "email": "john@example.com",
  "otp": "123456"
}
```

### Resend Register OTP

```json
{
  "email": "john@example.com"
}
```

### Login

```json
{
  "email": "john@example.com",
  "password": "secret123"
}
```

Login and verify-register now return:

```json
{
  "access_token": "jwt",
  "access_expires_at": "2026-04-02T20:00:00Z",
  "refresh_token": "opaque-token",
  "refresh_expires_at": "2026-05-02T20:00:00Z",
  "user": {}
}
```

### Forgot Password

```json
{
  "email": "john@example.com"
}
```

### Reset Password

```json
{
  "email": "john@example.com",
  "token": "123456",
  "new_password": "newSecret123"
}
```

### Refresh Session

```json
{
  "refresh_token": "opaque-token"
}
```

### Logout

Send `Authorization: Bearer <access_token>` and:

```json
{
  "refresh_token": "opaque-token"
}
```

### Get Me

Send `Authorization: Bearer <token>`.

The profile payload can now include:

```json
{
  "role": "user | organizer_admin | super_admin",
  "organizer_id": "uuid-or-null"
}
```

### Create Category

Super admin only.

```json
{
  "name": "Cinema",
  "slug": "cinema",
  "image_url": "https://example.com/cinema.jpg"
}
```

### Create Event

Organizer admin or super admin.

`organizer_id` is optional for organizer admins because it is derived from their token.
Super admins must send the target `organizer_id`.

```json
{
  "organizer_id": "6e10d730-d4d0-4c3b-bef0-b46913cd22b6",
  "category_id": "5d8e6e3f-07d5-45ce-b0ea-cfe4a2434247",
  "title": "Lagos Tech Expo 2026",
  "slug": "lagos-tech-expo-2026",
  "description": "A full-day conference for builders, founders, and product teams.",
  "venue_name": "Landmark Centre",
  "venue_address": "Water Corporation Drive, Victoria Island",
  "city": "Lagos",
  "country": "Nigeria",
  "banner_url": "https://example.com/banner.jpg",
  "poster_url": "https://example.com/poster.jpg",
  "status": "draft",
  "currency": "NGN",
  "is_featured": false
}
```

### List Events

Send `Authorization: Bearer <token>`.

- organizer admins only see events that belong to their organizer.
- super admins can see all events.

### Get Event

Send `Authorization: Bearer <token>`.

- organizer admins can only fetch events that belong to their organizer.
- super admins can fetch any event.

## Local setup

1. Install Go 1.22+
2. Install MySQL and create the `eventy` database
3. Apply [db/migrations/000001_create_users.up.sql](C:\Users\SKYMIL\Documents\Playground\eventy-api\db\migrations\000001_create_users.up.sql)
4. Apply [db/migrations/000002_create_pending_registrations.up.sql](C:\Users\SKYMIL\Documents\Playground\eventy-api\db\migrations\000002_create_pending_registrations.up.sql)
5. Apply [db/migrations/000003_create_password_reset_tokens.up.sql](C:\Users\SKYMIL\Documents\Playground\eventy-api\db\migrations\000003_create_password_reset_tokens.up.sql)
6. Apply [db/migrations/000004_create_auth_sessions.up.sql](C:\Users\SKYMIL\Documents\Playground\eventy-api\db\migrations\000004_create_auth_sessions.up.sql)
7. Apply [db/migrations/000005_create_organizers_and_user_ownership.up.sql](C:\Users\SKYMIL\Documents\Playground\eventy-api\db\migrations\000005_create_organizers_and_user_ownership.up.sql)
8. Apply [db/migrations/000006_create_categories_events_sessions_ticket_types.up.sql](C:\Users\SKYMIL\Documents\Playground\eventy-api\db\migrations\000006_create_categories_events_sessions_ticket_types.up.sql)
9. Copy `.env.example` to `.env`
10. Configure MySQL and Mailjet values in `.env`
11. Run `go mod tidy`
12. Optional: run `sqlc generate`
13. Run `go run ./cmd/api`

## Notes

- `register` now stores a pending signup, sends a 6-digit OTP email through Mailjet, and only creates the user after `/register/verify`.
- `login`, `/register/verify`, and `/refresh` now return short-lived access tokens plus DB-backed refresh tokens.
- `logout` now revokes the submitted refresh-token session.
- platform roles are now `user`, `organizer_admin`, and `super_admin`.
- organizer admins are linked to an `organizer_id`, and reusable role middleware is available for upcoming admin/events routes.
- categories are now available as a public listing plus super-admin creation API.
- event creation/list/detail APIs are now available for organizer admins and super admins with organizer ownership checks.
- event, event-session, and ticket-type tables are created as the schema foundation for the next module step.
- A lightweight `sqlc`-compatible query package is included under `internal/platform/db/sqlc` so the module is usable before code generation is wired on your machine.
