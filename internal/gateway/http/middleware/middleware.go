package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"voicechat/internal/auth"
	"voicechat/pkg/utils"

	"github.com/gin-gonic/gin"
)

// RateLimiter 实现简单的滑动窗口限流器。
// 按 IP 地址跟踪请求时间戳并强制执行请求限制。
type RateLimiter struct {
	requests map[string][]time.Time // IP -> 请求时间戳列表
	mu      sync.RWMutex
	limit   int                    // 窗口期内允许的最大请求数
	window  time.Duration          // 限流时间窗口
}

// NewRateLimiter 创建具有指定限制和时间窗口的新限流器。
// 启动后台清理 goroutine 以移除过期的条目。
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
	go rl.cleanup()
	return rl
}

// cleanup 定期从所有 IP 移除过期的请求时间戳。
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(time.Minute)
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, times := range rl.requests {
			var valid []time.Time
			for _, t := range times {
				if now.Sub(t) < rl.window {
					valid = append(valid, t)
				}
			}
			if len(valid) == 0 {
				delete(rl.requests, ip)
			} else {
				rl.requests[ip] = valid
			}
		}
		rl.mu.Unlock()
	}
}

// Allow 检查来自给定 IP 的请求是否应该被允许。
// 如果在限制内返回 true，超过限制则返回 false。
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	times := rl.requests[ip]

	var valid []time.Time
	for _, t := range times {
		if now.Sub(t) < rl.window {
			valid = append(valid, t)
		}
	}

	if len(valid) >= rl.limit {
		rl.requests[ip] = valid
		return false
	}

	valid = append(valid, now)
	rl.requests[ip] = valid
	return true
}

// RateLimitMiddleware 创建强制按客户端 IP 限流的 Gin 中间件。
func RateLimitMiddleware(limiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !limiter.Allow(ip) {
			utils.NewErrorResponse(c, utils.ErrCodeRateLimit, "请求过于频繁", http.StatusTooManyRequests)
			c.Abort()
			return
		}
		c.Next()
	}
}

// AuthMiddleware 创建验证 JWT 令牌的 Gin 中间件。
// 验证成功后会在上下文设置 user_id 和 email。
func AuthMiddleware(jwtService *auth.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			utils.NewErrorResponse(c, utils.ErrCodeAuthInvalidToken, "缺少认证令牌", http.StatusUnauthorized)
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			utils.NewErrorResponse(c, utils.ErrCodeAuthInvalidToken, "无效的认证格式", http.StatusUnauthorized)
			c.Abort()
			return
		}

		token := parts[1]
		claims, err := jwtService.ValidateToken(token)
		if err != nil {
			utils.NewErrorResponse(c, utils.ErrCodeAuthInvalidToken, "无效的令牌", http.StatusUnauthorized)
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Next()
	}
}

// CORSMiddleware 创建处理 CORS 预检请求的 Gin 中间件。
// 使用适当的 CORS 头允许所有来源。
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// LoggerMiddleware 创建占位符日志中间件。
// 目前是空操作，可扩展为请求日志记录。
func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}
