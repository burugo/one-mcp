<p align="right">
    <strong>English</strong> | <a href="./README.zh.md">中文</a>
</p>

# One MCP

<div align="center">

**One MCP** - A centralized proxy for Model Context Protocol (MCP) services

*✨ Manage, monitor, and configure your MCP services from a single interface ✨*

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
- **Redis Integration**: Optional Redis support for distributed caching and rate limiting
- **Docker Ready**: Full Docker support for easy deployment

![Screenshot](./images/services.png)
![Screenshot](./images/copy_config.png)

## Quick Start

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

- **Go**: Version 1.19 or later
- **Node.js**: Version 16 or later  
- **Database**: SQLite (default), MySQL, or PostgreSQL
- **Redis**: Optional, for distributed caching

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

# Redis (optional, for rate limiting)
REDIS_CONN_STRING=redis://localhost:6379

# GitHub API (optional, for rate limiting)
GITHUB_TOKEN=your-github-token
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