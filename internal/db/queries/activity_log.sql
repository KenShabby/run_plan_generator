-- name: CreateActivityLog :one
INSERT INTO activity_log (
    user_id, run_day_id, date, run_type,
    distance, duration, pace, rpe, notes
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
)
RETURNING *;

-- name: GetActivityLogByID :one
SELECT * FROM activity_log
WHERE id = $1;

-- name: ListActivityLogByUser :many
SELECT * FROM activity_log
WHERE user_id = $1
ORDER BY date DESC;

-- name: ListActivityLogByUserPaged :many
SELECT * FROM activity_log
WHERE user_id = $1
ORDER BY date DESC
LIMIT $2 OFFSET $3;

-- name: GetActivityLogByRunDay :one
SELECT * FROM activity_log
WHERE run_day_id = $1
LIMIT 1;

-- name: UpdateActivityLog :one
UPDATE activity_log
SET distance = $2,
    duration = $3,
    pace     = $4,
    rpe      = $5,
    notes    = $6
WHERE id = $1
RETURNING *;

-- name: DeleteActivityLog :exec
DELETE FROM activity_log WHERE id = $1;

-- name: GetRecentActivityByUser :many
SELECT * FROM activity_log
WHERE user_id = $1
AND date >= CURRENT_DATE - INTERVAL '30 days'
ORDER BY date DESC;

-- name: GetActivitySummaryByUser :one
SELECT
    COUNT(*)                          AS total_runs,
    COALESCE(SUM(distance), 0)        AS total_distance,
    COALESCE(SUM(duration), 0)        AS total_duration,
    COALESCE(AVG(rpe), 0)             AS avg_rpe,
    COALESCE(AVG(distance), 0)        AS avg_distance
FROM activity_log
WHERE user_id = $1
AND date >= CURRENT_DATE - INTERVAL '30 days';
