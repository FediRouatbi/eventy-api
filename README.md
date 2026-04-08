# eventy-api

Eventy API built with Go, Chi, MySQL, sqlc, and JWT.

## API docs

- `GET /openapi.json`
- `GET /docs`

Frontend and mobile can use `/openapi.json` to generate typed API clients.
`/docs` now serves Scalar interactive API documentation.

## Implemented module

- super admin organizer-admin creation
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
- `PATCH /v1/auth/change-password`
- `POST /v1/auth/logout`

## User endpoint

- `GET /v1/users/me`

## Category endpoints

- `GET /v1/categories`
- `POST /v1/categories`

## Public event endpoints

- `GET /v1/public/events`
- `GET /v1/public/events/{eventID}`

## Admin endpoints

- `GET /v1/admins/organizers`
- `POST /v1/admins/organizers`
- `GET /v1/admins/organizers/{organizerID}`
- `GET /v1/admins/organizers/{organizerID}/admin`
- `PATCH /v1/admins/organizers/{organizerID}/admin/password`

## Event endpoints

- `GET /v1/events`
- `POST /v1/events`
- `GET /v1/events/{eventID}`
- `GET /v1/events/{eventID}/sessions`
- `POST /v1/events/{eventID}/sessions`
- `PATCH /v1/events/{eventID}/sessions/{sessionID}`
- `GET /v1/events/{eventID}/sessions/{sessionID}/ticket-types`
- `POST /v1/events/{eventID}/sessions/{sessionID}/ticket-types`
- `PATCH /v1/events/{eventID}/sessions/{sessionID}/ticket-types/{ticketTypeID}`

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

### Change Password

Send `Authorization: Bearer <access_token>` and:

```json
{
  "current_password": "secret123",
  "new_password": "newSecret123"
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
  "description": "Curated screenings, premieres, and cultural film nights.",
  "image_url": "https://example.com/cinema.jpg"
}
```

### Create Organizer Admin

Super admin only.

This creates both the organizer record and its linked `organizer_admin` account in one request.

```json
{
  "organizer_name": "Eventy Lagos",
  "organizer_slug": "eventy-lagos",
  "admin_name": "Lagos Organizer Admin",
  "admin_email": "organizer@example.com",
  "admin_password": "secret123"
}
```

The `admin_password` in this request is the initial password you hand over to that organizer admin.
They use it with `POST /v1/auth/login`, and once they sign in their token includes the `organizer_admin` role plus their `organizer_id`.
That means the dashboard can use the existing event endpoints and they will only see and manage events that belong to their own organizer.

There is no shared default password for all admins.
Each organizer admin gets the exact password you send in `admin_password` at creation time, and super admins can later reset it through the password-reset admin route.

### Get Organizer Admin

Super admin only.

This returns the single `organizer_admin` account attached to the organizer.

### Get Organizer

Super admin only.

This is the best endpoint to call when someone clicks an organizer in the dashboard list.
It returns the organizer row, its linked admin account, and organizer-scoped event summaries in one response.

### List Organizers

Super admin only.

This list now returns each organizer row together with its linked admin account plus `event_count` and `session_count`, so the admin dashboard can render the organizer table from a single request.

### Reset Organizer Admin Password

Super admin only.

```json
{
  "password": "newSecret123"
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
  "latitude": 6.4281,
  "longitude": 3.4219,
  "banner_url": "https://example.com/banner.jpg",
  "poster_url": "https://example.com/poster.jpg",
  "status": "draft",
  "currency": "NGN",
  "is_featured": false
}
```

`latitude` and `longitude` are optional, but they must be sent together if you want the event to render with an exact Leaflet map pin.

### Create Event Session

Organizer admin or super admin.

```json
{
  "starts_at": "2026-06-21T09:00:00Z",
  "ends_at": "2026-06-21T17:00:00Z",
  "sales_starts_at": "2026-05-01T00:00:00Z",
  "sales_ends_at": "2026-06-20T23:59:59Z",
  "status": "scheduled"
}
```

