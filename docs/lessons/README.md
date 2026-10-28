# Lessons

Notes written while building Turnia, by someone whose daily languages are Elixir and Ruby, who
learned some Go building [quantic-agent](https://github.com/fleveque/quantic-agent-go), and who is
new to React.

Each lesson pairs with a milestone from the [roadmap](../../README.md#roadmap). They're not a Go or
React tutorial — plenty of those exist. They're the specific things that surprised me, the
assumptions I carried over from other languages that turned out to be wrong, and the reasoning
behind the choices where I found it non-obvious.

| # | Lesson | Milestone | Read online | Code walkthrough |
|---|---|---|---|---|
| [00](00-decisions-before-code.md) | Decisions before code | 0 — repository & design | [Decisions before code](https://claude.ai/artifact/SAfdpXAmgvfTBSPWkJ3ec4) | — |
| [01](01-serving-not-calling.md) | Serving, not calling | 1 — the Go server skeleton | [Serving, not calling](https://claude.ai/artifact/3Ph6s31BFKVrwfV2V5A6ot) | [The server, line by line](https://claude.ai/artifact/KSNkV2CncyoiwmY3FKYdVQ) |
| [02](02-sql-in-go-out.md) | SQL in, Go out | 2 — Postgres: pool, migrations, sqlc | [SQL in, Go out](https://claude.ai/artifact/J4eoHobirDcgXNaLggHB3U) | [02.1 migrations](https://claude.ai/artifact/A7NYjfG1awRodmvib2oAT4) · [02.2 the pool](https://claude.ai/artifact/HEhGJDFdgxbCshy6ZGXUmM) · [02.3 the queries](https://claude.ai/artifact/GTexHL9WKmrzQb4ZUnTgp1) |

Lessons tell the story of a milestone: what surprised me and why. A milestone is usually several
small pull requests, and each one with code has its own walkthrough: the code itself in reading
order, the way a tech lead would go through it with a new teammate. Walkthroughs start with
milestone 1, the first one with code. How the pages are built: [pages/](pages/README.md).
