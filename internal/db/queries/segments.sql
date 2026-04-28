-- name: ListSegmentsByRun :many
SELECT * FROM segments
WHERE run_id = $1
ORDER BY order_index ASC;

-- name: CreateSegment :one
INSERT INTO segments (run_id, order_index, description, effort_type, distance, duration, pace, repetitions, hr_zone_min, hr_zone_max, hr_abs_min, hr_abs_max, set_index, set_repetitions)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
RETURNING *;
