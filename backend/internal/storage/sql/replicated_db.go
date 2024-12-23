package sqlstorage

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/olga-larina/otus-highload/backend/internal/logger"
	"golang.org/x/exp/rand"
)

type DbConfig struct {
	Uri             string
	MaxConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}

type ReplicatedDb struct {
	master   *pgxpool.Pool
	replicas []*pgxpool.Pool
}

func NewReplicatedDb(ctx context.Context, masterConfig *DbConfig, replicasConfig []*DbConfig) (*ReplicatedDb, error) {
	master, err := initPool(ctx, masterConfig)
	if err != nil {
		logger.Error(ctx, err, "failed creating master db")
		return nil, err
	}
	replicas := make([]*pgxpool.Pool, len(replicasConfig))
	for i, replicaConfig := range replicasConfig {
		replicas[i], err = initPool(ctx, replicaConfig)
		if err != nil {
			logger.Error(ctx, err, fmt.Sprintf("failed creating %d replica db", i))
			return nil, err
		}
	}
	return &ReplicatedDb{
		master:   master,
		replicas: replicas,
	}, nil
}

func (r *ReplicatedDb) Connect(ctx context.Context) (err error) {
	return nil
}

func (r *ReplicatedDb) Close(_ context.Context) error {
	r.master.Close()
	for _, replica := range r.replicas {
		replica.Close()
	}
	return nil
}

func (r *ReplicatedDb) Write(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return r.connWrite().Exec(ctx, sql, args...)
}

func (r *ReplicatedDb) WriteReturn(ctx context.Context, sql string, args ...any) pgx.Row {
	return r.connWrite().QueryRow(ctx, sql, args...)
}

func (r *ReplicatedDb) QueryRows(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return r.connRead().Query(ctx, sql, args...)
}

func (r *ReplicatedDb) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return r.connRead().QueryRow(ctx, sql, args...)
}

func initPool(ctx context.Context, config *DbConfig) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(config.Uri)
	if err != nil {
		return nil, err
	}
	cfg.MaxConns = config.MaxConns
	cfg.MaxConnLifetime = config.MaxConnLifetime
	cfg.MaxConnIdleTime = config.MaxConnIdleTime
	return pgxpool.NewWithConfig(ctx, cfg)
}

func (r *ReplicatedDb) connRead() *pgxpool.Pool {
	if len(r.replicas) == 0 {
		return r.master
	}
	idx := rand.Intn(len(r.replicas))
	return r.replicas[idx]
}

func (r *ReplicatedDb) connWrite() *pgxpool.Pool {
	return r.master
}
