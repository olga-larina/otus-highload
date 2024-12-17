package sqlstorage

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	ERR_CODE_FOREIGN_KEY_VIOLATION = "23503"
	ERR_CODE_UNIQUE_VIOLATION      = "23505"
)

type Db interface {
	Write(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	QueryRows(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}
