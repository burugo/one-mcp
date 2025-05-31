# 用户配置管理

本文档详细描述了One MCP项目用户配置管理功能的实现任务。

## 完成的任务

- [x] 4.1: 定义UserConfig, ConfigService模型
- [x] 4.2: 将UserConfig, ConfigService添加到AutoMigrate

## 进行中的任务

暂无进行中的任务。

## 未来任务

- [ ] 4.3: 实现UserConfig模型的CRUD方法和API处理器
- [ ] 4.4: 实现配置导出存根端点

## 实现计划

### 用户配置模型

已成功定义UserConfig和ConfigService模型并添加到AutoMigrate。模型结构如下:

```go
// UserConfig 表示用户的一个配置
type UserConfig struct {
    ID        int64     `db:"id" json:"id"`
    UserID    int64     `db:"user_id" json:"user_id"`
    Name      string    `db:"name" json:"name"`
    Config    string    `db:"config" json:"config"` // JSON格式的配置
    CreatedAt time.Time `db:"created_at" json:"created_at"`
    UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// ConfigService 表示配置与服务的关联
type ConfigService struct {
    ID        int64     `db:"id" json:"id"`
    ConfigID  int64     `db:"config_id" json:"config_id"`
    ServiceID int64     `db:"service_id" json:"service_id"`
    CreatedAt time.Time `db:"created_at" json:"created_at"`
}
```

### API处理器实现计划

将为UserConfig模型实现以下API处理器:

1. GET `/api/configs` - 获取当前用户的所有配置
2. GET `/api/configs/:id` - 获取单个配置(仅所有者可访问)
3. POST `/api/configs` - 创建新配置
4. PUT `/api/configs/:id` - 更新配置(仅所有者可访问)
5. DELETE `/api/configs/:id` - 删除配置(仅所有者可访问)
6. GET `/api/configs/:id/:client` - 导出配置(仅所有者可访问)

### 用户配置所有权验证

为确保用户只能访问自己的配置，所有配置相关API处理器将包含所有权验证逻辑:

```go
// 验证配置所有权
func verifyConfigOwnership(c *gin.Context, configID int64) bool {
    // 从JWT获取当前用户ID
    userID := getUserIDFromJWT(c)
    
    // 检查配置是否属于当前用户
    config, err := model.GetConfigByID(configID)
    if err != nil || config.UserID != userID {
        return false
    }
    
    return true
}
```

## 相关文件

- `backend/model/userconfig.go` - UserConfig模型定义
- `backend/model/configservice.go` - ConfigService模型定义
- `backend/api/handler/config.go` (待实现) - 配置API处理器 