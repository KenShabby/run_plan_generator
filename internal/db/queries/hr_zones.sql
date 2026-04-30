-- name: GetHRZonesByProfile :many
SELECT * FROM hr_zones
WHERE profile_id = $1
ORDER BY zone_number ASC;

-- name: CreateHRZone :one
INSERT INTO hr_zones (profile_id, zone_number, name, hr_min, hr_max, description)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: DeleteHRZonesByProfile :exec
DELETE FROM hr_zones
WHERE profile_id = $1;
