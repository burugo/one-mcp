#!/bin/bash

# 设置错误处理
set -e

# 使用环境变量PORT或默认3000端口
export PORT=${PORT:-3000}
echo "Using port: $PORT"

# 先清理已存在的进程
echo "Cleaning up existing processes..."
lsof -ti :${PORT},5173-5176 | xargs kill -9 2>/dev/null || echo "No existing processes found"

# 存储进程ID的数组
declare -a PIDS=()

# 清理函数
cleanup() {
    echo -e "\nShutting down all processes..."
    # 终止所有子进程
    for pid in "${PIDS[@]}"; do
        if ps -p $pid > /dev/null; then
            echo "Killing process $pid"
            kill -TERM $pid 2>/dev/null || kill -9 $pid 2>/dev/null
        fi
    done
    # 确保没有遗留的vite进程
    for port in {5173..5176}; do
        pid=$(lsof -ti :$port 2>/dev/null)
        if [ ! -z "$pid" ]; then
            echo "Killing vite process on port $port (PID: $pid)"
            kill -9 $pid 2>/dev/null || true
        fi
    done
    exit 0
}

# 设置信号处理
trap cleanup INT TERM

# 运行后端
echo "Starting backend server..."
go run main.go &
PIDS+=($!)

# 运行前端
echo "Starting frontend development server..."
cd frontend
npm run dev &
PIDS+=($!)

echo -e "\nDevelopment servers started. Press Ctrl+C to stop all servers.\n"

# 等待所有子进程
wait 