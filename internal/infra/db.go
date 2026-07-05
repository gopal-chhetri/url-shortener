package infra

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"
)

func NewDb(env *Env) *pgxpool.Pool {
	dbUser := env.DBUser
	dbPass := env.DBPass
	dbName := env.DBName
	dbHost := env.DBHost
	dbPort := env.DBPort
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPass, dbHost, dbPort, dbName)

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil
	}

	// Configure logging
	config.ConnConfig.Tracer = &tracelog.TraceLog{
		Logger:   &pgxLogger{},
		LogLevel: tracelog.LogLevelDebug, // or LogLevelInfo
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		log.Panic(err)
	}
	return pool
}

type pgxLogger struct{}

func (l *pgxLogger) Log(ctx context.Context, level tracelog.LogLevel, msg string, data map[string]interface{}) {
	if msg == "Query" {
		log.Printf("SQL: %s\nArgs: %v\nDuration: %v",
			data["sql"],
			data["args"],
			data["time"],
		)
	}
}

// BeginTx begins a transaction
func BeginTx(pool *pgxpool.Pool, ctx context.Context) (pgx.Tx, error) {
	return pool.Begin(ctx)
}

// Close closes the pool
func Close(pool *pgxpool.Pool) {
	pool.Close()
}
