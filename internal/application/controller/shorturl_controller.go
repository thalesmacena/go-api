package controller

import (
	"github.com/labstack/echo/v4"
	"go-api/internal/domain/model"
	"go-api/internal/domain/usecase/shorturl"
	"go-api/pkg/util/numberutils"
	"net/http"
)

type ShortUrlController struct {
	api     *echo.Group
	useCase shorturl.UseCase
}

func NewShortUrlController(api *echo.Group, useCase shorturl.UseCase) *ShortUrlController {
	return &ShortUrlController{api: api, useCase: useCase}
}

// InitShortUrlRoutes initializes short url routes
func (controller *ShortUrlController) InitShortUrlRoutes() {
	controller.api.GET("/short-url", controller.FindAll)
	controller.api.GET("/short-url/:hash", controller.FindByHash)
	controller.api.POST("/short-url", controller.Create)
	controller.api.PUT("/short-url/:hash", controller.UpdateByHash)
	controller.api.DELETE("/short-url/:hash", controller.DeleteByHash)
}

// FindAll handles GET requests to retrieve all short URLs
func (controller *ShortUrlController) FindAll(c echo.Context) error {
	var offset int = numberutils.ToIntWithDefault(c.Param("page"), 0)
	var limit int = numberutils.ToIntWithDefault(c.Param("size"), 10)
	shortUrls, err := controller.useCase.FindAll(offset, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, shortUrls)
}

// FindByHash handles GET requests to find a short URL by hash and redirect
func (controller *ShortUrlController) FindByHash(c echo.Context) error {
	hash := c.Param("hash")
	shortUrl, err := controller.useCase.FindByHash(hash)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Short URL not found"})
	}
	return c.Redirect(http.StatusMovedPermanently, shortUrl.Url)
}

// Create handles POST requests to create a new short URL
func (controller *ShortUrlController) Create(c echo.Context) error {
	var dto model.CreateShortUrlDTO
	if err := c.Bind(&dto); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	shortUrl, err := controller.useCase.Create(dto)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, shortUrl)
}

// UpdateByHash handles PUT requests to update a short URL by hash
func (controller *ShortUrlController) UpdateByHash(c echo.Context) error {
	hash := c.Param("hash")
	var dto model.UpdateShortUrlDTO
	if err := c.Bind(&dto); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	shortUrl, err := controller.useCase.UpdateByHash(hash, dto)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, shortUrl)
}

// DeleteByHash handles DELETE requests to delete a short URL by hash
func (controller *ShortUrlController) DeleteByHash(c echo.Context) error {
	hash := c.Param("hash")
	if err := controller.useCase.DeleteByHash(hash); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.NoContent(http.StatusNoContent)
}
