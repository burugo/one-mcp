# i18n国际化错误处理框架实现

本文档详细描述了One MCP项目国际化(i18n)错误处理框架的实现任务。

## 背景说明

当前项目中存在大量硬编码的中文错误消息，这对国际化支持不友好。我们需要实现一个完整的i18n国际化错误处理框架，支持多语言错误消息，便于项目走向国际化。

## 完成的任务

- [x] 1. 创建错误码常量文件 `backend/common/errors/code.go`
- [x] 2. 创建多语言资源文件结构
- [x] 3. 实现i18n错误处理库
- [x] 4. 示例实现：修改User模型中GetUserById和DeleteUserById方法使用i18n
- [x] 5. 添加i18n单元测试并验证功能正常

## 进行中的任务

- [ ] 6. 继续将其他模型和函数中的错误处理改为使用i18n框架

## 未来任务

- [ ] 7. 修改API处理器，从HTTP请求中获取用户语言偏好
- [ ] 8. 实现i18n上下文传递机制，避免在每个函数中传递lang参数
- [ ] 9. 添加更多语言支持（如需要）

## 实现计划

### 当前问题

在`backend/model/user.go`文件中存在大量中文错误消息：

```go
if id == 0 {
    return nil, errors.New("id 为空！")
}
```

```go
if email == "" || password == "" {
    return errors.New("邮箱地址或密码为空！")
}
```

这些硬编码的中文错误消息对国际化支持不友好，难以在英语环境下使用，也不利于后续添加其他语言支持。

### 已实现解决方案

我们实现了完整的i18n框架，包括：

1. 创建错误码常量和国际化资源文件:
```go
// backend/common/errors/code.go
const (
    ErrEmptyID             = "ERR_EMPTY_ID"
    ErrEmptyCredentials    = "ERR_EMPTY_CREDENTIALS"
    ErrUserNotFound        = "ERR_USER_NOT_FOUND"
    // ...
)
```

2. 建立语言资源文件:
```json
// locales/en.json
{
    "ERR_EMPTY_ID": "ID is empty",
    "ERR_EMPTY_CREDENTIALS": "Username or password is empty",
    "ERR_USER_NOT_FOUND": "User not found"
}

// locales/zh.json
{
    "ERR_EMPTY_ID": "ID 为空",
    "ERR_EMPTY_CREDENTIALS": "用户名或密码为空",
    "ERR_USER_NOT_FOUND": "未找到用户"
}
```

3. 创建i18n错误处理函数:
```go
// backend/common/i18n/errors.go
package i18n

func New(code string, lang string, args ...interface{}) *I18nError {
    msg := Translate(code, lang, args...)
    return &I18nError{
        Code: code,
        Msg:  msg,
        Err:  errors.New(msg),
    }
}

func Wrap(err error, code string, lang string, args ...interface{}) *I18nError {
    msg := Translate(code, lang, args...)
    return &I18nError{
        Code: code,
        Msg:  msg,
        Err:  err,
    }
}
```

## 实现结果

成功实现了国际化错误处理框架，包括：

1. **错误码常量**：在`backend/common/errors/code.go`中定义了统一的错误码
2. **语言资源文件**：在`locales/`目录下创建了英文和中文错误消息文件
3. **i18n核心库**：实现了`backend/common/i18n/i18n.go`和`backend/common/i18n/error.go`
4. **改进的错误处理**：示例改进了`GetUserById`和`DeleteUserById`函数
5. **单元测试**：添加了测试并验证国际化功能正常工作

### 使用示例

```go
// 改进前
func GetUserById(id int64, selectAll bool) (*User, error) {
    if id == 0 {
        return nil, errors.New("id 为空！")
    }
    return UserDB.ByID(id)
}

// 改进后
func GetUserById(id int64, selectAll bool, lang string) (*User, error) {
    if id == 0 {
        return nil, i18n.New(mcperrors.ErrEmptyID, lang)
    }
    user, err := UserDB.ByID(id)
    if err != nil {
        return nil, i18n.Wrap(err, mcperrors.ErrUserNotFound, lang)
    }
    return user, nil
}
```

### 错误消息示例

```
// 英文错误消息 (lang="en")
"ID is empty"
"User not found"

// 中文错误消息 (lang="zh")
"ID 为空"
"未找到用户"
```

## 相关文件

- `backend/common/errors/code.go` (新增) - 错误码常量定义
- `backend/common/i18n/i18n.go` (新增) - i18n核心功能
- `backend/common/i18n/error.go` (新增) - i18n错误类型
- `locales/en.json` (新增) - 英文错误消息
- `locales/zh.json` (新增) - 中文错误消息
- `backend/model/user.go` (修改) - 示例实现
- `backend/common/i18n/i18n_test.go` (测试) - 单元测试验证 