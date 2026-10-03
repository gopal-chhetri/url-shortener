package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	docs "github.com/gopal-chhetri/url-shortener/docs" // generated swagger docs
	"github.com/gopal-chhetri/url-shortener/internal/bootstrap"
	"github.com/gopal-chhetri/url-shortener/internal/routes"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
)

// @title URL Shortener API
// @version 1.0
// @description This is a URL Shortener server with authentication, authorization, and analytics.
// @termsOfService http://swagger.io/terms/
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @BasePath /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.
func main() {
	app := bootstrap.NewApplication()

	defer app.Close()

	// An empty host makes Swagger UI target whichever host served it, so
	// "Try it out" works locally and in production alike.
	docs.SwaggerInfo.Host = ""

	if !app.Env.IsLocal() {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()

	// Never trust X-Forwarded-For from arbitrary peers; it would let clients
	// spoof their IP past the rate limiter and demo quota. In production the
	// real client IP comes from Cloudflare's CF-Connecting-IP header.
	if err := r.SetTrustedProxies(nil); err != nil {
		panic(err)
	}
	if !app.Env.IsLocal() {
		r.TrustedPlatform = gin.PlatformCloudflare
	}

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	routes.SetupRoute(app, r)

	srv := &http.Server{
		Addr:              ":" + app.Env.Port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			app.Logger.Fatal("Server failed", zap.Error(err))
		}
	}()
	app.Logger.Info("Server started", zap.String("addr", srv.Addr))

	<-ctx.Done()
	app.Logger.Info("Shutting down, draining in-flight requests")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		app.Logger.Error("Graceful shutdown failed", zap.Error(err))
	}
}
