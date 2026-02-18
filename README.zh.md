<p align="right">
    <a href="./README.md">English</a> | <strong>中文</strong>
</p>

# One MCP

<div align="center">

**One MCP** - 模型上下文协议 (MCP) 服务的集中式代理

*✨ 从单一界面管理、监控和配置您的 MCP 服务 ✨*

<br />

[![Go Report Card](https://goreportcard.com/badge/github.com/burugo/one-mcp?style=flat-square)](https://goreportcard.com/report/github.com/burugo/one-mcp)
[![Go CI](https://img.shields.io/github/actions/workflow/status/burugo/one-mcp/go.yml?style=flat-square&logo=github&label=Go%20CI)](https://github.com/burugo/one-mcp/actions)
[![GitHub license](https://img.shields.io/github/license/burugo/one-mcp?style=flat-square)](https://github.com/burugo/one-mcp/blob/main/LICENSE)

[![Go](https://img.shields.io/badge/Go-1.19+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://golang.org/)
[![React](https://img.shields.io/badge/React-18+-61DAFB?style=flat-square&logo=react&logoColor=black)](https://reactjs.org/)
[![TypeScript](https://img.shields.io/badge/TypeScript-5+-3178C6?style=flat-square&logo=typescript&logoColor=white)](https://www.typescriptlang.org/)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=flat-square&logo=docker&logoColor=white)](https://hub.docker.com/r/buru2020/one-mcp)

</div>

<p align="center">
  <a href="#功能特性">功能特性</a> •
  <a href="#快速开始">快速开始</a> •
  <a href="#安装部署">安装部署</a> •
  <a href="#配置说明">配置说明</a> •
  <a href="#开发指南">开发指南</a> •
  <a href="#贡献代码">贡献代码</a>
</p>

---

## 概述

One MCP 是一个全面的模型上下文协议 (MCP) 服务管理平台。作为集中式代理，它让您可以从各种提供商发现、安装、配置和监控 MCP 服务。使用 Go 和 React 构建，提供强大的后端功能和直观的 Web 界面。

<!-- 截图占位符 - 仪表板/主界面 -->
![Screenshot](./images/dashboard.png)

## 功能特性

- **服务管理** — 从市场或自定义源安装、配置和监控 MCP 服务（stdio / SSE / 可流式 HTTP）
- **服务组合** — 将多个 MCP 服务组合为单一端点；可导出为 Anthropic Skills，支持 Claude Code 和 Droid
- **数据分析** — 实时跟踪使用量、请求速率、响应时间和系统健康状态
- **多用户 & OAuth** — 基于角色的访问控制，支持 GitHub / Google 登录
- **灵活部署** — SQLite（默认）/ MySQL / PostgreSQL，可选 Redis，支持 Docker，国际化（中 / 英）

<!-- 截图占位符 - 服务管理界面 -->
![Screenshot](./images/services.png)
![Screenshot](./images/copy_config.png)

### 服务组合

创建服务组合，将多个 MCP 服务组合并导出为技能：

![Screenshot](./images/group.png)

## 快速开始

### 使用 Homebrew（macOS & Linux）

```bash
# 添加 tap
brew tap burugo/tap

# 安装 one-mcp
brew install one-mcp

# 以后台服务启动（默认端口：3000）
brew services start one-mcp

# 停止服务
brew services stop one-mcp
```

如果端口 `3000` 已被占用，可用自定义端口重启服务：

```bash
ONE_MCP_PORT=3001 brew services restart one-mcp
```

访问地址：http://localhost:3000（或你的自定义端口）

### 使用 Docker（推荐）

```bash
# 使用 Docker 运行
docker run --name one-mcp -d \
  --restart always \
  -p 3000:3000 \
  -v $(pwd)/data:/data \
  buru2020/one-mcp:latest

# 访问应用程序
open http://localhost:3000
```

### 手动安装

```bash
# 克隆仓库
git clone https://github.com/burugo/one-mcp.git
cd one-mcp

# 设置环境
cp .env_example .env

# 安装依赖并构建
go mod tidy
cd frontend && npm install && npm run build && cd ..

# 运行应用程序
go run main.go
```

**默认登录**：用户名 `root`，密码 `123456`

## 安装部署

### 前置要求

#### Homebrew 安装（macOS & Linux）

- 已安装 **Homebrew**

#### 手动安装

- **Go**：版本 1.19 或更高
- **Node.js**：版本 16 或更高
- **数据库**：SQLite（默认）、MySQL 或 PostgreSQL
- **Redis**：可选，用于分布式缓存

### 环境配置

从模板创建 `.env` 文件：

```bash
cp .env_example .env
```

主要配置选项：

```bash
# 服务器配置
PORT=3000

# 数据库（可选，默认使用 SQLite）
# SQLite(default)
# SQLITE_PATH=/data/one-mcp.db
# MySQL:
# SQL_DSN=root:password@tcp(localhost:3306)/one_mcp
# PostgreSQL:
# SQL_DSN=postgres://username:password@localhost/database_name?sslmode=disable
# Redis（可选，代替本地缓存，用于生产环境）
REDIS_CONN_STRING=redis://localhost:6379

# GitHub API（可选，在服务器查询npm所在github主页的star数，不填会有rate limit的问题）
GITHUB_TOKEN=your-github-token
```

### Homebrew 安装（macOS & Linux）

```bash
# 添加 tap
brew tap burugo/tap

# 安装 one-mcp
brew install one-mcp

# 前台运行
one-mcp --port 3000

# 或作为系统服务运行（默认端口：3000）
brew services start one-mcp

# 当 3000 被占用时，使用自定义端口
ONE_MCP_PORT=3001 brew services restart one-mcp
```

### Docker 部署

```bash
# 构建 Docker 镜像
docker build -t one-mcp .

# 使用 docker-compose 运行（推荐）
docker-compose up -d

# 或直接运行
docker run -d \
  --name one-mcp \
  -p 3000:3000 \
  -v ./data:/data \
  -e PORT=3000 \
  one-mcp
```

### 手动部署

1. **构建应用程序**：
   ```bash
   ./deploy/build.sh
   ```

2. **运行服务器**：
   ```bash
   ./one-mcp --port 3000
   ```

3. **访问应用程序**：
   在浏览器中打开 http://localhost:3000

## 配置说明

### 运行时配置（`~/.config/one-mcp/config.ini`）

One MCP 支持从以下 INI 文件读取运行时配置：

```bash
~/.config/one-mcp/config.ini
```

运行时优先级为：

```text
defaults < config file < environment variables < flags
```

- `defaults`：内置默认值（例如端口 `3000`）
- `config file`：`~/.config/one-mcp/config.ini` 中的配置
- `environment variables`：如 `PORT`、`SQLITE_PATH`、`ENABLE_GZIP`
- `flags`：命令行参数（如 `--port`，优先级最高）

首次启动时，One MCP 会自动创建最简默认 `config.ini`。

`config.ini` 示例：

```ini
PORT=3000
SQLITE_PATH=one-mcp.db
ENABLE_GZIP=true
```

说明：

- 运行时文件配置仅读取 `~/.config/one-mcp/config.ini`。
- Homebrew service 的值（`ONE_MCP_PORT`、`--port`）仍会覆盖 `config.ini`。

### OAuth 设置

#### GitHub OAuth
1. 在 https://github.com/settings/applications/new 创建 GitHub OAuth 应用
2. 设置主页 URL：`http://your-domain.com`
3. 设置授权回调 URL：`http://your-domain.com/oauth/github`
4. 在应用程序首选项中配置

#### Google OAuth
1. 在 https://console.developers.google.com/ 创建凭据
2. 设置授权的 JavaScript 来源：`http://your-domain.com`
3. 设置授权的重定向 URI：`http://your-domain.com/oauth/google`
4. 在应用程序首选项中配置

### 数据库配置

#### SQLite（默认）
无需额外配置。数据库文件在 `./data/one-mcp.db` 创建。

#### MySQL
```bash
SQL_DSN=username:password@tcp(localhost:3306)/database_name
```

#### PostgreSQL
```bash
SQL_DSN=postgres://username:password@localhost/database_name?sslmode=disable
```

## API 文档

应用程序为所有功能提供 RESTful API：

- **基础 URL**：`http://localhost:3000/api`
- **认证**：Bearer 令牌（通过登录获取）
- **内容类型**：`application/json`

### 主要端点

- `POST /api/auth/login` - 用户认证
- `GET /api/services` - 列出已安装的服务
- `POST /api/services` - 安装新服务
- `GET /api/market/search` - 搜索市场
- `GET /api/analytics/usage` - 使用统计

## 开发指南

### 开发环境

```bash
# 启动开发服务器
./run.sh

# 这将启动：
# - 后端服务器在 :3000
# - 前端开发服务器在 :5173（支持热重载）
```

### 项目结构

```
one-mcp/
├── backend/         # Go 后端代码
├── frontend/        # React 前端代码  
├── data/           # 数据库和上传文件
├── main.go         # 应用程序入口点
├── deploy/         # 部署脚本
│   ├── build.sh        # 生产构建脚本
│   ├── local-deploy.sh # 本地部署脚本
│   └── push-to-dockerhub.sh # Docker Hub 推送脚本
└── run.sh          # 开发脚本
```

### 测试

```bash
# 前端测试
cd frontend && npm test

# 后端测试
go test ./...
```

详细的开发说明请参见 [DEVELOPMENT.md](./DEVELOPMENT.md)。

## 贡献代码

我们欢迎贡献！请参见我们的贡献指南：

1. **Fork** 仓库
2. **创建** 功能分支 (`git checkout -b feature/amazing-feature`)
3. **提交** 更改 (`git commit -m 'Add amazing feature'`)
4. **推送** 到分支 (`git push origin feature/amazing-feature`)
5. **打开** Pull Request

### 开发指南

- 遵循 Go 和 TypeScript 最佳实践
- 为新功能添加测试
- 根据需要更新文档
- 确保所有测试在提交前通过

## 路线图

## 支持

- **文档**：[Wiki](https://github.com/burugo/one-mcp/wiki)
- **问题反馈**：[GitHub Issues](https://github.com/burugo/one-mcp/issues)

## 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件。

---

<div align="center">

如果您觉得这个项目有帮助，请 **[⭐ 给项目点星](https://github.com/burugo/one-mcp)**！

由 One MCP 团队用 ❤️ 制作

</div>