-- +goose Up
ALTER TABLE segments
    ADD COLUMN set_index      INTEGER,
    ADD COLUMN set_repetitions INTEGER;

ALTER TABLE template_segments
    ADD COLUMN set_index      INTEGER,
    ADD COLUMN set_repetitions INTEGER;

-- +goose Down
ALTER TABLE segments
    DROP COLUMN set_index,
    DROP COLUMN set_repetitions;

ALTER TABLE template_segments
    DROP COLUMN set_index,
    DROP COLUMN set_repetitions;
