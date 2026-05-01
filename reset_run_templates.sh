#!/bin/bash
psql -c "TRUNCATE template_segments, template_run_days, template_plans RESTART IDENTITY CASCADE;" run_plan_generator
for f in seeds/*.yaml; do
    echo "Seeding $f..."
    go run ./cmd/seed/main.go "$f"
done
