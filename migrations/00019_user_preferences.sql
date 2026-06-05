-- +goose Up
CREATE TABLE user_preferences (
    id          serial primary key,
    user_id     integer not null unique references users(id) on delete cascade,
    distance_unit varchar(10) not null default 'miles',
    created_at  timestamptz not null default now(),
    updated_at  timestamptz not null default now()
);

-- +goose Down
DROP TABLE user_preferences;
