# One MCP 开发指南

本文档介绍如何设置和运行 One MCP 开发环境。

## 项目结构

```
one-mcp/
├── backend/         # Go后端代码
├── frontend/        # React前端代码
├── tests/           # 测试文件
├── main.go          # 程序入口点
├── build.sh         # 构建脚本
└── dev.sh           # 开发环境启动脚本
```

## 开发环境设置

### 安装依赖

1. 安装Go依赖：
```bash
go mod tidy
```

2. 安装前端依赖：
```bash
cd frontend
npm install
cd ..
```

### 运行开发环境

使用开发脚本同时启动前端和后端服务器：

```bash
# 使用默认3000端口
./dev.sh

# 或指定自定义端口
PORT=8080 ./dev.sh
```

- 后端服务器将在 http://localhost:$PORT 上运行（默认为3000）
- 前端开发服务器将在另一个端口上运行（通常是5173，请查看控制台输出）
- 前端开发服务器会自动将API请求代理到后端（使用相同的PORT环境变量）

### 构建生产版本

要构建生产版本并运行服务器：

```bash
# 使用默认3000端口
./build.sh

# 或指定自定义端口
PORT=8080 ./build.sh
```

这将：
1. 构建前端代码并输出到 `frontend/dist` 目录
2. 启动后端服务器，后端服务器将提供编译好的前端资源

服务器将在 http://localhost:$PORT 上运行（默认为3000）。

## API 访问

所有API端点都以 `/api/` 开头，并由后端处理。在开发模式下，Vite开发服务器会自动将这些请求代理到后端服务器。

### API使用

前端代码应使用我们提供的API工具函数访问后端API：

```typescript
// 导入API工具
import api from '@/utils/api';

// 使用API
async function fetchData() {
  try {
    const response = await api.get('/endpoint');
    // response已经自动提取data部分
    console.log(response);
  } catch (error) {
    // 错误已在api.ts中得到统一处理
    console.error('Could not fetch data');
  }
}
```

## 端口配置

项目中的前端和后端端口配置是同步的：

- 默认使用`3000`端口
- 可以通过环境变量`PORT`来更改
- 前端开发服务器在开发模式下会使用一个不同的端口运行（如5173），但会自动将API请求代理到后端的PORT端口
- 生产模式下，前端和后端共用相同端口，前端资源被嵌入到后端二进制文件中

## 注意事项

- 构建过程中使用了Go的嵌入功能将前端文件嵌入到后端二进制文件中
- `frontend/dist` 目录中的文件在每次构建时都会被重新创建
- API请求将自动代理到后端服务器，无需手动配置CORS 