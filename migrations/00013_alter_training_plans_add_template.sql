-- +goose Up
ALTER TABLE training_plans
    ADD COLUMN template_id INTEGER REFERENCES template_plans(id) ON DELETE SET NULL;

-- +goose Down
ALTER TABLE training_plans
    DROP COLUMN template_id;
