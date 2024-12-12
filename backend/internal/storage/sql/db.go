package sqlstorage

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Db interface {
	Write(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	QueryRows(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}
