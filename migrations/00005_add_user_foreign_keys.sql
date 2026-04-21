-- +goose Up
ALTER TABLE training_plans
    ADD CONSTRAINT fk_training_plans_user
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- +goose Down
ALTER TABLE training_plans DROP CONSTRAINT fk_training_plans_user;
