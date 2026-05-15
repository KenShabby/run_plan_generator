-- +goose Up

CREATE TABLE activity_log (
    id          serial primary key,
    user_id     integer not null references users(id) on delete cascade,
    run_day_id  integer references run_days(id) on delete set null,
    date        date not null,
    run_type    text not null,
    distance    float8,
    duration    integer,          -- seconds
    pace        integer,          -- seconds per mile/km, stored explicitly
    rpe         smallint check (rpe between 1 and 10),
    notes       text,
    logged_at   timestamptz not null default now()
);

CREATE INDEX activity_log_user_id_idx ON activity_log(user_id);
CREATE INDEX activity_log_run_day_id_idx ON activity_log(run_day_id);
CREATE INDEX activity_log_date_idx ON activity_log(user_id, date DESC);

-- Mark run_days as completed when an activity is logged against them
-- We'll handle this in application code rather than a trigger for simplicity

-- +goose Down

DROP TABLE activity_log;
