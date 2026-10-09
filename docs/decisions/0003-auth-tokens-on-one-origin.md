# 0003 — Passwordless sign-in; short-lived access tokens and rotated refresh cookies on one origin

**Status:** accepted · **Date:** 2026-10-09

## Context

Employees open Turnia from a phone home screen, often days apart, and expect to still be signed in.
Many will share a counter computer. Nobody in a pharmacy wants another password, and a password
brings a reset flow, hash storage and "I forgot it" support that a small product would rather not
own. quantic.finance signs people in with magic links and Google, and that works well there.

Turnia has a constraint quantic doesn't: it is used as an **installed PWA on iPhones**. On iOS, an
app added to the home screen has its own cookies and storage, separate from Safari's, and a link
tapped in the Mail app opens in Safari. A magic link alone signs the person into Safari while the
installed app stays signed out.

Tokens stored where JavaScript can read them (localStorage) are stolen by any XSS; cookies sent to
another site are increasingly blocked, especially by Safari on iOS.

## Decision

**Signing in — one email, two ways to use it.**

- The person enters their email; Turnia sends one message with a **6-digit code** and a **link**.
  In a browser, the link is one tap. In the installed app, or on another device, they type the
  code. Both are single-use, valid for 15 minutes, and the code allows 5 attempts.
- Codes and link tokens are stored only as hashes (SHA-256), in a `login_codes` table.
- **The response never says whether the email exists** ("if this address has an account, we've sent
  a code"), so the form can't be used to discover who works where.
- Rate limits: per email (a few codes per hour) and per IP.
- **Invite-only.** Registering creates a pharmacy and its first admin; everyone else is invited by
  an admin. The invitation is the same email with a longer-lived link (7 days); accepting it is the
  first sign-in.
- Turnia staff (ADR 0008) sign in the same way at their own endpoint.
- Email goes through **Resend** in production (as quantic) and **Mailpit**, a local mail catcher, in
  development.

**Sessions — unchanged by how you signed in.**

- **Access token**: a JWT (HS256) valid for 15 minutes, carrying the user id, the pharmacy id and
  the role. Returned in the response body; the frontend keeps it **in memory only** and sends it as
  `Authorization: Bearer`.
- **Refresh token**: 32 random bytes, stored only as a SHA-256 hash, valid for **90 days**, **rotated
  on every use** (using an old one revokes the whole family). Delivered as a cookie:
  `HttpOnly; Secure; SameSite=Strict; Path=/api/v1/auth`. A phone stays signed in for months
  without a stored password; signing out on a shared computer revokes it.
- **One origin**: the PWA and the API are served by the same Go binary (ADR 0004), the API under
  `/api`. The cookie is first-party, `SameSite=Strict` works, and production has no CORS.

**Google, later, without a second account.** The account is the `users` row; the email is how you
reach it. Google sign-in, when it comes, adds a `user_identities` table (provider, the provider's
account id) and links to the existing user:

- a Gmail or Workspace address that Google reports as verified links directly (Google is the
  authority for those);
- any other address on a Google account must confirm with an email code once before linking;
- an address with no Turnia account is refused (invite-only), never created;
- after linking, sign-in matches on Google's account id, not the email.

It is its own milestone after the MVP, once it has been tried in the installed app on a real
iPhone, where the OAuth redirect has the same storage problem as magic links.

## Alternatives

- **Email and password** — no email provider needed to sign in, but passwords to hash, reset and
  forget; and the reset flow needs email anyway.
- **Magic link only** — breaks in the installed iOS app, as above.
- **Google only** — not everyone has, or wants to use, a Google account at work.
- **Sessions in the database, cookie only** — simpler and revocable, but every request hits the
  session table; the JWT keeps the hot path (reading shifts) to one query. Revocation happens at
  refresh time, which with 15-minute access tokens is acceptable.
- **Long-lived JWT in localStorage** — readable by any injected script, unrevocable.
- **API on a separate domain** — needs CORS and cross-site cookies, which iOS Safari blocks by
  default.

## Consequences

- Email is infrastructure from the first auth milestone: an outage at the provider means nobody new
  can sign in (people already signed in are unaffected for up to 90 days).
- On load, the app calls `/auth/refresh` to get its first access token; a 401 triggers one refresh
  and a retry.
- Local development runs Vite's dev server proxying `/api` to the Go server, so even development is
  one origin, and Mailpit shows every email sent.
