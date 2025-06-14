#!/bin/bash
echo "Building frontend..."
cd frontend
npm run build
cd ..

echo "Building backend..."
go build -o one-mcp main.go

echo "Build complete"