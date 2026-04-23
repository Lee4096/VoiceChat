package http

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"voicechat/internal/repository/postgres"
	redisclient "voicechat/internal/repository/redis"
)

type HealthChecker struct {
	db    *postgres.DB
	redis *redisclient.Client
}

func NewHealthChecker(db *postgres.DB, redis *redisclient.Client) *HealthChecker {
	return &HealthChecker{
		db:    db,
		redis: redis,
	}
}

type HealthStatus struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks"`
}

func (h *HealthChecker) HealthReadiness() HealthStatus {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	checks := make(map[string]string)
	allHealthy := true

	if h.db != nil {
		if err := h.db.Pool().Ping(ctx); err != nil {
			checks["database"] = "unhealthy: " + err.Error()
			allHealthy = false
		} else {
			checks["database"] = "healthy"
		}
	}

	if h.redis != nil {
		if err := h.redis.Redis().Ping(ctx).Err(); err != nil {
			checks["redis"] = "unhealthy: " + err.Error()
			allHealthy = false
		} else {
			checks["redis"] = "healthy"
		}
	}

	status := "healthy"
	if !allHealthy {
		status = "degraded"
	}

	return HealthStatus{
		Status: status,
		Checks: checks,
	}
}

func (h *HealthChecker) HealthLiveness() HealthStatus {
	return HealthStatus{
		Status: "alive",
		Checks: map[string]string{
			"service": "alive",
		},
	}
}

func (h *HealthChecker) RegisterRoutes(e *gin.Engine) {
	e.GET("/health/live", func(c *gin.Context) {
		status := h.HealthLiveness()
		c.JSON(http.StatusOK, status)
	})

	e.GET("/health/ready", func(c *gin.Context) {
		status := h.HealthReadiness()
		statusCode := http.StatusOK
		if status.Status == "degraded" {
			statusCode = http.StatusServiceUnavailable
		}
		c.JSON(statusCode, status)
	})

	e.GET("/health", func(c *gin.Context) {
		status := h.HealthReadiness()
		statusCode := http.StatusOK
		if status.Status == "degraded" {
			statusCode = http.StatusServiceUnavailable
		}
		c.JSON(statusCode, gin.H{
			"service": "voicechat",
			"status":  status.Status,
			"checks":  status.Checks,
		})
	})
}
