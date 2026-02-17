<p align="right">
    <strong>English</strong> | <a href="./README.zh.md">中文</a>
</p>

# One MCP

<div align="center">

**One MCP** - A centralized proxy for Model Context Protocol (MCP) services

*✨ Manage, monitor, and configure your MCP services from a single interface ✨*

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
  <a href="#features">Features</a> •
  <a href="#quick-start">Quick Start</a> •
  <a href="#installation">Installation</a> •
  <a href="#configuration">Configuration</a> •
  <a href="#development">Development</a> •
  <a href="#contributing">Contributing</a>
</p>

---

## Overview

One MCP is a comprehensive management platform for Model Context Protocol (MCP) services. Acting as a centralized proxy, it lets you discover, install, configure, and monitor MCP services from various providers. Built with Go and React, it offers both powerful backend capabilities and an intuitive web interface.

![Screenshot](./images/dashboard.png)

## Features

### 🚀 **Service Management**
- **Install & Configure**: Deploy MCP services from marketplace or custom sources
- **Multiple Service Types**: Support for stdio, Server-Sent Events (SSE), and streamable HTTP services
- **Environment Management**: Secure handling of service environment variables and configurations
- **Health Monitoring**: Real-time service health checks and status monitoring

### 🔗 **Service Groups (Combo)**
- **Combine Services**: Group multiple MCP services into a single unified endpoint
- **Skill Export**: Export groups as Anthropic Skills for Claude Code and Droid
- **Flexible Configuration**: Custom group naming and service selection
- **SSE/HTTP Support**: Access grouped services via SSE or streamable HTTP protocols

### 🛒 **Service Marketplace**
- **Discover Services**: Browse and search MCP services from various repositories
- **One-Click Installation**: Simple installation process with automatic dependency resolution
- **Custom Services**: Create and deploy custom MCP services with flexible configuration options

### 📊 **Analytics & Monitoring**
- **Usage Statistics**: Track service utilization and performance metrics
- **Request Analytics**: Monitor API requests, response times, and error rates
- **System Health**: Comprehensive system status and uptime monitoring

### 👥 **User Management**
- **Multi-User Support**: Role-based access control with admin and user roles
- **OAuth Integration**: Login with GitHub and Google accounts
- **Secure Authentication**: Token-based authentication with refresh token support

### 🌐 **Internationalization**
- **Multi-Language Support**: English and Chinese (Simplified) interface
- **Localized Content**: Fully translated user interface and error messages
- **Language Persistence**: User language preferences saved across sessions

### ⚙️ **Advanced Configuration**
- **Environment Variables**: Flexible configuration management
- **Database Support**: SQLite (default) with MySQL/PostgreSQL support
- **Redis Integration**: Optional Redis support for distributed caching
- **Docker Ready**: Full Docker support for easy deployment

![Screenshot](./images/services.png)
![Screenshot](./images/copy_config.png)

### Service Groups

Create service groups to combine multiple MCP services and export as Skills:

![Screenshot](./images/group.png)

## Quick Start

### Using Homebrew (macOS & Linux)

```bash
# Add tap
brew tap burugo/tap

# Install one-mcp
brew install one-mcp

# Start as background service (default port: 3000)
brew services start one-mcp

# Stop service
brew services stop one-mcp
```

If port `3000` is already in use, restart with a custom port:

```bash
ONE_MCP_PORT=3001 brew services restart one-mcp
```

Access the application at http://localhost:3000 (or your custom port).

### Using Docker (Recommended)

```bash
# Run with Docker
docker run --name one-mcp -d \
  --restart always \
  -p 3000:3000 \
  -v $(pwd)/data:/data \
  buru2020/one-mcp:latest

# Access the application
open http://localhost:3000
```

### Manual Installation

```bash
# Clone the repository
git clone https://github.com/burugo/one-mcp.git
cd one-mcp

# Set up environment
cp .env_example .env

bash ./run.sh
```

**Default Login**: Username `root`, Password `123456`

## Installation

### Prerequisites

#### Homebrew Installation (macOS & Linux)

- **Homebrew** installed

#### Manual Installation

- **Go**: Version 1.19 or later
- **Node.js**: Version 16 or later
- **Database**: SQLite (default), MySQL, or PostgreSQL
- **Redis**: Optional

### Environment Configuration

Create a `.env` file from the template:

```bash
cp .env_example .env
```

Key configuration options:

