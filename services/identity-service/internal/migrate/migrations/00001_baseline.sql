-- +goose Up
-- +goose StatementBegin

-- Baseline.
--
-- Deliberately creates NOTHING. The schema up to this point was built by
-- GORM's AutoMigrate across identity, community and organization services,
-- and it already exists in every environment that matters — including
-- production, which holds live donation records.
--
-- Re-declaring ~30 tables here would be a large hand-written approximation
-- of a schema that already exists, and any drift between the two would
-- surface as a failed deploy or, worse, a silent difference between
-- environments. So this migration marks the line rather than redrawing it:
-- everything before it belongs to AutoMigrate, everything after it belongs
-- here.
--
-- AutoMigrate still runs at service boot and still creates tables and adds
-- columns. What it cannot do — drop, alter, constrain, index concurrently,
-- backfill — lives in this directory from now on, versioned and recorded.

SELECT 1;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 1;
-- +goose StatementEnd
