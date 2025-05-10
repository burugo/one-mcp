package middleware

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
)

// LangMiddleware 注入 lang 到 context
func LangMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		lang := c.GetHeader("Accept-Language")
		if lang == "" {
			lang = "zh-CN" // 默认中文
		} else {
			// 只取第一个语言
			lang = strings.Split(lang, ",")[0]
		}
		ctx := context.WithValue(c.Request.Context(), "lang", lang)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
