package bootstrap

import (
	"github.com/gopal-chhetri/url-shortener/internal/infra"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type Application struct {
	Env      *infra.Env
	Logger   *zap.Logger
	Database *pgxpool.Pool
	Redis    *redis.Client
}

func NewApplication() *Application {
	env := infra.NewEnv()
	logger := infra.NewLogger(env)
	dbConn := infra.NewDb(env)
	redisClient := infra.NewRedisClient(env, logger)
	return &Application{
		Env:      env,
		Logger:   logger,
		Database: dbConn,
		Redis:    redisClient,
	}
}
