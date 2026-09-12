package csrfctx

import "context"

type contextKey string

const tokenKey contextKey = "csrf_token"

func WithToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, tokenKey, token)
}

func Token(ctx context.Context) string {
	token, _ := ctx.Value(tokenKey).(string)
	return token
}
