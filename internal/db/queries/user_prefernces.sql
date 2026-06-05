-- name: GetUserPreferences :one
SELECT * FROM user_preferences
WHERE user_id = $1;

-- name: UpsertUserPreferences :one
INSERT INTO user_preferences (user_id, distance_unit)
VALUES ($1, $2)
ON CONFLICT (user_id) DO UPDATE
SET distance_unit = $2,
    updated_at = now()
RETURNING *;
