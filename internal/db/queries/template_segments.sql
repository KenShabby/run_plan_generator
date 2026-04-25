-- name: ListTemplateSegmentsByRun :many
SELECT * FROM template_segments
WHERE run_id = $1
ORDER BY order_index ASC;

-- name: CreateTemplateSegment :one
INSERT INTO template_segments (run_id, order_index, description, effort_type, distance, duration, pace, repetitions, hr_zone_min, hr_zone_max)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;
