---
id: database
title: Database
sidebar_position: 1
---

# Database

Postgres 16, single instance shared by all four Go services.

## Why one DB, not one per service?

The Engineering Playbook calls for **database-per-service**. The MVP
uses a **shared database with logically-separate tables** — each service
owns its tables, no service reads or writes another service's tables
except:

- The `audit_logs` table is written by identity, community, **and**
  organization services (schema lives in identity's
  `internal/domain/models.go`).
- Some cross-entity queries in admin metrics (`SELECT COUNT(*) FROM
issues WHERE …`) reach across service boundaries because the admin
  console needs one aggregated view.

Splitting per-service adds real complexity (backups, migrations,
inter-service reads via HTTP). We'll do it when scale demands it.

## Schema management — GORM AutoMigrate

Every service calls `db.AutoMigrate(&Model{}, …)` in `cmd/server/main.go`
on startup. GORM inspects the models and issues idempotent DDL:

- Adds new columns.
- Adds new indexes.
- Never drops columns (safe by default — you must drop by hand).
- Never renames columns (safe by default — a rename is add + copy + drop).

**When AutoMigrate is enough:** additive changes only. New table, new
column with a safe default, new index.

**When you need a real migration:** anything destructive — dropping
columns, renaming, backfilling data, tightening a constraint on
existing rows. Write a `.sql` file into the service's `migrations/`
folder and apply it manually before deploying.

## Connection setup

Every service has `pkg/database/postgres.go` that:

- Opens a GORM connection with `postgres.Open(DATABASE_URL)`.
- Sets sensible pool params.
- Wraps a small helper to fail-fast if the connection isn't reachable at
  boot.

## Model conventions

- **UUID primary keys.** `ID string \`gorm:"type:uuid;primaryKey"\``.
- **`created_at` / `updated_at`** on entities (auto-populated by GORM).
- **JSON-serialised slice columns** for `image_urls`, `proof_urls`,
  etc. — `\`gorm:"type:jsonb;serializer:json"\``.
- **Compound unique indexes** for "one-per" invariants
  (`issue_upvotes.(issue_id, user_id)`,
  `petition_signatures.(petition_id, user_id)`,
  `content_flags.(content_type, content_id, reporter_id)`).
- **Soft-delete columns** for user PII (`banned_at`, `deleted_at`).
  We do **not** use GORM's `gorm.DeletedAt` — we want the row visible
  in queries with an explicit state flag rather than silently filtered.

## Tables by owner

| Owner service        | Tables                                                                                                                                                                                                         |
| -------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| identity-service     | `users`, `user_community_memberships`, `refresh_tokens`, `audit_logs`, `content_flags`, `representative_applications`, `organization_applications`, `application_review_events`                                |
| community-service    | `communities`, `issues`, `issue_comments`, `issue_upvotes`, `petitions`, `petition_signatures`, `petition_comments`, `representatives`, `representative_followers`, `representative_comments`, `notifications` |
| organization-service | `organizations`, `org_members`, `announcements`, `projects`, `issue_assignments`, `progress_updates`                                                                                                           |

## Local development

- Postgres runs in Docker (`infrastructure/docker-compose.yml`).
- Host port `5433` maps to container port `5432` — the non-default port
  keeps local dev out of the way of any host Postgres.
- Volume is named `postgres_data` — `docker compose down -v` wipes it.

Connect from the host with:

```bash
docker exec -it civicos_postgres psql -U civicos -d civicos
```

Or with any client at `postgresql://civicos:civicos@localhost:5433/civicos`.

## Reset in dev

```bash
# Nuclear
docker compose -f infrastructure/docker-compose.yml down -v
docker compose -f infrastructure/docker-compose.yml up -d

# Then restart every Go service — AutoMigrate rebuilds the schema.
```

## Production

- Render provisions a managed Postgres 16.
- The connection string is set by Render into `DATABASE_URL` on each
  service.
- Backups are Render's daily snapshots (7-day retention on the smallest
  plans).
- **Do not** rely on AutoMigrate for destructive changes in production.
  Push a `.sql` migration first and apply it manually before deploying
  the code that expects the new shape.

Next: [Events](./events.md).
