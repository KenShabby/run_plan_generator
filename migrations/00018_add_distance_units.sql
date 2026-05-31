-- +goose Up
ALTER TABLE segments ADD COLUMN distance_unit varchar(10) NOT NULL DEFAULT 'miles';
ALTER TABLE template_segments ADD COLUMN distance_unit varchar(10) NOT NULL DEFAULT 'miles';

-- +goose Down
ALTER TABLE segments DROP COLUMN distance_unit;
ALTER TABLE template_segments DROP COLUMN distance_unit;
