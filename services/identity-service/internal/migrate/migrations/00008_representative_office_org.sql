-- +goose Up
-- +goose StatementBegin

-- Which REPRESENTATIVE profile a constituency office belongs to.
--
-- An elected representative publishes campaigns, projects, consultations
-- and announcements through an organization of kind REPRESENTATIVE_OFFICE
-- rather than through a parallel set of rep-owned tables. That keeps the
-- whole money path — sub-account, split, ledger, reconciliation, payout,
-- admin review — untouched, instead of growing a second owner type through
-- code that moves donations.
--
-- AutoMigrate adds the column but cannot express the constraint below, so
-- both live here.
ALTER TABLE organizations
  ADD COLUMN IF NOT EXISTS representative_id uuid;

-- One office per representative, enforced at the database rather than in
-- the provisioning endpoint alone.
--
-- Provisioning is idempotent and checks first, but "check then insert" is
-- not atomic: two requests racing — a double-clicked button, a client
-- retry — both see no office and both create one. The loser of that race
-- must fail, because two offices for the same official means two campaigns
-- soliciting from the same constituents under names nobody can tell apart,
-- and donations split across ledgers that never reconcile.
--
-- Partial, because every organization that is not an office leaves this
-- column NULL and they must not collide with each other.
CREATE UNIQUE INDEX IF NOT EXISTS idx_organization_representative
  ON organizations (representative_id)
  WHERE representative_id IS NOT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_organization_representative;
ALTER TABLE organizations DROP COLUMN IF EXISTS representative_id;
-- +goose StatementEnd
