---
id: database
title: Database
sidebar_position: 1
---

# Database

Postgres 16, single instance shared by every Go service that stores data (identity, community and organization; civicai-service holds no tables of its own).

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
columns, renaming, backfilling data, tightening a constraint on existing
rows, or adding a partial/unique index.

Those live in **`services/identity-service/internal/migrate/migrations/`**
as goose `.sql` files, and they run **automatically, in-process, at
identity-service boot** — before its own `AutoMigrate`. They are not
applied by hand.

### Why identity-service owns migrations for everybody

All services share one database, so migrations need exactly one owner —
three services racing to migrate at startup would contend over a single
version table. identity-service owns `users`, which everything else
references, and it runs on a plan that never sleeps.

A Postgres advisory lock serialises concurrent deploys: two instances
starting together cannot both migrate, and the second finds the work done
and carries on. A failed migration stops the service rather than letting
it serve against a half-changed schema.

They run in-process rather than as a deploy hook because the images are
distroless — no shell, nothing to run a command with. In-process needs
none of that and behaves identically on a laptop and on Cloud Run.

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

| Owner service        | Tables                                                                                                                                                                                                                                                                                                                           |
| -------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| identity-service     | `users`, `user_community_memberships`, `refresh_tokens`, `audit_logs`, `content_flags`, `representative_applications`, `organization_applications`, `application_review_events`, plus `schema_migrations` (goose's version table)                                                                                                |
| community-service    | `communities`, `issues`, `issue_comments`, `issue_upvotes`, `petitions`, `petition_signatures`, `petition_comments`, `representatives`, `representative_followers`, `representative_comments`, `representative_announcements`, `representative_announcement_comments`, `notifications`                                           |
| organization-service | `organizations`, `org_members`, `announcements`, `projects`, `issue_assignments`, `progress_updates`, `consultations`, `consultation_questions`, `consultation_responses`, `consultation_answers`, `consultation_outcomes`, `campaigns`, `milestones`, `donations`, `webhook_events`, `spend_records`, `reconciliation_findings` |
| civicai-service      | none — holds no tables of its own                                                                                                                                                                                                                                                                                                |

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

# Then restart the Go services — AutoMigrate rebuilds the schema.
```

:::note Boot order after a full wipe
identity-service's migrations operate on tables that **other** services
own — `petition_signatures`, `issues`, `representatives`, `organizations`
and more. Nothing orders the three services, so on an empty database
identity can reach a migration whose subject does not exist yet.

It now detects this and stops with a message naming the missing tables and
the service that creates each one, instead of a bare
`relation "..." does not exist`. Nothing is applied and no version is
recorded, so it retries cleanly: start the named services, restart
identity-service, and it proceeds.

It deliberately **fails rather than skipping**. An `IF EXISTS` guard would
let a migration record itself as applied while doing nothing, and goose
would never revisit it — permanently losing the `ON DELETE` policies in
`00005` and the partial indexes in `00006`–`00008`, none of which any GORM
model declares.

identity-service's own tables are handled by ordering rather than by
waiting: against an empty database it runs `AutoMigrate` **before** the
migrations, because `users` is its own table and no other service will
ever create it. On an established database the original order holds —
migrations first, so `AutoMigrate` never meets a column type a migration
is about to change.
:::

## Production

- Cloud SQL runs a managed Postgres 16 (`db-f1-micro`, europe-west1).
- Cloud Run reaches it over the mounted socket rather than an address —
  there is no static egress IP — so `DATABASE_URL` uses
  `host=/cloudsql/PROJECT:REGION:INSTANCE` and each service needs
  `--add-cloudsql-instances`.
- Cloud SQL takes automated daily backups. Check the retention on the
  current tier before relying on it for compliance.
- **Do not** rely on AutoMigrate for destructive changes in production.
  Add a goose migration under
  `services/identity-service/internal/migrate/migrations/` — it applies
  itself at the next identity-service boot, behind the advisory lock, and
  is recorded in `schema_migrations`.

Next: [Events](./events.md).
