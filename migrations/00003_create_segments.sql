-- +goose Up
CREATE TABLE segments (
    id             SERIAL PRIMARY KEY,
    run_id         INTEGER NOT NULL REFERENCES run_days(id) ON DELETE CASCADE,
    order_index    INTEGER NOT NULL,
    description    TEXT,
    effort_type    VARCHAR(20) NOT NULL,
    duration       INTERVAL,
    distance       NUMERIC,
    pace           INTERVAL,
    repetitions    INTEGER NOT NULL DEFAULT 1,
    hr_zone_min    INTEGER,
    hr_zone_max    INTEGER,
    hr_abs_min     INTEGER,
    hr_abs_max     INTEGER,
    created_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE segments;
