-- +goose Up
ALTER TABLE run_days
    ALTER COLUMN total_distance TYPE DOUBLE PRECISION USING total_distance::DOUBLE PRECISION;

ALTER TABLE segments
    ALTER COLUMN distance TYPE DOUBLE PRECISION USING distance::DOUBLE PRECISION;

-- +goose Down
ALTER TABLE run_days
    ALTER COLUMN total_distance TYPE NUMERIC USING total_distance::NUMERIC;

ALTER TABLE segments
    ALTER COLUMN distance TYPE NUMERIC USING distance::NUMERIC;
