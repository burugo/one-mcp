#!/bin/bash

# Set error handling
set -e
echo "Building frontend..."
cd frontend && npm install && npm run build && cd ..

# Load .env environment variables
if [ -f .env ]; then
  export $(cat .env | grep -v '^#' | grep -v '^$' | xargs)
fi

# Ensure PATH includes /usr/local/bin
export PATH=$PATH:/usr/local/bin

# Set port number from environment variable, default to 3000
PORT=${PORT:-3000}

# Clean up existing processes
echo "Cleaning up existing processes..."
# Kill processes LISTENING on backend port (not clients connecting to remote port 3000)
lsof -ti TCP:$PORT -sTCP:LISTEN | xargs kill -9 2>/dev/null || echo "No existing backend processes found on port $PORT"
# Kill processes LISTENING on frontend Vite port
lsof -ti TCP:5173 -sTCP:LISTEN | xargs kill -9 2>/dev/null || echo "No existing Vite processes found on port 5173"

# Store process IDs
BACKEND_PID=""
# Cleanup function
cleanup() {
    echo -e "\nShutting down development servers..."
    
    # Clean up backend process
    if [ ! -z "$BACKEND_PID" ] && ps -p $BACKEND_PID > /dev/null; then
        echo "Killing backend process $BACKEND_PID"
        kill -TERM $BACKEND_PID 2>/dev/null || kill -9 $BACKEND_PID 2>/dev/null
    fi
    
    # Clean up frontend process
    if [ ! -z "$FRONTEND_PID" ] && ps -p $FRONTEND_PID > /dev/null; then
        echo "Killing frontend process $FRONTEND_PID"
        kill -TERM $FRONTEND_PID 2>/dev/null || kill -9 $FRONTEND_PID 2>/dev/null
    fi
    
    # Ensure no lingering processes
    # Backend port (only LISTEN, not clients)
    pid=$(lsof -ti TCP:$PORT -sTCP:LISTEN 2>/dev/null)
    if [ ! -z "$pid" ]; then
        echo "Killing lingering backend process on port $PORT (PID: $pid)"
        kill -9 $pid 2>/dev/null || true
    fi
    
    # Frontend port (only LISTEN, not clients)
    pid=$(lsof -ti TCP:5173 -sTCP:LISTEN 2>/dev/null)
    if [ ! -z "$pid" ]; then
        echo "Killing lingering Vite process on port 5173 (PID: $pid)"
        kill -9 $pid 2>/dev/null || true
    fi
    
    # Clean up copied .env file
    if [ -f "frontend/.env" ]; then
        echo "Removing copied .env file from frontend directory..."
        rm -f "frontend/.env"
    fi
    
    exit 0
}

# Set signal handling
trap cleanup INT TERM
# Start frontend development server
echo "Starting frontend development server..."
cd frontend

# Copy .env file from root directory to frontend directory (if exists)
if [ -f "../.env" ]; then
    echo "Copying .env file to frontend directory..."
    cp ../.env .
fi

npm run dev &
FRONTEND_PID=$!

cd ..
# Build backend service
echo "Building backend service..."
go build -o one-mcp .

# Start backend service
echo "Starting backend service..."
nohup ./one-mcp > backend.log 2>&1 &
BACKEND_PID=$!
echo "Backend started on :$PORT (PID: $BACKEND_PID), logs in backend.log"




echo -e "\nDevelopment servers started:"
echo "- Backend: :$PORT (PID: $BACKEND_PID)"
echo "- Frontend: :5173 (PID: $FRONTEND_PID)"
echo "Open http://localhost:5173/ to access the frontend."
echo "Press Ctrl+C to stop all servers."

# Wait for all processes
wait 