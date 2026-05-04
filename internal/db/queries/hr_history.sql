-- name: InsertHRHistory :exec
INSERT INTO user_hr_history (user_id, max_hr, resting_hr, lthr, method)
VALUES ($1, $2, $3, $4, $5);

-- name: GetHRHistoryByUser :many
SELECT * FROM user_hr_history
WHERE user_id = $1
ORDER BY recorded_at DESC
LIMIT 10;
