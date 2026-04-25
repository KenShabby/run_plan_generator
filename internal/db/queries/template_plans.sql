-- name: ListTemplatePlans :many
SELECT * FROM template_plans
ORDER BY plan_type, total_weeks ASC;

-- name: ListTemplatePlansWithCounts :many
SELECT
    t.*,
    COUNT(r.id) AS run_count
FROM template_plans t
LEFT JOIN template_run_days r ON r.plan_id = t.id
GROUP BY t.id
ORDER BY t.plan_type, t.total_weeks ASC;

-- name: GetTemplatePlan :one
SELECT * FROM template_plans
WHERE id = $1;

-- name: CreateTemplatePlan :one
INSERT INTO template_plans (name, description, plan_type, distance_unit, total_weeks, peak_weekly_mileage)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;
