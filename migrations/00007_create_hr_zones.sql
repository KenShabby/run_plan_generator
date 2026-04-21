-- +goose Up
CREATE TABLE hr_zones (
    id          SERIAL PRIMARY KEY,
    profile_id  INTEGER NOT NULL REFERENCES user_hr_profile(id) ON DELETE CASCADE,
    zone_number INTEGER NOT NULL,
    name        VARCHAR(50),
    hr_min      INTEGER NOT NULL,
    hr_max      INTEGER NOT NULL,
    description TEXT,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE hr_zones;
