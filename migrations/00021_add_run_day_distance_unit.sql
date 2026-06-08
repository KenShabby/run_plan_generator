-- +goose Up
ALTER TABLE run_days ADD COLUMN distance_unit varchar(10) NOT NULL DEFAULT 'miles';

-- +goose Down
ALTER TABLE run_days DROP COLUMN distance_unit;
