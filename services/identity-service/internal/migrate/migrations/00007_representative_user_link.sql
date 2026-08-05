-- +goose Up
-- +goose StatementBegin

-- Which USER a representative profile belongs to.
--
-- `created_by_id` already exists but means "who created this row", which is
-- not the same thing and in practice is not the representative at all: every
-- profile seeded or minted by an admin points at that admin. Using it as an
-- ownership check would let one platform admin publish in the name of every
-- representative on the platform, while no actual representative could
-- publish anything.
--
-- Nullable on purpose. A profile can exist before anyone claims it — an admin
-- correcting data, or a seeded constituency — and an unclaimed profile simply
-- has nobody who may publish as it.
ALTER TABLE representatives
  ADD COLUMN IF NOT EXISTS user_id uuid;

-- One account per profile. Two people sharing a representative's voice would
-- make the audit trail meaningless: "the representative said this" has to
-- name one person.
CREATE UNIQUE INDEX IF NOT EXISTS idx_representative_user
  ON representatives (user_id)
  WHERE user_id IS NOT NULL;

-- Backfill only where the creator is demonstrably the representative — i.e.
-- the profile came from that person's own approved application. Anything
-- created by an admin is left unclaimed rather than guessed at, because a
-- wrong guess here hands someone else's constituents to the wrong account.
UPDATE representatives r
   SET user_id = r.created_by_id
  FROM users u
 WHERE u.id = r.created_by_id
   AND u.role = 'REPRESENTATIVE'
   AND r.user_id IS NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_representative_user;
ALTER TABLE representatives DROP COLUMN IF EXISTS user_id;
-- +goose StatementEnd
