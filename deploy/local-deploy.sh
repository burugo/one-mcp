#!/bin/bash
set -e

echo "Building One MCP Docker image..."
docker build -t one-mcp:$(cat VERSION) .
docker tag one-mcp:$(cat VERSION) one-mcp:latest

echo "Stopping existing container..."
docker stop one-mcp-prod 2>/dev/null || true
docker rm one-mcp-prod 2>/dev/null || true

echo "Starting new container..."
docker run -d \
  --name one-mcp-prod \
  --restart unless-stopped \
  -p 3000:3000 \
  -v $(pwd)/data:/data \
  -e SQLITE_PATH=/data/one-mcp.db \
  one-mcp:latest

echo "Deployment completed!"
echo "Application is running at http://localhost:3000"