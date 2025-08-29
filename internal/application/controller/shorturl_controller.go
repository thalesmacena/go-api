package controller

import (
	"go-api/internal/domain/entity"
	"go-api/internal/domain/model"
	"go-api/internal/domain/usecase/shorturl"
	"go-api/pkg/util/numberutils"
	"net/http"

	"github.com/labstack/echo/v4"
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

// FindAll godoc
// @Summary Get all short URLs
// @Description Retrieve all short URLs with pagination and optional URL filtering
// @Tags short-url
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(0)
// @Param size query int false "Page size" default(10)
// @Param urlPart query string false "URL part to filter by"
// @Success 200 {array} entity.ShortUrl "Paginated list of short URLs"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /short-url [get]
func (controller *ShortUrlController) FindAll(c echo.Context) error {
	var page int = numberutils.ToIntWithDefault(c.QueryParam("page"), 0)
	var size int = numberutils.ToIntWithDefault(c.QueryParam("size"), 10)
	var urlPart string = c.QueryParam("urlPart")

	var shortUrlsPage *model.Page[entity.ShortUrl]
	var err error

	if urlPart != "" {
		shortUrlsPage, err = controller.useCase.FindByURLPart(urlPart, page, size)
	} else {
		shortUrlsPage, err = controller.useCase.FindAll(page, size)
	}

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, shortUrlsPage)
}

// FindByHash godoc
// @Summary Get short URL by hash and redirect
// @Description Find a short URL by its hash and redirect to the original URL
// @Tags short-url
// @Accept json
// @Produce json
// @Param hash path string true "Short URL hash"
// @Success 301 "Redirect to original URL"
// @Failure 404 {object} map[string]string "Short URL not found"
// @Router /short-url/{hash} [get]
func (controller *ShortUrlController) FindByHash(c echo.Context) error {
	hash := c.Param("hash")
	shortUrl, err := controller.useCase.FindByHash(hash)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Short URL not found"})
	}
	return c.Redirect(http.StatusMovedPermanently, shortUrl.Url)
}

// Create godoc
// @Summary Create a new short URL
// @Description Create a new short URL from the provided URL and expiration
// @Tags short-url
// @Accept json
// @Produce json
// @Param shortUrl body model.CreateShortUrlDTO true "Short URL creation data"
// @Success 201 {object} entity.ShortUrl "Created short URL"
// @Failure 400 {object} map[string]string "Invalid request body"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /short-url [post]
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

// UpdateByHash godoc
// @Summary Update short URL by hash
// @Description Update a short URL's details by its hash
// @Tags short-url
// @Accept json
// @Produce json
// @Param hash path string true "Short URL hash"
// @Param shortUrl body model.UpdateShortUrlDTO true "Short URL update data"
// @Success 200 {object} entity.ShortUrl "Updated short URL"
// @Failure 400 {object} map[string]string "Invalid request body"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /short-url/{hash} [put]
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

// DeleteByHash godoc
// @Summary Delete short URL by hash
// @Description Delete a short URL by its hash
// @Tags short-url
// @Accept json
// @Produce json
// @Param hash path string true "Short URL hash"
// @Success 204 "Short URL deleted successfully"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /short-url/{hash} [delete]
func (controller *ShortUrlController) DeleteByHash(c echo.Context) error {
	hash := c.Param("hash")
	if err := controller.useCase.DeleteByHash(hash); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.NoContent(http.StatusNoContent)
}