### Create Ticket Type

Organizer admin or super admin.

```json
{
  "name": "Early Bird",
  "description": "Discounted ticket for early buyers",
  "price": 15000,
  "quantity": 100,
  "max_per_order": 4
}
```

### List Events

Send `Authorization: Bearer <token>`.

- organizer admins only see events that belong to their organizer.
- super admins can see all events.

### List Public Events

This route is public and returns published events enriched for discovery pages.
The payload includes organizer and category display fields plus session-derived values like:

- `organizer_name`
- `category_name`
- `category_slug`
- `summary`
- `next_session_starts_at`
- `price_from`
- `tickets_left`

### Get Public Event

This route is public and returns a published event with its scheduled sessions and ticket types.

### Get Event

Send `Authorization: Bearer <token>`.

- organizer admins can only fetch events that belong to their organizer.
- super admins can fetch any event.
- event detail now includes nested sessions and ticket types for dashboard configuration.

## Local setup

1. Install Go 1.22+
2. Install MySQL and create the `eventy` database
3. Apply [db/migrations/000001_create_users.up.sql](C:\Users\SKYMIL\Documents\Playground\eventy-api\db\migrations\000001_create_users.up.sql)
4. Apply [db/migrations/000002_create_pending_registrations.up.sql](C:\Users\SKYMIL\Documents\Playground\eventy-api\db\migrations\000002_create_pending_registrations.up.sql)
5. Apply [db/migrations/000003_create_password_reset_tokens.up.sql](C:\Users\SKYMIL\Documents\Playground\eventy-api\db\migrations\000003_create_password_reset_tokens.up.sql)
6. Apply [db/migrations/000004_create_auth_sessions.up.sql](C:\Users\SKYMIL\Documents\Playground\eventy-api\db\migrations\000004_create_auth_sessions.up.sql)
7. Apply [db/migrations/000005_create_organizers_and_user_ownership.up.sql](C:\Users\SKYMIL\Documents\Playground\eventy-api\db\migrations\000005_create_organizers_and_user_ownership.up.sql)
8. Apply [db/migrations/000006_create_categories_events_sessions_ticket_types.up.sql](C:\Users\SKYMIL\Documents\Playground\eventy-api\db\migrations\000006_create_categories_events_sessions_ticket_types.up.sql)
9. Apply [db/migrations/000007_add_event_coordinates.up.sql](C:\Users\SKYMIL\Documents\Playground\eventy-api\db\migrations\000007_add_event_coordinates.up.sql)
10. Apply [db/migrations/000008_add_category_description.up.sql](C:\Users\SKYMIL\Documents\Playground\eventy-api\db\migrations\000008_add_category_description.up.sql)
11. Copy `.env.example` to `.env`
12. Configure MySQL and Mailjet values in `.env`
13. Run `go mod tidy`
14. Optional: run `sqlc generate`
15. Run `go run ./cmd/api`

## Notes

- `register` now stores a pending signup, sends a 6-digit OTP email through Mailjet, and only creates the user after `/register/verify`.
- `login`, `/register/verify`, and `/refresh` now return short-lived access tokens plus DB-backed refresh tokens.
- `logout` now revokes the submitted refresh-token session.
- platform roles are now `user`, `organizer_admin`, and `super_admin`.
- organizer admins are linked to an `organizer_id`, and reusable role middleware is available for upcoming admin/events routes.
- categories are now available as a public listing plus super-admin creation API.
- categories now support optional description and image metadata for frontend discovery cards.
- super admins can now create an organizer together with its `organizer_admin` account from the API.
- event creation/list/detail APIs are now available for organizer admins and super admins with organizer ownership checks.
- public event discovery APIs are now available with organizer/category display data and session-derived pricing/availability summary fields.
- event sessions and ticket types can now be managed under events by organizer admins and super admins.
- event, event-session, and ticket-type tables are created as the schema foundation for the next module step.
- A lightweight `sqlc`-compatible query package is included under `internal/platform/db/sqlc` so the module is usable before code generation is wired on your machine.
