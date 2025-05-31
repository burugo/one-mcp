# 认证系统实现

本文档详细描述了One MCP项目认证系统实现的任务。

## 完成的任务

暂无完成的任务。

## 进行中的任务

暂无进行中的任务。

## 未来任务

- [ ] 2.1: 实现JWT生成和/api/auth/login处理器
- [ ] 2.2: 实现JWT验证中间件
- [ ] 2.3: 实现令牌刷新端点
- [ ] 2.4: 实现登出机制(Redis中的JWT黑名单)
- [ ] 2.5: 保留并适配注册和登录的验证码功能

## 实现计划

### JWT认证系统

本项目将用JWT取代原有的session认证系统，主要流程如下:

1. 用户登录(用户名/密码)
2. 服务器验证凭据，生成JWT(包含用户ID和角色等信息)
3. 前端存储JWT令牌
4. 后续API请求携带JWT
5. 服务器验证JWT有效性
6. 服务器提取JWT中的用户信息和角色

### JWT生成实现

将在`service/auth_service.go`中实现JWT生成逻辑:

```go
// GenerateJWT 生成JWT令牌
func GenerateJWT(user *model.User) (string, error) {
    // 创建令牌声明(包含用户ID和角色)
    claims := jwt.MapClaims{
        "id":   user.ID,
        "role": user.Role,
        "exp":  time.Now().Add(time.Hour * 24).Unix(),
    }
    
    // 创建令牌
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    
    // 签名令牌
    return token.SignedString([]byte(config.Config.JWTSecret))
}
```

### JWT验证中间件

将在`api/middleware/auth.go`中实现JWT验证中间件:

```go
// JWTMiddleware 验证JWT令牌
func JWTMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 从请求头获取令牌
        authHeader := c.GetHeader("Authorization")
        
        // 验证令牌
        // 解析令牌
        // 将用户信息添加到上下文
        
        c.Next()
    }
}
```

### 令牌刷新和登出

将实现令牌刷新端点和登出机制:

1. `/api/auth/refresh` - 使用刷新令牌获取新的访问令牌
2. `/api/auth/logout` - 将JWT添加到Redis黑名单
   
### 验证码功能

保留并适配原有的验证码功能:

1. 保持验证码生成API
2. 保持验证码验证逻辑
3. 集成到新的JWT认证流程

## 相关文件

- `backend/service/auth_service.go` - JWT生成逻辑
- `backend/api/middleware/auth.go` - JWT验证中间件
- `backend/api/handler/auth.go` - 认证API处理器
- `backend/config/config.go` - JWT配置 