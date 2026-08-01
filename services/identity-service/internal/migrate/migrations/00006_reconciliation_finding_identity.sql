-- +goose Up
-- +goose StatementBegin

-- Reconciliation findings: drift between our ledger and the payment provider,
-- recorded so it outlives the log line that noticed it.
--
-- This migration creates the table AND its index, rather than leaving the
-- table to organization-service's AutoMigrate.
--
-- An earlier version of this file guarded the index on the table already
-- existing, assuming it would be created on a later run once AutoMigrate had
-- caught up. That was wrong: goose marks a migration applied and never runs
-- it again, so on a fresh database the guard would skip, the migration would
-- be recorded as done, and the unique index would silently never exist —
-- leaving the sweep free to write a duplicate row every hour.
--
-- For a NEW table the migration owns creation. The AutoMigrate division of
-- labour applies to the tables that predate this tooling, not to ones
-- introduced alongside it.
CREATE TABLE IF NOT EXISTS reconciliation_findings (
  id               uuid PRIMARY KEY,
  kind             varchar(48) NOT NULL,
  donation_id      uuid NOT NULL,
  campaign_id      uuid,
  reference        varchar(100),
  amount_minor     bigint NOT NULL DEFAULT 0,
  detail           text,
  run_id           uuid,
  first_seen_at    timestamptz NOT NULL,
  last_seen_at     timestamptz NOT NULL,
  times_seen       integer NOT NULL DEFAULT 1,
  resolved_at      timestamptz,
  resolved_by_id   uuid,
  resolved_by_name text,
  resolution_note  text
);

-- One open finding per (donation, kind). The sweep re-detects every
-- unresolved disagreement on each hourly pass; without this, one stuck
-- donation would bury the finding that is actually new.
CREATE UNIQUE INDEX IF NOT EXISTS idx_reconciliation_finding_unique
  ON reconciliation_findings (donation_id, kind);

CREATE INDEX IF NOT EXISTS idx_reconciliation_finding_open
  ON reconciliation_findings (resolved_at, last_seen_at DESC);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS reconciliation_findings;
-- +goose StatementEnd
