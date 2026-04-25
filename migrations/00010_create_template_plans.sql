-- +goose Up
CREATE TABLE template_plans (
    id                  SERIAL PRIMARY KEY,
    name                VARCHAR(100) NOT NULL,
    description         TEXT,
    plan_type           VARCHAR(50) NOT NULL,
    distance_unit       VARCHAR(10) NOT NULL DEFAULT 'miles',
    total_weeks         INTEGER NOT NULL,
    peak_weekly_mileage NUMERIC,
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE template_plans;
