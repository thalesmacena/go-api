// Package main provides the entry point for the go-api application.
//
// @title Go API
// @version 1.0
// @description A REST API for URL shortening and weather monitoring services
// @termsOfService http://swagger.io/terms/
//
// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io
//
// @license.name MIT
// @license.url https://opensource.org/licenses/MIT
//
// @host localhost:8080
// @BasePath /go-api
//
// @schemes http https
package main

import (
	"context"
	"strings"

	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
	echoSwagger "github.com/swaggo/echo-swagger"

	_ "go-api/docs" // Import generated docs
	"go-api/internal/application/controller"
	appmw "go-api/internal/application/middleware"
	"go-api/internal/application/processor"
	"go-api/internal/application/schedule"
	"go-api/internal/domain/gateway/api"
	"go-api/internal/domain/gateway/db"
	"go-api/internal/domain/gateway/queue"
	"go-api/internal/domain/usecase/health"
	"go-api/internal/domain/usecase/shorturl"
	"go-api/internal/domain/usecase/weather"
	"go-api/internal/infra/aws"
	"go-api/internal/infra/database/sqlc"
	"go-api/pkg/http"
	"go-api/pkg/log"
	"go-api/pkg/msg"
	"go-api/pkg/redis"
	"go-api/pkg/resource"
	"go-api/pkg/sqs"
)

