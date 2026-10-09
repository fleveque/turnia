# Decisions

Architecture decision records: one file per decision that had alternatives worth writing down.
A decision that changes gets a new record that supersedes the old one; old records stay.

| # | Decision | Status |
|---|---|---|
| [0001](0001-stack.md) | Go standard library, pgx, sqlc and goose; React PWA with Vite | accepted |
| [0002](0002-shared-schema-with-row-level-security.md) | One schema for all pharmacies, isolated by row-level security | accepted |
| [0003](0003-auth-tokens-on-one-origin.md) | Passwordless sign-in; short-lived access tokens, rotated refresh cookies, one origin | accepted |
| [0004](0004-one-hetzner-vm.md) | Everything on one Hetzner VM with Docker Compose | accepted |
| [0005](0005-audit-shifts-with-a-database-trigger.md) | Shift changes audited by a database trigger | accepted |
| [0006](0006-agpl-3.0.md) | AGPL-3.0 | accepted |
| [0007](0007-shift-swaps-between-employees.md) | Shift swaps between employees, both accepting | accepted |
| [0008](0008-platform-staff-and-aggregate-stats.md) | Turnia staff are separate, and see aggregates only | accepted |
