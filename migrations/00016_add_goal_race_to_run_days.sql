-- +goose Up
ALTER TABLE run_days
    ADD COLUMN is_goal_race BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE run_days
    DROP COLUMN is_goal_race;
