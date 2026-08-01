package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"

	"one-mcp/backend/common"
	"one-mcp/backend/library/proxy"
	"one-mcp/backend/model"

	"github.com/gin-gonic/gin"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUninstallServiceUnregistersServiceWithoutInstalledVersion(t *testing.T) {
	originalSQLitePath := common.SQLitePath
	common.SQLitePath = filepath.Join(t.TempDir(), "market_uninstall_test.db")
	t.Cleanup(func() { common.SQLitePath = originalSQLitePath })
	require.NoError(t, model.InitDB())

	service := &model.MCPService{
		Name:             "uninstall-empty-version",
		DisplayName:      "Uninstall Empty Version",
		Type:             model.ServiceType("test"),
		Enabled:          true,
		InstalledVersion: "",
	}
	require.NoError(t, model.CreateService(service))

	serviceManager := proxy.GetServiceManager()
	require.NoError(t, serviceManager.RegisterService(context.Background(), service))
	proxy.GetHealthCacheManager().SetServiceHealth(service.ID, &proxy.ServiceHealth{Status: proxy.StatusHealthy})
	proxy.GetToolsCacheManager().SetServiceTools(service.ID, &proxy.ToolsCacheEntry{Tools: []mcp.Tool{{Name: "stale"}}})
	t.Cleanup(func() {
		_ = serviceManager.UnregisterService(context.Background(), service.ID)
		proxy.GetHealthCacheManager().DeleteServiceHealth(service.ID)
		proxy.GetToolsCacheManager().DeleteServiceTools(service.ID)
	})

	request := newJSONRequest(t, http.MethodPost, "/api/mcp_market/uninstall", map[string]any{"service_id": service.ID})
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request
	ctx.Set("lang", "en")

	UninstallService(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)
	_, err := serviceManager.GetService(service.ID)
	assert.ErrorIs(t, err, proxy.ErrServiceNotFound)
	_, err = serviceManager.ForceCheckServiceHealth(service.ID)
	assert.ErrorIs(t, err, proxy.ErrServiceNotRegistered)
	_, healthFound := proxy.GetHealthCacheManager().GetServiceHealth(service.ID)
	assert.False(t, healthFound)
	_, toolsFound := proxy.GetToolsCacheManager().GetServiceTools(service.ID)
	assert.False(t, toolsFound)

	deletedService, err := model.GetServiceByID(service.ID)
	require.NoError(t, err)
	assert.True(t, deletedService.Deleted)
	assert.False(t, deletedService.Enabled)
	assert.Error(t, serviceManager.RegisterService(context.Background(), deletedService))

	healthRequest, err := http.NewRequest(http.MethodPost, "/api/mcp_services/1/health/check", nil)
	require.NoError(t, err)
	healthRecorder := httptest.NewRecorder()
	healthCtx, _ := gin.CreateTestContext(healthRecorder)
	healthCtx.Request = healthRequest
	healthCtx.Params = gin.Params{{Key: "id", Value: strconv.FormatInt(service.ID, 10)}}
	healthCtx.Set("lang", "en")
	CheckMCPServiceHealth(healthCtx)
	assert.Equal(t, http.StatusBadRequest, healthRecorder.Code)
	_, err = serviceManager.GetService(service.ID)
	assert.ErrorIs(t, err, proxy.ErrServiceNotFound)
}

