package bootstrap

import (
	"github.com/gopal-chhetri/url-shortener/internal/infra"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type Application struct {
	Env      *infra.Env
	Logger   *zap.Logger
	Database *infra.DataBase
	Redis    *redis.Client
}

func NewApplication() *Application {
	env := infra.NewEnv()
	logger := infra.NewLogger(env)
	dbConn := infra.NewDb(env)
	database := infra.NewDataBase(dbConn)
	redisClient := infra.NewRedisClient(env, logger)
	return &Application{
		Env:      env,
		Logger:   logger,
		Database: database,
		Redis:    redisClient,
	}
}
