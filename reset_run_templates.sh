#!/bin/bash
psql run_plan_generator -c "DELETE FROM template_plans;"
for f in seeds/*.yaml; do
    echo "Seeding $f..."
    go run ./cmd/seed/main.go "$f"
done
