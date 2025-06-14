package middleware

import (
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
		// 设置到 gin.Context 中，这样 c.GetString("lang") 就能获取到
		c.Set("lang", lang)
		c.Next()
	}
}
