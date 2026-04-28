# Eventy Mobile (Expo) — Product & Engineering Blueprint

## 1) Goal

Build `eventy-mobile` as a high-quality Expo app that brings the core Eventy experience to mobile:

- Discover events
- Reserve tickets
- Checkout and pay
- View orders, receipts, and QR tickets
- Manage account/security

This plan is based on the current `eventy-web` flows and `eventy-api` endpoints.

---

## 2) Current Platform Reality (Web + API)

Already available and reusable for mobile:

- Auth: register + verify OTP, login, refresh, logout, forgot/reset password
- Public discovery: categories, events list/detail
- Reservation/cart flow: create/update reservation, checkout-order creation, Stripe session creation
- Post-purchase: order history, order detail, tickets list (QR-ready), receipt timeline
- Account: profile update, password change
- Role model: `user`, `organizer_admin`, `super_admin`
- Admin APIs: overview, payments, CSV export, organizers, events/sessions/ticket-types

---

## 3) Mobile Product Scope

## Phase 1 (MVP — User App)

- Authentication + account recovery
- Home/discovery + search/filter
- Event detail + ticket selection
- Cart + reservation countdown
- Checkout handoff to Stripe
- Payment success state sync
- Orders list + order detail + timeline
- Tickets list + QR display
- Account profile/security

## Phase 2 (Trust & Retention)

- In-app receipt PDF download/share
- Push notifications:
  - payment success
  - upcoming event reminder
  - reservation expiring soon
- Better order timeline clarity + support entry points

## Phase 3 (Role-based Extension)

- `organizer_admin` mobile-lite mode:
  - dashboard snapshot
  - event/session quick edits
  - ticket check-in scanner (QR camera)
- Keep full super-admin operations on web initially.

---

## 4) Recommended Tech Stack

- Expo SDK (latest stable)
- TypeScript (strict)
- Expo Router (file-based navigation)
- TanStack Query (React Query) for **all** server state
- `openapi-fetch` + generated OpenAPI types for API client
- Zustand (or small context) for local app state (auth session + UI state)
- `expo-secure-store` for token/session storage
- `expo-notifications` for push
- `expo-linking` for payment return deep links
- `react-native-reanimated` + `react-native-gesture-handler` for fluid UX
- `react-native-svg` + `react-native-qrcode-svg` for ticket QR rendering
- `@shopify/flash-list` for performant long lists

---

## 5) App Architecture

Use this package layering:

1. `src/lib/api`  
   - generated schema from OpenAPI  
   - `createApiClient(accessToken?)` via `openapi-fetch`  
   - typed functions per domain (`auth`, `public`, `orders`, `tickets`, `admin`)

2. `src/lib/auth`  
   - secure storage read/write  
   - token refresh orchestrator  
   - bootstrap current user

3. `src/features/*`  
   - screen logic, components, hooks by domain

4. `src/app` (Expo Router)  
   - route groups: `(public)`, `(auth)`, `(user)`, `(admin-lite?)`

---

## 6) Suggested Folder Structure

```txt
eventy-mobile/
  app/
    (public)/
      index.tsx
      events/[eventId].tsx
      categories/[slug].tsx
    (auth)/
      login.tsx
      register.tsx
      register-verify.tsx
      forgot-password.tsx
      reset-password.tsx
    (user)/
      checkout/index.tsx
      checkout/success.tsx
      orders/index.tsx
      orders/[orderId].tsx
      tickets/index.tsx
      account/profile.tsx
      account/security.tsx
    _layout.tsx
  src/
    features/
    lib/
      api/
        generated/
        client.ts
        auth.ts
        public.ts
        reservations.ts
        orders.ts
        tickets.ts
        admin.ts
      auth/
      query/
      storage/
    components/
    theme/
```

---

## 7) Data & API Rules (Non-Negotiable)

1. **All network calls go through React Query + `openapi-fetch`.**
2. No direct `fetch` inside screens.
3. Query keys mirror domain:
   - `["public","events"]`
   - `["public","events",eventId]`
   - `["me","orders"]`
   - `["me","orders",orderId]`
   - `["me","tickets"]`
4. Optimistic update only where safe (cart quantity local + reservation confirm).
5. Retry policy:
   - GET: retry 1–2 times
   - Mutations: no blind retries for payment-sensitive operations

---

## 8) Auth & Session Flow (Mobile)

Startup:

1. Read session from `SecureStore`
2. If access token expired but refresh token valid, call `/v1/auth/refresh`
3. Hydrate user via `/v1/users/me`
4. Route by auth state + role

On 401:

- Single-flight refresh
- Replay failed request once
- If refresh fails: clear session and redirect to login

---

## 9) Checkout & Payment on Mobile (Critical)

Current backend creates Stripe Checkout URL via:

- `POST /v1/public/checkout-orders/{orderID}/stripe-session`

Mobile approach:

