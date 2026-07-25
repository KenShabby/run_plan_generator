#!/bin/sh
goose -dir migrations postgres "postgres://postgres:postgres@db:5432/run_plan_generator?sslmode=disable" up
