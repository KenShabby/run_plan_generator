-- +goose Up
CREATE TABLE run_days (
    id             SERIAL PRIMARY KEY,
    plan_id        INTEGER NOT NULL REFERENCES training_plans(id) ON DELETE CASCADE,
    date           DATE NOT NULL,
    run_type       VARCHAR(50) NOT NULL,
    total_distance NUMERIC,
    total_duration INTERVAL,
    completed      BOOLEAN NOT NULL DEFAULT FALSE,
    notes          TEXT,
    created_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE run_days;
