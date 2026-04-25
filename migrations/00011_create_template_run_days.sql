-- +goose Up
CREATE TABLE template_run_days (
    id          SERIAL PRIMARY KEY,
    plan_id     INTEGER NOT NULL REFERENCES template_plans(id) ON DELETE CASCADE,
    day_offset  INTEGER NOT NULL,
    run_type    VARCHAR(50) NOT NULL,
    distance    DOUBLE PRECISION,
    notes       TEXT,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE template_run_days;
