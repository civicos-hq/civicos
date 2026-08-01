-- +goose Up
-- +goose StatementBegin

-- Normalises FK columns from text to uuid.
--
-- Previously an orphan file in infrastructure/migrations/ that nothing ran.
-- It was written to be idempotent: ALTER COLUMN ... TYPE uuid is a no-op on
-- a column that is already uuid, so this is safe whether or not it was
-- applied by hand.

BEGIN;
ALTER TABLE users
  ALTER COLUMN community_id TYPE uuid USING community_id::uuid;
ALTER TABLE refresh_tokens
  ALTER COLUMN user_id   TYPE uuid USING user_id::uuid,
  ALTER COLUMN family_id TYPE uuid USING family_id::uuid;
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

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Not reversible. Going back to text would silently accept malformed ids
-- again, which is the problem this fixed.
SELECT 1;
-- +goose StatementEnd
