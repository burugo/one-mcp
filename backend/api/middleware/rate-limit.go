package middleware

import (
	"fmt"
	"net/http"
	"one-mcp/backend/common"
	"time"

	"github.com/burugo/thing"
	"github.com/gin-gonic/gin"
)

func thingCacheRateLimiter(c *gin.Context, maxRequestNum int, durationSeconds int64, mark string) {
	cacheClient := thing.Cache()
	if cacheClient == nil {
		common.SysError("[RateLimit] thing.Cache() returned nil, rate limiting cannot proceed.")
		c.Status(http.StatusInternalServerError)
		c.Abort()
		return
	}

	key := "rateLimit:" + mark + c.ClientIP()
	ctx := c.Request.Context()

	count, err := cacheClient.Incr(ctx, key)
	if err != nil {
		common.SysError(fmt.Sprintf("[RateLimit] Error incrementing cache for key %s: %v", key, err))
		c.Status(http.StatusInternalServerError)
		c.Abort()
		return
	}

	if count == 1 {
		windowDuration := time.Duration(durationSeconds) * time.Second
		expireErr := cacheClient.Expire(ctx, key, windowDuration)
		if expireErr != nil {
			common.SysError(fmt.Sprintf("[RateLimit] Error setting expiration for key %s: %v", key, expireErr))
		}
	}

	if count > int64(maxRequestNum) {
		c.Status(http.StatusTooManyRequests)
		c.Abort()
		return
	}
}

func rateLimitFactory(maxRequestNum int, duration int64, mark string) func(c *gin.Context) {
	return func(c *gin.Context) {
		thingCacheRateLimiter(c, maxRequestNum, int64(duration), mark)
	}
}

func GlobalWebRateLimit() func(c *gin.Context) {
	return rateLimitFactory(common.GlobalWebRateLimitNum, common.GlobalWebRateLimitDuration, "GW")
}

func GlobalAPIRateLimit() func(c *gin.Context) {
	return rateLimitFactory(common.GlobalApiRateLimitNum, common.GlobalApiRateLimitDuration, "GA")
}

func CriticalRateLimit() func(c *gin.Context) {
	return rateLimitFactory(common.CriticalRateLimitNum, common.CriticalRateLimitDuration, "CT")
}

func DownloadRateLimit() func(c *gin.Context) {
	return rateLimitFactory(common.DownloadRateLimitNum, common.DownloadRateLimitDuration, "DW")
}

func UploadRateLimit() func(c *gin.Context) {
	return rateLimitFactory(common.UploadRateLimitNum, common.UploadRateLimitDuration, "UP")
}
