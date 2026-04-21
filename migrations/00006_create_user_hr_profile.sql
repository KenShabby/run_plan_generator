-- +goose Up
CREATE TABLE user_hr_profile (
    id                   SERIAL PRIMARY KEY,
    user_id              INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    max_hr               INTEGER,
    resting_hr           INTEGER,
    lactate_threshold_hr INTEGER,
    calculation_method   VARCHAR(20) NOT NULL,
    created_at           TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE user_hr_profile;
