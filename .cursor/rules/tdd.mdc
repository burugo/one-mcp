---
description: 
globs: *.tsx,*.go,*.ts
alwaysApply: false
---
## Test-Driven Development (TDD)
**Final Verification:**
Always use: `go test -v ./backend/api/handler/... | grep FAIL`

**Debugging & Handling Verbose Output:**
**Avoid** `go test -v ./...` directly in the terminal due to excessive output.
**Recommended Alternatives:**
*   **Specific Tests:** `go test -v ./... -run ^TestSpecificFunction$` (Fastest for pinpointing).

## Project Startup and Logging Rules

**Service Startup:**
- Use `bash ./run.sh` to start the one-mcp service
- This script automatically loads .env, ensures PATH, kills port 3000 processes, and starts the Go backend in background
- Logs are output to `backend.log`

**Log Monitoring:**
- Use `tail -f backend.log` to monitor real-time logs
- Use `tail -n 50 backend.log` to view recent log entries
- Use `grep "ERROR\|WARN\|Failed" backend.log` to filter error messages
**Service Management:**
- Use `pkill -f one-mcp` to stop the service
- Check service status with `ps aux | grep one-mcp | grep -v grep`
- API status endpoint: `curl "http://localhost:3003/api/status"`

## 数据库重置和缓存管理

**数据库重置规则:**
- **重要：** 删除数据库前必须先备份：`cp data/one-mcp.db data/one-mcp.db.backup.$(date +%Y%m%d_%H%M%S)`
- 仅删除数据库文件 `rm -f data/one-mcp.db` 是不够的
- **必须同时清理Redis缓存：** `redis-cli flushdb` 或 `redis-cli flushall`
- 原因：用户列表查询有缓存机制，如果缓存中存在用户数据，系统不会创建新的root用户

**完整的数据库重置流程:**
```bash
# 1. 停止服务
pkill -f one-mcp

# 2. 备份现有数据库
cp data/one-mcp.db data/one-mcp.db.backup.$(date +%Y%m%d_%H%M%S)

# 3. 删除数据库文件
rm -f data/one-mcp.db

# 4. 清理Redis缓存
redis-cli flushdb

# 5. 重新启动服务
bash ./run.sh
```

**验证root用户创建:**
- 查看日志：`grep "no user exists\|create a root user" backend.log`
- 检查数据库：`sqlite3 data/one-mcp.db "SELECT id, username, LENGTH(token) as token_length FROM users;"`

## 前端API调用规范

**API路径构建:**
- 当使用 `frontend/src/utils/api.ts` 中导出的 `api` 实例进行API调用时 (例如 `api.get`, `api.post` 等)，**请勿在请求路径中再次添加 `/api` 前缀**。
- 这是因为 `api` 实例的 `baseURL` 已经被配置为 `/api`，重复添加会导致 `/api/api/` 这样的错误路径。
- **正确示例:** `api.get('/user/self')` 或 `api.post('/user', data)`。
- **错误示例:** `api.get('/api/user/self')`。

# 项目目录结构

## 根目录 (`/`)
- `backend.log` - 后端服务的运行日志文件。包含服务启动、运行过程中的各种信息，用于调试和监控。
- `data/` - 数据存储目录。通常包含数据库文件、缓存数据或其它持久化数据。
- `one-mcp` - 主应用程序可执行文件（Go语言编译产物）。
- `.git/` - Git版本控制系统目录。包含所有版本历史和配置信息。
- `.cursor/` - Cursor IDE的配置文件和工作区特定文件，包括Agent的规则和任务文件。
  - `rules/` - 存放Agent的规则文件，如 `tdd.mdc` 和 `plan-mode.mdc`。
    - `tdd.mdc` - TDD（测试驱动开发）相关规则和测试指南。
    - `plan-mode.mdc` - Agent的计划模式（PLAN Mode）操作指南。
    - `act-mode.mdc` - Agent的执行模式（ACT Mode）操作指南。
  - (其他可能的Cursor配置或任务文件)
- `main.go` - Go语言后端服务的主入口文件。
- `go.sum` - Go模块依赖校验和文件，用于确保依赖的完整性和安全性。
- `go.mod` - Go模块定义文件，声明项目依赖的外部模块。
- `.gitignore` - Git忽略文件配置，指定不应被版本控制的文件和目录。
- `.cursorignore` - Cursor IDE忽略文件配置，指定在IDE中不应被索引或处理的文件和目录。
- `frontend/` - 前端应用程序的根目录。
  - `src/` - 前端源代码目录。 (内部文件和子目录待进一步探索和注释，但为了简洁，此处暂不列出)
  - `package-lock.json` - npm或yarn的锁定文件，记录了项目依赖的精确版本信息。
  - `package.json` - 前端项目的元数据文件，包含项目信息、脚本命令和依赖列表。
  - `node_modules/` - Node.js模块安装目录，存放前端项目的所有依赖包。
  - `dist/` - 前端项目构建后的输出目录，包含用于部署的静态文件（HTML, CSS, JS等）。
  - `tsconfig.tsbuildinfo` - TypeScript构建信息文件，用于增量编译优化。
  - `vite.config.ts` - Vite前端构建工具的配置文件（TypeScript版本）。
  - `vite.config.d.ts` - Vite配置文件的类型声明文件。
  - `tsconfig.json` - TypeScript编译器的配置文件，定义了如何编译TypeScript代码。
  - `tailwind.config.js` - Tailwind CSS框架的配置文件，用于自定义CSS样式。
  - `components.json` - 可能用于UI组件库的配置文件，定义组件路径或配置。
  - `tsconfig.node.json` - 针对Node.js环境的TypeScript配置文件。
  - `tsconfig.app.json` - 针对前端应用本身的TypeScript配置文件。
  - `postcss.config.js` - PostCSS工具的配置文件，用于处理CSS。
  - `index.html` - 前端应用的HTML入口文件。
  - `public/` - 静态资源目录，其中文件不会被Webpack等打包工具处理，直接复制到`dist`。
  - `.gitignore` - 前端项目的Git忽略文件配置。
  - `README.md` - 前端项目的说明文档。
  - `eslint.config.js` - ESLint代码风格检查工具的配置文件。
- `run.sh` - 服务启动脚本。
- `dev.sh` - 开发环境启动脚本。
- `.vscode/` - VS Code编辑器的工作区配置目录。
- `DEVELOPMENT.md` - 开发指南或说明文档。
- `build.sh` - 项目构建脚本。
- `config/` - 项目的配置目录，可能包含通用配置文件。
- `locales/` - 国际化（i18n）文件目录，存放多语言资源。
- `backend/` - 后端服务的根目录。
  - `data/` - 后端数据模型或持久化层相关代码。
  - `model/` - 后端数据模型定义。
  - `common/` - 后端通用工具函数、常量或共享代码。
  - `library/` - 后端第三方库的封装或自定义库。
  - `service/` - 后端业务逻辑服务层代码。
  - `api/` - 后端API接口定义和处理逻辑。
  - `config/` - 后端配置相关代码或文件。
- `.tool-versions` - asdf版本管理工具的配置文件，指定项目所需的工具版本。
- `doc/` - 项目文档目录。
- `upload/` - 文件上传存储目录。
- `Dockerfile` - Docker容器的构建文件，定义了如何构建应用程序的Docker镜像。
- `LICENSE` - 项目的开源许可证文件。
- `README.en.md` - 英文版的项目说明文档。
- `README.md` - 项目的中文说明文档。
- `VERSION` - 版本号文件。

