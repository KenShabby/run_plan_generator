-- +goose Up
CREATE TABLE training_plans (
	id          SERIAL PRIMARY KEY,
	user_id     INTEGER NOT NULL,
	name        VARCHAR(100) NOT NULL,
	description TEXT,
	plan_type   VARCHAR(50) NOT NULL,
	distance_unit VARCHAR(10) NOT NULL DEFAULT 'miles',
	start_date  DATE NOT NULL,
	end_date    DATE NOT NULL,
	created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE training_plans;
