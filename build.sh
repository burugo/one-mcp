#!/bin/bash

export PORT=${PORT:-3000}
echo "Using port: $PORT"

echo "Building frontend..."
cd frontend
npm run build
cd ..

echo "Starting backend server on port $PORT..."
go run main.go 