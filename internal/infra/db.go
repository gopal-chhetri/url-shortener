package infra

import (
	"context"
	"fmt"
	"log"
	"time"

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
		log.Printf("Failed to parse database config: %v", err)
		return nil
	}

	// Configure connection pool
	config.MaxConns = 25
	config.MinConns = 5
	config.MaxConnLifetime = 5 * time.Minute
	config.MaxConnIdleTime = 30 * time.Second

	// Configure logging
	config.ConnConfig.Tracer = &tracelog.TraceLog{
		Logger:   &pgxLogger{},
		LogLevel: tracelog.LogLevelDebug, // or LogLevelInfo
	}

	// Retry connection with exponential backoff
	var pool *pgxpool.Pool
	maxRetries := 5
	for i := 0; i < maxRetries; i++ {
		pool, err = pgxpool.NewWithConfig(context.Background(), config)
		if err == nil {
			// Verify connection
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err = pool.Ping(ctx)
			cancel()
			if err == nil {
				log.Printf("Database connection established successfully")
				return pool
			}
			pool.Close()
		}
		
		if i < maxRetries-1 {
			waitTime := time.Duration(1<<uint(i)) * time.Second // 1s, 2s, 4s, 8s, 16s
			log.Printf("Failed to connect to database (attempt %d/%d): %v. Retrying in %v...", i+1, maxRetries, err, waitTime)
			time.Sleep(waitTime)
		}
	}
	
	log.Printf("Failed to connect to database after %d attempts: %v", maxRetries, err)
	log.Panic(err)
	return nil
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
