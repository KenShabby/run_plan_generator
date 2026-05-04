-- name: ListRunDaysByPlan :many
SELECT * FROM run_days
WHERE plan_id = $1
ORDER BY date ASC;

-- name: CreateRunDay :one
INSERT INTO run_days (plan_id, date, run_type, total_distance, total_duration, notes, is_goal_race)
VALUES ($1, $2, $3, $4, $5, $6, $7)
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

-- name: GetNextRace :one
SELECT rd.*, tp.name as plan_name FROM run_days rd
JOIN training_plans tp ON tp.id = rd.plan_id
WHERE tp.user_id = $1
AND rd.is_goal_race = TRUE
AND rd.date >= CURRENT_DATE
ORDER BY rd.date ASC
LIMIT 1;

-- name: GetUpcomingRunsThisWeek :many
SELECT rd.*, tp.name as plan_name FROM run_days rd
JOIN training_plans tp ON tp.id = rd.plan_id
WHERE tp.user_id = $1
AND rd.date >= CURRENT_DATE
AND rd.date < CURRENT_DATE + INTERVAL '7 days'
ORDER BY rd.date ASC;
