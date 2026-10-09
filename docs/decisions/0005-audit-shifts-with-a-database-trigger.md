# 0005 — Shift changes audited by a database trigger

**Status:** accepted · **Date:** 2026-10-09

## Context

"Who changed my Thursday?" is a question the spreadsheet can't answer and Turnia must (design N2).
The record has to be complete: an audit log that some code path forgot to write is worse than none,
because it is trusted.

## Decision

- A `shift_events` table: shift id, pharmacy id, actor id, action (`created`, `updated`,
  `deleted`), the row before and after as `jsonb`, and the time.
- Written by an `AFTER INSERT OR UPDATE OR DELETE` **trigger on `shifts`**, not by Go code.
- The actor comes from a transaction setting, `app.user_id`, which `db.WithTenant` sets next to
  `app.pharmacy_id`. If it is missing, the trigger raises an error and the change is rolled back:
  a shift cannot change anonymously.
- `turnia_app` has `INSERT` and `SELECT` on `shift_events`, never `UPDATE` or `DELETE`. Rows are
  isolated by pharmacy like every other table (ADR 0002).
- The same trigger sets `shifts.last_changed_at` when the times, assignee or type of an existing
  shift change; the API derives the *Modificado* badge from it.
- It ships right after the shifts table, before any real data exists: history can't be backfilled.

## Alternatives

- **Write events from the Go service** — easy to read in Go, but one new endpoint, a script or a
  manual fix in `psql` can change shifts without a trace.
- **Generic audit extensions (e.g. pgAudit)** — log statements, not business events, and aren't
  queryable per shift by the application.

## Consequences

- Behaviour that lives in SQL is less visible to someone reading the Go code; the shift package
  documents it, and tests assert it ("an update without a record is impossible").
- Every shift write pays for one extra insert, which is negligible at this scale.
