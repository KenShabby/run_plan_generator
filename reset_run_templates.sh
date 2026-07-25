#!/bin/bash
set -e

psql -h db -U postgres -c "TRUNCATE template_segments, template_run_days, template_plans RESTART IDENTITY CASCADE;" run_plan_generator

for f in seeds/*.yaml; do
    echo "Seeding $f..."
    ./seed "$f" || { echo "FAILED: $f"; exit 1; }
done

echo "Done - all templates seeded."
