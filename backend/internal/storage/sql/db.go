package sqlstorage

import (
	"context"

	_ "github.com/jackc/pgx/v5/stdlib" // for postgres
	"github.com/jmoiron/sqlx"
)

type Db struct {
	sqlDb      *sqlx.DB
	driverName string
	dsn        string
}

func NewDb(driverName string, dsn string) *Db {
	return &Db{
		driverName: driverName,
		dsn:        dsn,
	}
}

func (s *Db) Connect(ctx context.Context) (err error) {
	s.sqlDb, err = sqlx.ConnectContext(ctx, s.driverName, s.dsn)
	return
}

func (s *Db) Close(_ context.Context) error {
	return s.sqlDb.Close()
}
