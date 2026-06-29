-- Migration: add unique index to prevent duplicate petition signatures
-- Up: create unique index
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS idx_petition_user ON petition_signatures (petition_id, user_id);

-- Down (manual rollback reference):
-- DROP INDEX CONCURRENTLY IF EXISTS idx_petition_user;