```bash
# Server Configuration
PORT=3000

# Database (SQLite is default, MySQL and PostgreSQL are supported)
# SQLite(default)
# SQLITE_PATH=/data/one-mcp.db
# MySQL:
# SQL_DSN=root:password@tcp(localhost:3306)/one_mcp
# PostgreSQL:
# SQL_DSN=postgres://username:password@localhost/database_name?sslmode=disable

# Redis (optional, replace local cache for production environment)
REDIS_CONN_STRING=redis://localhost:6379

# GitHub API (optional, for querying npm's github homepage star count, without this, there will be rate limit issues)
GITHUB_TOKEN=your-github-token
```

### Homebrew Installation (macOS & Linux)

```bash
# Add tap
brew tap burugo/tap

# Install one-mcp
brew install one-mcp

# Run in foreground
one-mcp --port 3000

# Or run as system service (default port: 3000)
brew services start one-mcp

# Use a custom service port when 3000 is occupied
ONE_MCP_PORT=3001 brew services restart one-mcp
```

### Docker Deployment

```bash
# Build the Docker image
docker build -t one-mcp .

# Run with docker-compose (recommended)
docker-compose up -d

# Or run directly
docker run -d \
  --name one-mcp \
  -p 3000:3000 \
  -v ./data:/data \
  -e PORT=3000 \
  one-mcp
```

### Manual Deployment

1. **Build the application**:
   ```bash
   ./deploy/build.sh
   ```

2. **Run the server**:
   ```bash
   ./one-mcp --port 3000
   ```

3. **Access the application**:
   Open http://localhost:3000 in your browser

## Configuration

### OAuth Setup

#### GitHub OAuth
1. Create a GitHub OAuth App at https://github.com/settings/applications/new
2. Set Homepage URL: `http://your-domain.com`
3. Set Authorization callback URL: `http://your-domain.com/oauth/github`
4. Configure in the application preferences

#### Google OAuth
1. Create credentials at https://console.developers.google.com/
2. Set Authorized JavaScript origins: `http://your-domain.com`
3. Set Authorized redirect URIs: `http://your-domain.com/oauth/google`
4. Configure in the application preferences

### Database Configuration

#### SQLite (Default)
No additional configuration required. Database file is created at `./data/one-mcp.db`.

#### MySQL
```bash
SQL_DSN=username:password@tcp(localhost:3306)/database_name
```

#### PostgreSQL
```bash
SQL_DSN=postgres://username:password@localhost/database_name?sslmode=disable
```

## API Documentation

The application provides RESTful APIs for all functionality:

- **Base URL**: `http://localhost:3000/api`
- **Authentication**: Bearer token (obtained via login)
- **Content-Type**: `application/json`

### Key Endpoints

- `POST /api/auth/login` - User authentication
- `GET /api/services` - List installed services
- `POST /api/services` - Install new service
- `GET /api/market/search` - Search marketplace
- `GET /api/analytics/usage` - Usage statistics

## Development

### Development Environment

```bash
# Start development servers
./run.sh

# This will start:
# - Backend server on :3000
# - Frontend dev server on :5173 (with hot reload)
```

### Project Structure

```
one-mcp/
├── backend/         # Go backend code
├── frontend/        # React frontend code  
├── data/           # Database and uploads
├── main.go         # Application entry point
├── build.sh        # Production build script
└── run.sh          # Development script
```

### Testing

```bash
# Frontend tests
cd frontend && npm test

# Backend tests
go test ./...
```

For detailed development instructions, see [DEVELOPMENT.md](./DEVELOPMENT.md).

## Contributing

We welcome contributions! Please see our contributing guidelines:

1. **Fork** the repository
2. **Create** a feature branch (`git checkout -b feature/amazing-feature`)
3. **Commit** your changes (`git commit -m 'Add amazing feature'`)
4. **Push** to the branch (`git push origin feature/amazing-feature`)
5. **Open** a Pull Request

### Development Guidelines

- Follow Go and TypeScript best practices
- Add tests for new functionality
- Update documentation as needed
- Ensure all tests pass before submitting

## Roadmap


## Support

- **Documentation**: [Wiki](https://github.com/burugo/one-mcp/wiki)
- **Issues**: [GitHub Issues](https://github.com/burugo/one-mcp/issues)

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

<div align="center">

**[⭐ Star this project](https://github.com/burugo/one-mcp)** if you find it helpful!

Made with ❤️ by the One MCP team

</div>