package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"one-mcp/backend/model"
	"testing"

	"one-mcp/backend/common"

	"github.com/burugo/thing"
	"github.com/burugo/thing/drivers/db/sqlite"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupOptionRouter() *gin.Engine {
	r := gin.Default()
	r.GET("/api/option/", GetOptions)
	r.PUT("/api/option/", UpdateOption)
	return r
}

func TestOptionAPI(t *testing.T) {
	common.SQLitePath = ":memory:"
	// 用内存数据库初始化 ORM
	dbAdapter, err := sqlite.NewSQLiteAdapter(":memory:")
	assert.NoError(t, err)
	thing.Configure(dbAdapter, nil)
	// 让 OptionDB 用同一个 dbAdapter
	model.OptionDB, err = thing.New[*model.Option](dbAdapter, nil)
	assert.NoError(t, err)
	// 先建表，再初始化数据库和 OptionMap
	err = thing.AutoMigrate(&model.Option{})
	assert.NoError(t, err)

	router := setupOptionRouter()

	// 1. 保存配置
	putBody := map[string]string{"key": "TestKey", "value": "TestValue"}
	bodyBytes, _ := json.Marshal(putBody)
	req, _ := http.NewRequest("PUT", "/api/option/", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	// 2. 获取配置
	req2, _ := http.NewRequest("GET", "/api/option/", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	assert.Equal(t, 200, w2.Code)

	// 打印响应内容，便于调试
	println("GET /api/option/ response:", w2.Body.String())

	var resp struct {
		Success bool                     `json:"success"`
		Message string                   `json:"message"`
		Data    []map[string]interface{} `json:"data"`
	}
	err = json.Unmarshal(w2.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.True(t, resp.Success)
	assert.NotNil(t, resp.Data)
	found := false
	for _, opt := range resp.Data {
		if opt["key"] == "TestKey" && opt["value"] == "TestValue" {
			found = true
		}
	}
	assert.True(t, found, "Should find the saved option in GET /api/option/")
}