func TestSanitizeServiceName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Only spaces",
			input:    "   ",
			expected: "",
		},
		{
			name:     "Simple name",
			input:    "MyService",
			expected: "myservice",
		},
		{
			name:     "Name with spaces",
			input:    "My Service Name",
			expected: "my-service-name",
		},
		{
			name:     "Name with multiple spaces",
			input:    "My   Service    Name",
			expected: "my-service-name",
		},
		{
			name:     "Name with tabs and newlines",
			input:    "My\tService\nName\rTest",
			expected: "my-service-name-test",
		},
		{
			name:     "Name with leading/trailing spaces and dashes",
			input:    "  -My Service-  ",
			expected: "my-service",
		},
		{
			name:     "Name with multiple consecutive dashes",
			input:    "My--Service---Name",
			expected: "my-service-name",
		},
		{
			name:     "Chinese characters",
			input:    "我的服务 Test",
			expected: "我的服务-test",
		},
		{
			name:     "Mixed case with special chars",
			input:    "MyService_123 Test",
			expected: "myservice_123-test",
		},
		{
			name:     "Only dashes",
			input:    "---",
			expected: "",
		},
		{
			name:     "Complex mixed input",
			input:    "  My\t\tService\n\n  Name  --  Test  ",
			expected: "my-service-name-test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeServiceName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCreateCustomService_DuplicateName(t *testing.T) {
	// 这个测试需要数据库连接，所以我们先跳过实际的数据库操作
	// 在实际环境中，你需要设置测试数据库
	t.Skip("需要数据库连接的集成测试")

	// 设置Gin为测试模式
	gin.SetMode(gin.TestMode)

	// 创建测试路由
	router := gin.New()
	router.POST("/api/mcp_market/custom_service", CreateCustomService)

	// 第一次创建服务的请求体
	serviceRequest := map[string]interface{}{
		"name":      "Test Service",
		"type":      "stdio",
		"command":   "echo",
		"arguments": "hello",
	}

	// 第一次请求 - 应该成功
	reqBody1, _ := json.Marshal(serviceRequest)
	req1 := httptest.NewRequest("POST", "/api/mcp_market/custom_service", bytes.NewBuffer(reqBody1))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	// 第二次请求 - 应该返回冲突错误
	reqBody2, _ := json.Marshal(serviceRequest)
	req2 := httptest.NewRequest("POST", "/api/mcp_market/custom_service", bytes.NewBuffer(reqBody2))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	// 验证第二次请求返回冲突状态码
	assert.Equal(t, http.StatusConflict, w2.Code)

	// 验证响应消息包含冲突信息
	var response map[string]interface{}
	err := json.Unmarshal(w2.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["message"], "已存在")
}

// MockCreateCustomService 用于测试的模拟函数，不依赖数据库
func MockCreateCustomService(c *gin.Context, mockGetServiceByName func(string) (*model.MCPService, error)) {

	type CustomServiceRequest struct {
		Name         string `json:"name" binding:"required"`
		Type         string `json:"type" binding:"required"`
		Command      string `json:"command"`
		Arguments    string `json:"arguments"`
		Environments string `json:"environments"`
		URL          string `json:"url"`
		Headers      string `json:"headers"`
	}

	var requestBody CustomServiceRequest
	if err := c.ShouldBindJSON(&requestBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid_request_data"})
		return
	}

	// 清理和验证服务名称
	sanitizedName := sanitizeServiceName(requestBody.Name)
	if sanitizedName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "service_name_cannot_be_empty"})
		return
	}

	// 使用模拟的数据库查询函数
	existingService, err := mockGetServiceByName(sanitizedName)
	if err == nil && existingService != nil {
		c.JSON(http.StatusConflict, gin.H{
			"success": false,
			"message": "service_name_already_exists: " + sanitizedName,
		})
		return
	}

	// 模拟成功创建
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "自定义服务创建成功",
		"data": gin.H{
			"sanitized_name": sanitizedName,
			"display_name":   requestBody.Name,
		},
	})
}

