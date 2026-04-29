-- name: CreateTrainingPlan :one
INSERT INTO training_plans (user_id, name, description, plan_type, distance_unit, start_date, end_date, template_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetTrainingPlan :one
SELECT * FROM training_plans WHERE id = $1;

-- name: ListTrainingPlansByUser :many
SELECT * FROM training_plans WHERE user_id = $1;

-- name: DeleteTrainingPlan :exec
DELETE FROM training_plans WHERE id = $1;

-- name: UpdateTrainingPlan :one
UPDATE training_plans
SET name = $2, description = $3, plan_type = $4, distance_unit = $5, start_date = $6, end_date = $7, template_id = $8
WHERE id = $1
RETURNING *;

-- name: DeleteTrainingPlanIfOwner :exec
DELETE FROM training_plans
WHERE id = $1
AND user_id = $2;
