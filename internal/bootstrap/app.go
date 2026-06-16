package bootstrap

import (
	"github.com/gopal-chhetri/url-shortener/internal/infra"
	"go.uber.org/zap"
)

type Application struct {
	Env      *infra.Env
	Logger   *zap.Logger
	Database *infra.DataBase
}

func NewApplication() *Application {
	env := infra.NewEnv()
	logger := infra.NewLogger(env)
	dbConn := infra.NewDb(env)
	database := infra.NewDataBase(dbConn)
	return &Application{
		Env:      env,
		Logger:   logger,
		Database: database,
	}
}
