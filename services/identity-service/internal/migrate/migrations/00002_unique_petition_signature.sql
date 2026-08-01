-- +goose Up
-- +goose NO TRANSACTION

-- Prevents a petition being signed twice by the same person.
--
-- Previously an orphan .sql file in services/community-service/migrations/
-- that nothing ever executed — whether it reached production was unknowable
-- from the repo. IF NOT EXISTS makes it safe either way.
--
-- CONCURRENTLY cannot run inside a transaction, hence NO TRANSACTION. It
-- also means the index build does not lock out signing while it runs.
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS idx_petition_user
  ON petition_signatures (petition_id, user_id);

-- +goose Down
-- +goose NO TRANSACTION
DROP INDEX CONCURRENTLY IF EXISTS idx_petition_user;
