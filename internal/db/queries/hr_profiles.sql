-- name: GetHRProfileByUser :one
SELECT * FROM user_hr_profile
WHERE user_id = $1;

-- name: CreateHRProfile :one
INSERT INTO user_hr_profile (user_id, max_hr, resting_hr, lactate_threshold_hr, calculation_method)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateHRProfile :one
UPDATE user_hr_profile
SET max_hr = $2,
    resting_hr = $3,
    lactate_threshold_hr = $4,
    calculation_method = $5
WHERE user_id = $1
RETURNING *;
