-- +goose Up
-- +goose StatementBegin

-- At most one PENDING invitation per (organization, email).
--
-- Without this, clicking "Invite" twice sends two live tokens for the same
-- person. Both work, both can be accepted, and the second acceptance hits
-- the already-a-member check and fails — leaving an invitation that looks
-- outstanding forever and an inviter who cannot tell which link is real.
--
-- Partial rather than plain, because the same address legitimately appears
-- many times over an organization's life: invited, left, invited again;
-- invited, revoked, invited at a different level. Only the rows that can
-- still be accepted must be unique, so accepted and revoked rows fall out
-- of the index and history is preserved.
--
-- Expiry is deliberately NOT part of the predicate. It is time-dependent,
-- and a partial index predicate must be immutable — Postgres rejects
-- now() here. An expired-but-unaccepted row therefore still occupies the
-- slot; the service treats re-inviting over one as replacing it, which is
-- what an inviter means by clicking Invite again.
CREATE UNIQUE INDEX IF NOT EXISTS idx_org_invitation_pending
  ON org_invitations (organization_id, email)
  WHERE accepted_at IS NULL AND revoked_at IS NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_org_invitation_pending;
-- +goose StatementEnd
