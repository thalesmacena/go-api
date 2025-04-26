package controller

import (
	"github.com/labstack/echo/v4"
	"go-api/internal/domain/usecase/health"
	"net/http"
)

type HealthController struct {
	api     *echo.Group
	useCase health.UseCase
}

func NewHealthController(api *echo.Group, useCase health.UseCase) *HealthController {
	return &HealthController{api: api, useCase: useCase}
}

// InitHealthRoutes initializes health check routes
func (controller *HealthController) InitHealthRoutes() {
	controller.api.GET("/health", controller.CheckHealth())
}

func (controller *HealthController) CheckHealth() echo.HandlerFunc {
	return func(c echo.Context) error {
		healthResponse := controller.useCase.CheckHealth()

		return c.JSON(http.StatusOK, healthResponse)
	}
}
