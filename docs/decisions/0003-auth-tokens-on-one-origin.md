# 0003 — Short-lived access tokens, rotated refresh cookies, one origin

**Status:** accepted · **Date:** 2026-10-09

## Context

Employees open the app from a phone home screen, often days apart, and expect to still be signed
in. The app is a single-page PWA talking to a JSON API. Tokens stored where JavaScript can read
them (localStorage) are stolen by any XSS; cookies sent to another site are increasingly blocked,
especially by Safari on iOS — where most employees will be.

## Decision

- **Access token**: a JWT (HS256) valid for 15 minutes, carrying the user id, the pharmacy id and
  the role. Returned in the response body; the frontend keeps it **in memory only** and sends it as
  `Authorization: Bearer`.
- **Refresh token**: 32 random bytes, stored in the database only as a SHA-256 hash, valid for 30
  days, **rotated on every use** (using an old one revokes the family). Delivered as a cookie:
  `HttpOnly; Secure; SameSite=Strict; Path=/api/v1/auth`.
- **One origin**: the PWA and the API are served by the same host (Caddy, ADR 0004), the API under
  `/api`. The cookie is first-party, `SameSite=Strict` works, and production has no CORS.
- Passwords hashed with **argon2id**. Login is rate-limited per IP.

## Alternatives

- **Sessions in the database, cookie only** — simpler and revocable, but every request hits the
  session table; the JWT keeps the hot path (reading shifts) to one query. Revocation is handled at
  refresh time, which with 15-minute access tokens is acceptable.
- **Long-lived JWT in localStorage** — readable by any injected script, unrevocable.
- **API on a separate domain (e.g. a hosted frontend)** — needs CORS and cross-site cookies, which
  iOS Safari blocks by default.

## Consequences

- On load, the app calls `/auth/refresh` to get its first access token; a 401 triggers one refresh
  and a retry.
- Local development runs Vite's dev server proxying `/api` to the Go server, so even development is
  one origin.
