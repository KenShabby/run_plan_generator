-- +goose Up
CREATE TABLE template_segments (
    id           SERIAL PRIMARY KEY,
    run_id       INTEGER NOT NULL REFERENCES template_run_days(id) ON DELETE CASCADE,
    order_index  INTEGER NOT NULL,
    description  TEXT,
    effort_type  VARCHAR(20) NOT NULL DEFAULT 'distance',
    distance     DOUBLE PRECISION,
    duration     BIGINT,
    pace         BIGINT,
    repetitions  INTEGER NOT NULL DEFAULT 1,
    hr_zone_min  INTEGER,
    hr_zone_max  INTEGER,
    created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE template_segments;
