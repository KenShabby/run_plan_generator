-- name: CreateTrainingPlan :one
INSERT INTO training_plans (user_id, name, description, plan_type, distance_unit, start_date, end_date)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetTrainingPlan :one
SELECT * FROM training_plans WHERE id = $1;

-- name: ListTrainingPlansByUser :many
SELECT * FROM training_plans WHERE user_id = $1;

-- name: DeleteTrainingPlan :exec
DELETE FROM training_plans WHERE id = $1;
