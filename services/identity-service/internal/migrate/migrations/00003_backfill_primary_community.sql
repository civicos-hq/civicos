-- +goose Up
-- +goose StatementBegin

-- Backfills primary_community_id for users who predate the column.
--
-- This ran on EVERY identity-service boot, inside its database connection
-- code, because there was nowhere else to put it. It is idempotent, so it
-- was harmless — but it is a one-off data fix wearing the costume of
-- connection logic, and it re-ran forever with no record that it had ever
-- succeeded.
--
-- Their active community becomes their primary: the safest guess, and the
-- one that leaves current behaviour unchanged.
UPDATE users
   SET primary_community_id = community_id
 WHERE primary_community_id IS NULL
   AND community_id IS NOT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Not reversible: the pre-backfill NULLs are not recoverable, and restoring
-- them would log people out of their own community for no benefit.
SELECT 1;
-- +goose StatementEnd
