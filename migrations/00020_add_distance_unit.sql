-- +goose Up
ALTER TABLE activity_log ADD COLUMN distance_unit varchar(10) NOT NULL DEFAULT 'miles';

-- +goose Down
ALTER TABLE activity_log DROP COLUMN distance_unit;
