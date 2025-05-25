#!/bin/bash

# 加载 .env 环境变量
if [ -f .env ]; then
  export $(cat .env | xargs)
fi
# 确保 PATH 包含 /usr/local/bin
export PATH=$PATH:/usr/local/bin
# 杀掉占用 3000 端口的进程
lsof -ti:3003 | xargs kill -9 2>/dev/null

# 启动后端服务
nohup go run main.go > backend.log 2>&1 &
echo "Backend started on :3003, logs in backend.log" 