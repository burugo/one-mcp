#!/bin/bash

# 使用环境变量PORT或默认3000端口
export PORT=${PORT:-3000}
echo "Using port: $PORT"

echo "Building frontend..."
cd frontend
npm run build
cd ..

echo "Starting backend server on port $PORT..."
go run main.go 