# One MCP Development Guide

This document describes how to set up and run the One MCP development environment.

## Project Structure

```
one-mcp/
├── backend/         # Go backend code
│   ├── api/         # API handlers and routes
│   ├── common/      # Common utilities and helpers
│   ├── config/      # Configuration management
│   ├── data/        # Data access layer
│   ├── library/     # Third-party library integrations
│   ├── model/       # Data models and database schemas
│   └── service/     # Business logic services
├── frontend/        # React frontend code
│   ├── src/         # Frontend source code
│   ├── public/      # Static assets and localization files
│   └── dist/        # Built frontend files (generated)
├── data/            # Database and persistent data
├── upload/          # File upload storage
├── locales/         # Internationalization files
├── doc/             # Documentation
├── config/          # Configuration files
├── main.go          # Program entry point
├── build.sh         # Production build script
├── run.sh           # Development environment startup script
└── .env_example     # Environment variables template
```

## Development Environment Setup

### Prerequisites

- Go 1.19 or later
- Node.js 16 or later
- Redis (optional, for rate limiting)

### Install Dependencies

1. Install Go dependencies:
```bash
go mod tidy
```

2. Install frontend dependencies:
```bash
cd frontend
npm install
cd ..
```

### Environment Configuration

1. Copy the environment template:
```bash
cp .env_example .env
```

2. Edit `.env` file with your configuration:
```bash
# Server configuration
PORT=3000

# Database configuration (optional, defaults to SQLite)
# SQL_DSN=root:password@tcp(localhost:3306)/one_mcp

# Redis configuration (optional, for rate limiting)
# REDIS_CONN_STRING=redis://localhost:6379

# Github Token (optional, for calling github api rate limiting)
# GITHUB_TOKEN=your-github-token
```

### Run Development Environment

Use the development script to start both frontend and backend servers simultaneously:

```bash
# Use default port 3000
./run.sh

# Or specify a custom port
PORT=8080 ./run.sh
```

This will:
- Build and start the backend server on http://localhost:$PORT (default 3000)
- Start the frontend development server on http://localhost:5173
- Automatically proxy API requests from frontend to backend
- Set up hot reloading for both frontend and backend changes
- Create necessary database tables on first run

The script will:
- Load environment variables from `.env` file
- Clean up any existing processes on the specified ports
- Build the Go backend and start it in the background
- Start the Vite development server for the frontend
- Handle graceful shutdown when you press Ctrl+C

### Build Production Version

To build the production version:

```bash
# Use default port 3000
./build.sh

# Or specify a custom port
PORT=8080 ./build.sh
```

This will:
1. Build the frontend code and output to `frontend/dist` directory
2. Start the backend server which serves the compiled frontend resources

The server will run on http://localhost:$PORT (default 3000).

## API Access

All API endpoints are prefixed with `/api/` and handled by the backend. In development mode, the Vite development server automatically proxies these requests to the backend server.


### API Authentication

The API uses token-based authentication:
- Login endpoints: `/api/auth/login`, `/api/oauth/github`, `/api/oauth/google`
- Protected endpoints require `Authorization: Bearer <token>` header
- The frontend API utility automatically handles token management

## Database

The application uses SQLite by default, with the database file stored in `./data/one-mcp.db`.

### Database Initialization

On first startup, the application will:
- Create the database file if it doesn't exist
- Run database migrations to create necessary tables
- Create a default root user (username: `root`, password: `123456`)

### Using External Database

You can configure an external database by setting the `SQL_DSN` environment variable:

```bash
# MySQL example
SQL_DSN=root:password@tcp(localhost:3306)/one_mcp

# PostgreSQL example  
SQL_DSN=postgres://user:password@localhost/one_mcp?sslmode=disable
```

## Port Configuration

The project synchronizes frontend and backend port configuration:

- Default port is `3000`
- Can be changed via the `PORT` environment variable
- In development mode:
  - Frontend dev server runs on port `5173`
  - Backend runs on the specified `PORT`
  - API requests are automatically proxied from frontend to backend
- In production mode:
  - Frontend and backend share the same port
  - Frontend assets are embedded in the backend binary

## Development Workflow

### Making Changes

1. **Backend changes**: The `run.sh` script will automatically rebuild and restart the backend when you make changes
2. **Frontend changes**: Vite provides hot module replacement for instant updates
3. **Database changes**: Modify models in `backend/model/` and restart the application

### Testing

Run tests for different components:

```bash
# Frontend tests
cd frontend
npm test

# Backend tests (if available)
go test ./...
```

### Logs and Debugging

- Backend logs are written to `backend.log`
- View real-time logs: `tail -f backend.log`
- Filter error logs: `grep "ERROR\|WARN\|Failed" backend.log`

## Internationalization (i18n)

The frontend supports multiple languages using react-i18next:

- Translation files are located in `frontend/public/locales/{language}/translation.json`
- Supported languages: English (`en`) and Chinese Simplified (`zh-CN`)
- Language preference is stored in localStorage
- Backend API responses include localized content based on `Accept-Language` header

## Notes

- The build process uses Go's embed functionality to embed frontend files into the backend binary
- Files in the `frontend/dist` directory are recreated on each build
- API requests are automatically proxied to the backend server, no manual CORS configuration needed
- The application supports OAuth authentication with GitHub and Google (requires configuration)
- Redis can be used for distributed rate limiting in production environments 