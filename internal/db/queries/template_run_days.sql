-- name: ListTemplateRunDaysByPlan :many
SELECT * FROM template_run_days
WHERE plan_id = $1
ORDER BY day_offset ASC;

-- name: CreateTemplateRunDay :one
INSERT INTO template_run_days (plan_id, day_offset, run_type, distance, notes)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;
