package pages

import "github.com/jackc/pgx/v5/pgtype"

func mustFloat(n pgtype.Numeric) float64 {
	f, _ := n.Float64Value()
	return f.Float64
}
