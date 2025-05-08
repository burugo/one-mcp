# MCP服务管理

本文档详细描述了One MCP项目MCP服务管理功能的实现任务。

## 完成的任务

- [x] 3.1: 定义MCPService模型
- [x] 3.2: 将MCPService添加到AutoMigrate

## 进行中的任务

暂无进行中的任务。

## 未来任务

- [ ] 3.3: 实现MCPService模型的CRUD方法和API处理器
- [ ] 3.4: 实现toggle端点
- [ ] 3.5: 实现配置复制存根端点
- [ ] 5.1: 在`library/proxy/`中定义基本服务结构
- [ ] 5.2: 实现基本健康检查逻辑(占位符)
- [ ] 5.3: 存储/更新健康状态(占位符，例如MCPService中的新字段)

## 实现计划

### MCPService模型

已成功定义MCPService模型并添加到AutoMigrate。模型结构如下:

```go
type MCPService struct {
    ID          int64     `db:"id" json:"id"`
    Name        string    `db:"name" json:"name"`
    Type        string    `db:"type" json:"type"`
    Description string    `db:"description" json:"description"`
    Enabled     bool      `db:"enabled" json:"enabled"`
    Config      string    `db:"config" json:"config"`
    CreatedAt   time.Time `db:"created_at" json:"created_at"`
    UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}
```

### API处理器实现计划

将为MCPService模型实现以下API处理器:

1. GET `/api/services` - 获取所有服务(所有已认证用户可访问)
2. GET `/api/services/:id` - 获取单个服务(所有已认证用户可访问)
3. POST `/api/services` - 创建新服务(仅管理员可访问)
4. PUT `/api/services/:id` - 更新服务(仅管理员可访问)
5. DELETE `/api/services/:id` - 删除服务(仅管理员可访问)
6. PUT `/api/services/:id/toggle` - 切换服务状态(仅管理员可访问)
7. GET `/api/services/:id/config/:client` - 复制服务配置(所有已认证用户可访问)

### 服务状态管理

服务状态管理包括:

1. 通过toggle端点启用/禁用服务
2. 实现健康检查逻辑监控服务状态
3. 存储和更新服务状态信息

## 相关文件

- `backend/model/mcpservice.go` - MCPService模型定义
- `backend/api/handler/service.go` (待实现) - 服务API处理器
- `backend/infrastructure/proxy/` (待实现) - 服务核心功能 