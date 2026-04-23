package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type HealthHandler struct {
	dbCheck    func() error
	redisCheck func() error
}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

func (h *HealthHandler) Health(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	_ = ctx

	checks := map[string]string{
		"service": "healthy",
	}

	if h.dbCheck != nil {
		if err := h.dbCheck(); err != nil {
			checks["database"] = "unhealthy"
		} else {
			checks["database"] = "healthy"
		}
	}

	if h.redisCheck != nil {
		if err := h.redisCheck(); err != nil {
			checks["redis"] = "unhealthy"
		} else {
			checks["redis"] = "healthy"
		}
	}

	status := "healthy"
	for _, v := range checks {
		if v == "unhealthy" {
			status = "degraded"
			break
		}
	}

	statusCode := http.StatusOK
	if status == "degraded" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, gin.H{
		"status": status,
		"checks": checks,
	})
}
