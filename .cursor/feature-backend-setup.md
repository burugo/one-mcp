# 后端基础设置与模型实现

本文档详细描述了One MCP项目后端基础设置和模型实现的任务。

## 完成的任务

- [x] 1.1: 适配`gin-template`结构到MVC布局
- [x] 1.2: 定义/重构User模型(带有`Role int`字段)，定义Role常量，移除Role模型
- [x] 1.3: 使用GORM AutoMigrate实现初始数据库模式(仅User表，SQLite)
- [x] 1.4: 实现Admin用户种子逻辑
- [x] 3.1: 定义MCPService模型
- [x] 3.2: 将MCPService添加到AutoMigrate
- [x] 4.1: 定义UserConfig, ConfigService模型
- [x] 4.2: 将UserConfig, ConfigService添加到AutoMigrate
- [x] 10.1: 优化User模型中的冗余代码，移除不必要的中间变量

## 进行中的任务

- [ ] 1.5: 根据MVC模式增强User模型的CRUD方法
- [ ] 1.6: 验证/实现密码哈希(bcrypt)
- [ ] 10.2: 升级@thing.mdc依赖，确保model标签支持省略db字段(自动snake_case)，并支持index/unique复合索引声明
- [ ] 10.3: 更新所有模型定义，去除冗余db标签，复合索引用法符合最新Thing规范
- [ ] 10.4: 验证AutoMigrate及索引创建逻辑，确保表结构和索引与模型定义一致

## 未来任务

- [ ] 5.1: 在`library/proxy/`中定义基本服务结构
- [ ] 5.2: 实现基本健康检查逻辑(占位符)
- [ ] 5.3: 存储/更新健康状态(占位符，例如MCPService中的新字段)

## 实现计划

### User模型优化（已完成）

当前User模型中大量使用了`userThing := UserDB`这样的中间变量赋值，然后再使用`userThing`进行操作。这种模式在所有方法中都重复出现，例如：

```go
func GetMaxUserId() int64 {
    userThing := UserDB
    users, err := userThing.Order("id DESC").Fetch(0, 1)
    // ...
}
```

这种写法产生了不必要的冗余，因为：
1. 中间变量`userThing`没有带来任何额外价值
2. 每个函数都重复这个赋值操作，增加了代码行数
3. 维护时需要修改更多的代码

#### 优化方案

直接使用全局变量`UserDB`，无需中间变量：

```go
func GetMaxUserId() int64 {
    users, err := UserDB.Order("id DESC").Fetch(0, 1)
    // ...
}
```

这种优化可以:
- 减少代码行数
- 提高代码可读性
- 减少不必要的变量声明
- 保持一致性

#### 优化结果

已成功移除User模型中所有不必要的`userThing := UserDB`中间变量赋值，直接使用全局变量`UserDB`。

### Thing ORM模型标签与索引功能更新

目前模型定义中存在大量冗余的`db`标签，而最新的Thing ORM支持自动snake_case转换和更简洁的索引声明。需要升级依赖并更新所有模型定义。

## 相关文件

- `backend/model/user.go` - 用户模型文件
- `backend/model/mcpservice.go` - MCP服务模型文件
- `backend/model/userconfig.go` - 用户配置模型文件
- `backend/model/configservice.go` - 配置服务模型文件
- `backend/common/constants.go` - 常量定义文件 