func TestCreateCustomService_DuplicateNameMock(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 模拟已存在的服务
	existingServices := map[string]*model.MCPService{
		"test-service": {
			Name: "test-service",
		},
	}

	mockGetServiceByName := func(name string) (*model.MCPService, error) {
		if service, exists := existingServices[name]; exists {
			return service, nil
		}
		return nil, model.ErrRecordNotFound
	}

	// 创建测试路由
	router := gin.New()
	router.POST("/test", func(c *gin.Context) {
		MockCreateCustomService(c, mockGetServiceByName)
	})

	// 测试用例1: 创建新服务 - 应该成功
	t.Run("Create new service", func(t *testing.T) {
		serviceRequest := map[string]interface{}{
			"name": "New Service",
			"type": "stdio",
		}

		reqBody, _ := json.Marshal(serviceRequest)
		req := httptest.NewRequest("POST", "/test", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["success"].(bool))

		data := response["data"].(map[string]interface{})
		assert.Equal(t, "new-service", data["sanitized_name"])
		assert.Equal(t, "New Service", data["display_name"])
	})

	// 测试用例2: 创建重复服务 - 应该返回冲突错误
	t.Run("Create duplicate service", func(t *testing.T) {
		serviceRequest := map[string]interface{}{
			"name": "Test Service", // 这会被清理为 "test-service"，与已存在的服务冲突
			"type": "stdio",
		}

		reqBody, _ := json.Marshal(serviceRequest)
		req := httptest.NewRequest("POST", "/test", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.False(t, response["success"].(bool))
		assert.Contains(t, response["message"], "service_name_already_exists")
	})

	// 测试用例3: 空名称 - 应该返回错误
	t.Run("Empty service name", func(t *testing.T) {
		serviceRequest := map[string]interface{}{
			"name": "   ",
			"type": "stdio",
		}

		reqBody, _ := json.Marshal(serviceRequest)
		req := httptest.NewRequest("POST", "/test", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.False(t, response["success"].(bool))
		assert.Contains(t, response["message"], "service_name_cannot_be_empty")
	})
}

func TestIsDirectUVSource(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{name: "pypi simple", input: "black", expected: false},
		{name: "pypi with version", input: "black==24.0.0", expected: false},
		{name: "pypi with extras", input: "mypkg[cli]", expected: false},
		{name: "pypi with range", input: "mypkg>=1.0", expected: false},
		{name: "git url", input: "git+https://github.com/org/repo", expected: true},
		{name: "https wheel", input: "https://example.com/pkg.whl", expected: true},
		{name: "file url", input: "file:///tmp/pkg.whl", expected: true},
		{name: "relative path", input: "./local-package", expected: true},
		{name: "parent path", input: "../local-package", expected: true},
		{name: "absolute path", input: "/opt/mcp/pkg", expected: true},
		{name: "home path", input: "~/mcp/pkg", expected: true},
		{name: "direct reference git", input: "mypkg @ git+https://github.com/org/repo", expected: true},
		{name: "direct reference file", input: "mypkg @ file:///tmp/pkg.whl", expected: true},
		{name: "direct reference https", input: "mypkg @ https://example.com/pkg.whl", expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isDirectUVSource(tt.input))
		})
	}
}

func TestUVXSourceClassificationCases(t *testing.T) {
	cases := []struct {
		name       string
		sourceSpec string
		isPyPI     bool
	}{
		{name: "from pypi package", sourceSpec: "black", isPyPI: true},
		{name: "from pypi extras", sourceSpec: "mypkg[cli]", isPyPI: true},
		{name: "from git url", sourceSpec: "git+https://github.com/org/repo", isPyPI: false},
		{name: "from local path", sourceSpec: "./local-package", isPyPI: false},
		{name: "from wheel url", sourceSpec: "https://example.com/pkg.whl", isPyPI: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, !tc.isPyPI, isDirectUVSource(tc.sourceSpec))
		})
	}
}

func TestResolveUVSourceSpec(t *testing.T) {
	tests := []struct {
		name        string
		packageName string
		customArgs  []string
		expected    string
	}{
		{name: "use from source spec", packageName: "grok-search", customArgs: []string{"--from", "git+https://github.com/org/repo", "grok-search"}, expected: "git+https://github.com/org/repo"},
		{name: "use first arg when no from", packageName: "ignored", customArgs: []string{"./local-package", "serve"}, expected: "./local-package"},
		{name: "fallback to package name", packageName: "black", customArgs: nil, expected: "black"},
		{name: "ignore leading flag and fallback", packageName: "black", customArgs: []string{"--verbose"}, expected: "black"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, resolveUVSourceSpec(tt.packageName, tt.customArgs))
		})
	}
}
