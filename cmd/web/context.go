package main

import (
	"context"

	"github.com/KenShabby/run_plan_generator/internal/db"
)

type contextKey string

const userContextKey contextKey = "user"

func withUser(ctx context.Context, user db.User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

func userFromContext(ctx context.Context) (db.User, bool) {
	user, ok := ctx.Value(userContextKey).(db.User)
	return user, ok
}
