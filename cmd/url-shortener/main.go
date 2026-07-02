package main

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	docs "github.com/gopal-chhetri/url-shortener/docs" // generated swagger docs
	"github.com/gopal-chhetri/url-shortener/internal/bootstrap"
	"github.com/gopal-chhetri/url-shortener/internal/routes"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title URL Shortener API
// @version 1.0
// @description This is a sample URL Shortener server with authentication and authorization.
// @termsOfService http://swagger.io/terms/
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @host localhost:8000
// @BasePath /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.
func main() {
	app := bootstrap.NewApplication()

	// Update Swagger host dynamically
	docs.SwaggerInfo.Host = "localhost:" + app.Env.Port

	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"*"},
	}))

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	routes.SetupRoute(app, r)

	if err := r.Run(":" + app.Env.Port); err != nil {
		panic(err)
	}
}
