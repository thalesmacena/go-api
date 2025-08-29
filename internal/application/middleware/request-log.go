package middleware

import (
	"strings"

	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"

	"go-api/pkg/log"
	"go-api/pkg/msg"
)

// SetupRequestLogger registers the request logging middleware with custom log output.
func SetupRequestLogger(e *echo.Echo) {
	e.Use(echomw.RequestLoggerWithConfig(echomw.RequestLoggerConfig{
		LogURI:     true,
		LogStatus:  true,
		LogMethod:  true,
		LogLatency: true,
		LogError:   true,
		Skipper: func(c echo.Context) bool {
			path := c.Request().URL.Path
			// Skip logging for health check and swagger requests
			if strings.Contains(path, "/health") || strings.Contains(path, "/swagger/") {
				return true
			}
			return false
		},
		LogValuesFunc: func(c echo.Context, v echomw.RequestLoggerValues) error {
			if v.Error == nil {
				log.Info(msg.GetMessage("app.req-end", v.Method, v.URI, v.Status, v.Latency, v.RequestID),
					zap.String("method", v.Method),
					zap.String("uri", v.URI),
					zap.Int("status", v.Status),
					zap.Duration("latency", v.Latency),
					zap.String("request_id", v.RequestID),
				)
			} else {
				log.Error(msg.GetMessage("app.req-fail", v.Method, v.URI, v.Status, v.Latency, v.RequestID, v.Error),
					zap.String("method", v.Method),
					zap.String("uri", v.URI),
					zap.Int("status", v.Status),
					zap.Duration("latency", v.Latency),
					zap.String("request_id", v.RequestID),
					zap.Error(v.Error),
				)
			}
			return nil
		},
	}))
}
