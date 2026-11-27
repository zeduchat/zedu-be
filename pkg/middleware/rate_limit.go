package middleware

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/utility"
)

// AllowRate counts hits on key in a fixed window and reports whether the
// caller is still within limit. Redis errors count as "not allowed".
func AllowRate(rdb *redis.Client, key string, limit int64, window time.Duration) bool {
	if rdb == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	k := "ratelimit:" + key
	n, err := rdb.Incr(ctx, k).Result()
	if err != nil {
		return false
	}
	if n == 1 || rdb.TTL(ctx, k).Val() < 0 {
		rdb.Expire(ctx, k, window)
	}
	return n <= limit
}

// RateLimitByIP limits requests per client IP (c.ClientIP honours TrustedProxies).
func RateLimitByIP(rdb *redis.Client, name string, limit int64, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if cfg := config.GetConfig(); cfg != nil && slices.Contains(cfg.Server.ExemptFromThrottle, ip) {
			c.Next()
			return
		}
		if !AllowRate(rdb, fmt.Sprintf("%s:ip:%s", name, ip), limit, window) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests,
				utility.BuildErrorResponse(http.StatusTooManyRequests, "error", "Too many requests, please try again later", nil, nil))
			return
		}
		c.Next()
	}
}
