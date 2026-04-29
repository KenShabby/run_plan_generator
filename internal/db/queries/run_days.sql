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

-- name: GetRunDayWithPlanOwner :one
SELECT r.*, tp.user_id
FROM run_days r
JOIN training_plans tp ON tp.id = r.plan_id
WHERE r.id = $1;

-- name: DeleteRunDayIfOwner :exec
DELETE FROM run_days
WHERE run_days.id = $1
AND plan_id IN (
    SELECT id FROM training_plans WHERE user_id = $2
);
