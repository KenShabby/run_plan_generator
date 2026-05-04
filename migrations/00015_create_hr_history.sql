-- +goose Up
CREATE TABLE user_hr_history (
    id          SERIAL PRIMARY KEY,
    user_id     INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    max_hr      INTEGER,
    resting_hr  INTEGER,
    lthr        INTEGER,
    method      VARCHAR(20) NOT NULL,
    recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE user_hr_history;
