package bootstrap

import (
	"github.com/gopal-chhetri/url-shortener/internal/infra"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type Application struct {
	Env        *infra.Env
	Logger     *zap.Logger
	Database   *pgxpool.Pool
	Redis      *redis.Client
	GeoService *infra.GeoService
}

func NewApplication() *Application {
	env := infra.NewEnv()
	logger := infra.NewLogger(env)
	dbConn := infra.NewDb(env)
	redisClient := infra.NewRedisClient(env, logger)

	var geoService *infra.GeoService
	if geoDB, err := infra.NewGeoService("./GeoLite2-City.mmdb"); err == nil {
		geoService = geoDB
		logger.Info("GeoIP database loaded successfully")
	} else {
		logger.Warn("GeoIP database not found, geo-location tracking disabled", zap.Error(err))
	}

	return &Application{
		Env:        env,
		Logger:     logger,
		Database:   dbConn,
		Redis:      redisClient,
		GeoService: geoService,
	}
}
