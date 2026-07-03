-- Normalize every foreign-key column from text → uuid to match the parents.
--
-- Symptom this fixes: SELECT / DELETE / JOIN queries that compare a child's
-- FK column against a parent's id error with "operator does not exist:
-- text = uuid" and need an explicit cast. Discovered via
-- scripts/smoke-test.sh cleanup logic.
--
-- Root cause: parent primary keys carry gorm:"type:uuid;primaryKey" but
-- child FK columns were declared as plain `SomeID string` without a type
-- hint, so GORM created them as text. Every value already conforms to
-- UUID shape (verified pre-migration), so the ::uuid cast succeeds.
--
-- Idempotent: uses ALTER COLUMN ... TYPE which is a no-op if the column
-- is already uuid.

BEGIN;

-- ─── identity-service ────────────────────────────────────────────────────
ALTER TABLE users
  ALTER COLUMN community_id TYPE uuid USING community_id::uuid;

ALTER TABLE refresh_tokens
  ALTER COLUMN user_id   TYPE uuid USING user_id::uuid,
  ALTER COLUMN family_id TYPE uuid USING family_id::uuid;

-- ─── community-service ───────────────────────────────────────────────────
ALTER TABLE communities
  ALTER COLUMN created_by_id TYPE uuid USING created_by_id::uuid;

ALTER TABLE issues
  ALTER COLUMN reported_by_id TYPE uuid USING reported_by_id::uuid;

ALTER TABLE issue_comments
  ALTER COLUMN author_id TYPE uuid USING author_id::uuid;

ALTER TABLE issue_upvotes
  ALTER COLUMN issue_id TYPE uuid USING issue_id::uuid,
  ALTER COLUMN user_id  TYPE uuid USING user_id::uuid;

ALTER TABLE petitions
  ALTER COLUMN created_by_id TYPE uuid USING created_by_id::uuid;

ALTER TABLE petition_comments
  ALTER COLUMN petition_id TYPE uuid USING petition_id::uuid,
  ALTER COLUMN author_id   TYPE uuid USING author_id::uuid;

ALTER TABLE petition_signatures
  ALTER COLUMN petition_id TYPE uuid USING petition_id::uuid,
  ALTER COLUMN user_id     TYPE uuid USING user_id::uuid;

ALTER TABLE representatives
  ALTER COLUMN community_id  TYPE uuid USING community_id::uuid,
  ALTER COLUMN created_by_id TYPE uuid USING created_by_id::uuid;

ALTER TABLE representative_comments
  ALTER COLUMN representative_id TYPE uuid USING representative_id::uuid,
  ALTER COLUMN author_id         TYPE uuid USING author_id::uuid;

ALTER TABLE representative_followers
  ALTER COLUMN representative_id TYPE uuid USING representative_id::uuid,
  ALTER COLUMN user_id           TYPE uuid USING user_id::uuid;

ALTER TABLE notifications
  ALTER COLUMN user_id TYPE uuid USING user_id::uuid;

-- ─── organization-service ────────────────────────────────────────────────
ALTER TABLE organizations
  ALTER COLUMN created_by_id TYPE uuid USING created_by_id::uuid;

ALTER TABLE org_members
  ALTER COLUMN organization_id TYPE uuid USING organization_id::uuid,
  ALTER COLUMN user_id         TYPE uuid USING user_id::uuid;

ALTER TABLE announcements
  ALTER COLUMN organization_id TYPE uuid USING organization_id::uuid,
  ALTER COLUMN author_id       TYPE uuid USING author_id::uuid;

ALTER TABLE projects
  ALTER COLUMN organization_id TYPE uuid USING organization_id::uuid,
  ALTER COLUMN community_id    TYPE uuid USING community_id::uuid,
  ALTER COLUMN created_by_id   TYPE uuid USING created_by_id::uuid;

ALTER TABLE issue_assignments
  ALTER COLUMN organization_id TYPE uuid USING organization_id::uuid,
  ALTER COLUMN issue_id        TYPE uuid USING issue_id::uuid,
  ALTER COLUMN assigned_by_id  TYPE uuid USING assigned_by_id::uuid;

ALTER TABLE progress_updates
  ALTER COLUMN organization_id TYPE uuid USING organization_id::uuid,
  ALTER COLUMN issue_id        TYPE uuid USING issue_id::uuid,
  ALTER COLUMN project_id      TYPE uuid USING project_id::uuid,
  ALTER COLUMN author_id       TYPE uuid USING author_id::uuid;

COMMIT;
