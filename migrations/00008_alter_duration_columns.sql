-- +goose Up
ALTER TABLE run_days
    ALTER COLUMN total_duration TYPE BIGINT USING EXTRACT(EPOCH FROM total_duration)::BIGINT * 1000000000;

ALTER TABLE segments
    ALTER COLUMN duration TYPE BIGINT USING EXTRACT(EPOCH FROM duration)::BIGINT * 1000000000;

ALTER TABLE segments
    ALTER COLUMN pace TYPE BIGINT USING EXTRACT(EPOCH FROM pace)::BIGINT * 1000000000;

-- +goose Down
ALTER TABLE run_days
    ALTER COLUMN total_duration TYPE INTERVAL USING (total_duration * INTERVAL '1 nanosecond');

ALTER TABLE segments
    ALTER COLUMN duration TYPE INTERVAL USING (duration * INTERVAL '1 nanosecond');

ALTER TABLE segments
    ALTER COLUMN pace TYPE INTERVAL USING (pace * INTERVAL '1 nanosecond');