func main() {
	log.Info(msg.GetMessage("app.start"))

	// Init infra
	e := echo.New()
	setupMiddleware(e)
	apiGroup := e.Group(resource.GetString("app.server.context-path"))

	// Init Database Gateways
	// dbGatewayGorm := db.NewGormHealthDBGateway(gorm.Db)
	dbGatewaySQLC := db.NewSQLCHealthDBGateway(sqlc.Db)
	shortUrlRepository := db.NewSQLCShortUrlGateway(sqlc.Db)
	cityGateway := db.NewSQLCCityGateway(sqlc.Db)

	// Init AWS Resources
	sqsClient := aws.NewSqsClient()
	queueSender := aws.NewSQSSenderAdapter(sqsClient)

	// Init Queue Health Gateway
	queueHealthGateway := queue.NewQueueHealthGateway()

	// Init Redis Client
	redisConfig := redis.NewRedisConfig().
		WithHost(resource.GetString("app.cache.redis.host")).
		WithPort(resource.GetInt("app.cache.redis.port")).
		WithPassword(resource.GetString("app.cache.redis.password")).
		WithDatabase(resource.GetInt("app.cache.redis.db")).
		WithMinIdleConns(resource.GetInt("app.cache.redis.pool.min-idle-conns")).
		WithMaxIdleConns(resource.GetInt("app.cache.redis.pool.max-idle-conns")).
		WithMaxActive(resource.GetInt("app.cache.redis.pool.max-active"))

	redisClient := redis.NewClient(redisConfig)
	defer func(client *redis.Client) {
		if err := client.Close(); err != nil {
			log.Errorf("Error closing Redis client: %v", err)
		}
	}(redisClient)

	// Init External API Gateways
	httpClientOptions := http.ClientOptions{
		FollowRedirect:      resource.GetBool("weather.follow-redirect"),
		Dismiss404:          resource.GetBool("weather.dismiss-404"),
		MaxIdleConns:        resource.GetInt("weather.max-idle-conns"),
		MaxIdleConnsPerHost: resource.GetInt("weather.max-idle-conns-per-host"),
		IdleConnTimeout:     resource.GetDuration("weather.idle-conn-timeout"),
		ConnectionTimeout:   resource.GetDuration("weather.connection-timeout"),
		ReadTimeout:         resource.GetDuration("weather.read-timeout"),
		DefaultContentType:  resource.GetString("weather.default-content-type"),
	}
	weatherGateway := api.NewWeatherGateway(resource.GetString("weather.base-url"), httpClientOptions)

	// Init UseCases
	healthUseCase := health.NewHealthUseCase(dbGatewaySQLC, queueHealthGateway)
	shortUrlUseCase := shorturl.NewShortUrlUseCase(shortUrlRepository)
	weatherUseCase := weather.NewWeatherUseCase(resource.GetString("weather.queue-name"),
		resource.GetInt("weather.batch-size"),
		queueSender,
		weatherGateway,
		cityGateway)

	// Init Controllers
	healthController := controller.NewHealthController(apiGroup, healthUseCase)
	shortUrlController := controller.NewShortUrlController(apiGroup, shortUrlUseCase)
	weatherController := controller.NewWeatherController(apiGroup, weatherUseCase)

	// Init Routes
	healthController.InitHealthRoutes()
	shortUrlController.InitShortUrlRoutes()
	weatherController.InitWeatherRoutes()

	// Swagger route
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	// Init Schedule
	shortUrlScheduler := schedule.NewShortUrlScheduler(shortUrlUseCase)
	shortUrlScheduler.InitShortUrlScheduleTasks()

	// Init Weather Schedule with distributed locking
	weatherScheduler := schedule.NewWeatherScheduler(
		weatherUseCase,
		redisClient,
		resource.GetString("weather.schedule.cron"),
		resource.GetInt("weather.schedule.lock-ttl"),
		resource.GetInt("weather.schedule.refresh-interval"),
	)

	// Initialize scheduler in background (goroutine handles lock acquisition)
	weatherScheduler.InitWeatherScheduleTasks(context.Background())

	// Init Weather Processor and Worker
	weatherProcessor := processor.NewWeatherProcessor(weatherUseCase)

	weatherWorker, err := sqs.NewWorker(sqsClient,
		resource.GetString("weather.queue-name"),
		weatherProcessor,
		&sqs.WorkerConfig{
			MaxNumberOfMessages: resource.GetInt64("weather.worker.max-number-of-messages"),
			WaitTimeSeconds:     resource.GetInt64("weather.worker.wait-time-seconds"),
			PoolSize:            resource.GetInt64("weather.worker.pool-size"),
			LogLevel:            sqs.ParseLogLevel(resource.GetString("weather.worker.log-level")),
		},
	)

	if err != nil {
		log.Fatalf("Failed to create weather worker: %v", err)
	}

	// Register worker in health gateway
	queueHealthGateway.RegisterWorker("weather-worker", weatherWorker)

	// Start Weather Worker in background
	go func() {
		log.Info("Starting weather queue worker...")
		weatherWorker.Start(context.Background())
	}()

	// Start Routes
	e.Logger.Fatal(e.Start(":" + resource.GetString("app.server.port")))
	log.Info(msg.GetMessage("app.started"))
}

func setupMiddleware(e *echo.Echo) {
	// Recovery middleware
	e.Use(echomw.Recover())

	// Logger middleware (moved to dedicated package)
	appmw.SetupRequestLogger(e)

	// CORS middleware
	e.Use(echomw.CORSWithConfig(echomw.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{echo.GET, echo.POST, echo.PUT, echo.DELETE, echo.OPTIONS},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
	}))

	// Security headers middleware
	e.Use(echomw.SecureWithConfig(echomw.SecureConfig{
		XSSProtection:         "1; mode=block",
		ContentTypeNosniff:    "nosniff",
		XFrameOptions:         "DENY",
		HSTSMaxAge:            3600,
		ContentSecurityPolicy: "default-src 'self'",
		Skipper: func(c echo.Context) bool {
			path := c.Request().URL.Path
			if strings.Contains(path, "/swagger/") {
				return true
			}
			return false
		},
	}))

	// Body limit middleware
	e.Use(echomw.BodyLimit(resource.GetString("app.server.body-limit")))

	// Timeout middleware
	e.Use(echomw.TimeoutWithConfig(echomw.TimeoutConfig{
		Timeout: resource.GetDuration("app.server.timeout"),
	}))
}
