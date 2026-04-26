-- name: ListRunDaysByPlan :many
SELECT * FROM run_days
WHERE plan_id = $1
ORDER BY date ASC;

-- name: CreateRunDay :one
INSERT INTO run_days (plan_id, date, run_type, total_distance, total_duration, notes)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: DeleteRunDay :exec
DELETE FROM run_days WHERE id = $1;

-- name: GetRunDay :one
SELECT * FROM run_days WHERE id = $1;
