package infra

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DataBase struct {
	pool *pgxpool.Pool
}

func NewDataBase(pool *pgxpool.Pool) *DataBase {
	return &DataBase{
		pool: pool,
	}
}

func (db *DataBase) GetPool() *pgxpool.Pool {
	return db.pool
}

func (db *DataBase) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return db.pool.Begin(ctx)
}

func (db *DataBase) Close() {
	db.pool.Close()
}
