package main

import (
	"github.com/labstack/echo/v4"
	_ "go-api/configs"
	"go-api/internal/application/controller"
	"go-api/internal/application/schedule"
	"go-api/internal/domain/gateway/db"
	"go-api/internal/domain/usecase/health"
	"go-api/internal/domain/usecase/shorturl"
	"go-api/internal/infra/database/sqlc"
	"go-api/pkg/log"
	"go-api/pkg/msg"
	"go-api/pkg/resource"
)

func main() {
	log.Info(msg.GetMessage("app.start"))

	// Init infra
	e := echo.New()
	api := e.Group(resource.GetString("app.server.context-path"))

	// Init ShortUrlGateway
	// dbGatewayGorm := db.NewGormHealthDBGateway(gorm.Db)
	dbGatewaySQLC := db.NewSQLCHealthDBGateway(sqlc.Db)
	shortUrlRepository := db.NewSQLCShortUrlGateway(sqlc.Db)

	// Init UseCase
	healthUseCase := health.NewHealthUseCase(dbGatewaySQLC)
	shortUrlUseCase := shorturl.NewShortUrlUseCase(shortUrlRepository)

	// Init Controller
	healthController := controller.NewHealthController(api, healthUseCase)
	shortUrlController := controller.NewShortUrlController(api, shortUrlUseCase)

	// Init Routes
	healthController.InitHealthRoutes()
	shortUrlController.InitShortUrlRoutes()

	// Init Schedule
	shortUrlScheduler := schedule.NewShortUrlScheduler(shortUrlUseCase)
	shortUrlScheduler.InitShortUrlScheduleTasks()

	// Start Routes
	e.Logger.Fatal(e.Start(":" + resource.GetString("app.server.port")))
	log.Info(msg.GetMessage("app.started"))
}
