#!/bin/bash

# 设置错误处理
set -e

# 先清理已存在的 Vite 进程
echo "Cleaning up existing Vite processes..."
lsof -ti tcp:5173-5176 | xargs kill -9 2>/dev/null || echo "No existing Vite processes found on ports 5173-5176"

# 存储前端进程ID
FRONTEND_PID=""

# 清理函数
cleanup() {
    echo -e "\nShutting down frontend development server..."
    if [ ! -z "$FRONTEND_PID" ] && ps -p $FRONTEND_PID > /dev/null; then
        echo "Killing frontend process $FRONTEND_PID"
        kill -TERM $FRONTEND_PID 2>/dev/null || kill -9 $FRONTEND_PID 2>/dev/null
    fi
    # 确保没有遗留的vite进程 (再次检查以防万一)
    for port in {5173..5176}; do
        pid=$(lsof -ti :$port 2>/dev/null)
        if [ ! -z "$pid" ]; then
            echo "Killing lingering Vite process on port $port (PID: $pid)"
            kill -9 $pid 2>/dev/null || true
        fi
    done
    # 清理复制的 .env 文件
    if [ -f "frontend/.env" ]; then
        echo "Removing copied .env file from frontend directory..."
        rm -f "frontend/.env"
    fi
    exit 0
}

# 设置信号处理
trap cleanup INT TERM

# 运行前端
echo "Starting frontend development server..."
cd frontend

# 复制根目录的 .env 文件到 frontend 目录 (如果存在)
if [ -f "../.env" ]; then
    echo "Copying .env file to frontend directory..."
    cp ../.env .
fi

npm run dev &
FRONTEND_PID=$!

echo -e "\nFrontend development server started (PID: $FRONTEND_PID). Press Ctrl+C to stop."

# 等待前端进程
wait "$FRONTEND_PID" 