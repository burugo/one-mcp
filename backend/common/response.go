package common

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// API响应的标准格式
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// 时间格式常量
const (
	RFC3339MilliZ = "2006-01-02T15:04:05.000Z07:00"
)

// RespSuccess 响应成功，返回数据
func RespSuccess(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Message: "",
		Data:    data,
	})
}

// RespSuccessStr 响应成功，返回消息
func RespSuccessStr(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Message: msg,
	})
}

// RespError 响应错误，包含错误信息
func RespError(c *gin.Context, statusCode int, msg string, err error) {
	errMsg := msg
	if err != nil {
		errMsg = msg + ": " + err.Error()
	}

	c.JSON(statusCode, APIResponse{
		Success: false,
		Message: errMsg,
	})
}

// RespErrorStr 响应错误，只包含错误消息
func RespErrorStr(c *gin.Context, statusCode int, msg string) {
	c.JSON(statusCode, APIResponse{
		Success: false,
		Message: msg,
	})
}

// FormatTime 格式化时间为RFC3339MilliZ格式
func FormatTime(t time.Time) string {
	return t.Format(RFC3339MilliZ)
}