1. Open Checkout URL in external browser (`WebBrowser.openBrowserAsync`) or in-app browser.
2. Stripe success/cancel returns via deep link (recommended).
3. Mobile app receives deep link and navigates to `checkout/success`.
4. Confirm final status using:
   - `/v1/public/stripe-sessions/{stripeSessionID}/checkout-order` (guest context)
   - or `/v1/orders/stripe-sessions/{stripeSessionID}/checkout-order` (logged in)

### Backend update recommended for clean mobile flow

Add support for mobile success/cancel URLs during stripe session creation:

- either from env (`MOBILE_CHECKOUT_SUCCESS_URL`, `MOBILE_CHECKOUT_CANCEL_URL`)
- or request body overrides validated against allowlist

This prevents hard-coding web return URLs when checkout starts from mobile.

---

## 10) Screen-by-Screen UX Requirements

## Public

- Home: featured categories + live events feed
- Events list: fast filter chips, skeleton loading, empty state
- Event detail: session selector, ticket qty steppers, sold-out states

## Checkout

- Cart grouped by session/event
- Reservation countdown banner (10 min hold)
- Clear pricing breakdown
- Payment handoff CTA with loading/guardrails

## Post-payment

- Success page with status polling fallback
- Orders list with status chips
- Order detail timeline (created → payment → receipt/tickets)
- Download/share receipt PDF from order detail

## Tickets

- Group tickets by order/session
- Show per-ticket QR and code copy action
- Checked-in state badges

## Account

- Profile update
- Change password
- Logout all devices (future enhancement)

---

## 11) Design System (Next-Level Direction)

- Build a mobile token system: colors, spacing, radius, shadows, typography
- Use premium card surfaces and consistent section rhythm
- Keep primary actions sticky when needed (checkout/ticket actions)
- Use skeletons for every list/detail loading state
- Improve trust cues:
  - payment security indicators
  - status badges with clear color semantics
  - “last updated” timestamps in order/ticket views

---

## 12) Observability & Analytics

- Crash reporting: Sentry (Expo integration)
- Analytics events:
  - `view_event_detail`
  - `add_to_cart`
  - `start_checkout`
  - `checkout_opened`
  - `payment_success`
  - `view_ticket_qr`
  - `download_receipt_pdf`
- Log API errors with route + endpoint metadata

---

## 13) Security Checklist

- Tokens only in SecureStore
- Never persist raw card/payment data
- Mask sensitive identifiers in logs
- TLS-only base URL in production
- Validate all deep links + route params
- Use role guards for protected stacks

---

## 14) Performance Checklist

- Use `FlashList` for event/order/ticket lists
- Image caching strategy for event media
- Preload key queries on navigation transitions
- Keep bundle lean; lazy-load heavy features (PDF generation)

---

## 15) Delivery Plan (Recommended)

## Sprint 1

- Expo app scaffold + theming + routing shell
- Auth/session core + API client generation
- Public discovery (home/events/event detail)

## Sprint 2

- Reservation/cart + checkout handoff
- Payment return/deep link handling
- Orders list/detail + timeline

## Sprint 3

- Tickets (QR display), account pages
- Receipt PDF download/share
- Push notification foundations

## Sprint 4

- Polish pass (UX, empty states, skeletons, edge cases)
- Analytics + crash reporting
- Release candidate hardening

---

## 16) Open Items / Gaps to Close Before Full Launch

1. **Mobile Stripe return URL strategy** (backend config + allowlist)
2. **Push pipeline** (provider keys + notification templates)
3. **Receipt source-of-truth**
   - current web generates PDF client-side
   - decide if mobile should also generate client-side or download server-generated PDF endpoint
4. **Role scope decision for mobile v1**
   - user-only first (recommended) vs mixed user/admin app
5. **Offline policy**
   - read-only cache for events/orders yes
   - no offline checkout mutations

---

## 17) Definition of Done (Mobile v1)

- User can register/login/reset password on mobile
- User can browse events, reserve tickets, and start payment
- Payment result syncs correctly after return from Stripe
- User can see order details, timeline, and QR tickets
- User can download/share receipt PDF
- All API calls are typed via OpenAPI + `openapi-fetch`
- All server state uses React Query
- Crash-free sessions and core analytics are in place

---

## 18) Quick Start Commands (When `eventy-mobile` repo is created)

```bash
npx create-expo-app@latest eventy-mobile --template
cd eventy-mobile
npx expo install expo-router expo-secure-store expo-notifications expo-linking
npm i @tanstack/react-query openapi-fetch zustand
npm i -D openapi-typescript
```

Generate API types from Eventy API:

```bash
npx openapi-typescript http://localhost:8080/openapi.json -o src/lib/api/generated/schema.ts
```

---

If we want, next step is I create a ready-to-run `eventy-mobile` starter scaffold (folders, providers, auth bootstrap, API client, and first screens) so the team can start building immediately.
