# 0007 — Shift swaps between employees

**Status:** accepted · **Date:** 2026-10-09

## Context

Employees already swap shifts informally ("I'll do your Saturday if you do my Tuesday") and tell the
owner afterwards, or forget to. The schedule then lies. Turnia should let them do it in the app, so
that the schedule stays true and the swap is on record — without giving employees general edit
rights over shifts, which stay with admins (design §3).

## Decision

- **A per-pharmacy setting**, `pharmacies.swaps_enabled`, off by default. An admin turns it on.
- **A swap is an exchange of two shifts between two employees.** Employee A proposes: one of A's
  shifts, one of B's shifts, an optional message. B accepts or declines; A can cancel while it is
  pending; an admin can cancel any. A pending swap expires when either shift starts.
- **Both accept, no admin approval** in this version: A by proposing, B by accepting. Admin approval
  as an extra setting is designed for (design §8), not built.
- **Accepting is one transaction**: lock the swap and both shifts (`SELECT … FOR UPDATE`), check the
  swap is still pending and both shifts still belong to A and B, exchange the assignees, mark the
  swap accepted. The overlap constraint is declared `DEFERRABLE INITIALLY IMMEDIATE` and the swap
  transaction defers it, because exchanging two overlapping shifts collides with itself halfway
  through; at commit it is checked as usual, so a swap that would double-book either person fails
  with 409 and changes nothing.
- **Audited like everything else (ADR 0005).** The shift trigger records the two assignee changes
  with the accepting employee as actor and the swap's id (read from a transaction setting,
  `app.swap_id`), so the history says *swapped*, not *edited*. A trigger on `shift_swaps` records
  each status change (proposed, accepted, declined, cancelled, expired) as an event on both shifts.
- **Shown on the shift**: an accepted swap gives both shifts an *Intercambiado* badge (with whom,
  when); a pending one, *Intercambio pendiente*. Distinct from *Modificado*, which means an admin
  edited the shift.
- Employees act on swaps only through the `/swaps` endpoints. The shift endpoints stay admin-only.

## Alternatives

- **Let employees edit their own shifts** — simple, but the owner loses control of coverage, and
  nothing guarantees the other person agreed.
- **Admin approval always** — safer, but the owner becomes a bottleneck for something the team
  already settles between themselves; kept as a future setting for pharmacies that want it.
- **Swap by deleting and recreating shifts** — breaks the shift's history in two and loses the link
  between the old and new assignment.

## Consequences

- The overlap constraint is deferrable everywhere, not only in swaps; outside a transaction that
  defers it, it behaves exactly as before.
- One more table under RLS and one more trigger, both covered by the isolation and audit tests.
