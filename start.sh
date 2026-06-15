#!/bin/bash
set -euo pipefail

is_docker_running() {
    docker info > /dev/null 2>&1
}

port_in_use() {
    local port=$1
    if command -v netstat > /dev/null 2>&1; then
        netstat -an 2>/dev/null | grep -q "[.:]${port} .*LISTEN"
        return $?
    fi
    return 1
}

echo "🔍 Checking if Docker is running..."

if ! is_docker_running; then
    echo "❌ Docker is not running. Attempting to start it..."
    if [[ "$OSTYPE" == "msys" || "$OSTYPE" == "cygwin" ]]; then
        powershell.exe -Command "Start-Process 'C:\Program Files\Docker\Docker\Docker Desktop.exe'" || true
    elif [[ "$OSTYPE" == "darwin"* ]]; then
        open -a Docker || true
    fi
    counter=0
    until is_docker_running || [ $counter -eq 24 ]; do
        sleep 5
        counter=$((counter + 1))
        echo "Still waiting... ($((counter * 5))s)"
    done
    if ! is_docker_running; then
        echo "🚨 Failed to start Docker. Please start it manually."
        exit 1
    fi
fi

echo "✅ Docker is running."

PROFILE="default"
if port_in_use 6379; then
    echo "⚠️  Port 6379 is already in use (existing Redis detected)."
    echo "   Using external profile — proxy will connect to host Redis on :6379"
    PROFILE="external"
fi

echo "🚀 Starting with profile: $PROFILE"
if [ "$PROFILE" = "external" ]; then
    docker compose --profile external up -d --build upstash-local-external
else
    docker compose up -d --build
fi

if [ $? -ne 0 ]; then
    echo "🚨 docker compose failed. Check errors above."
    exit 1
fi

echo ""
echo "✅ Upstash Redis Local is running!"
echo "🔗 REST API:  http://localhost:8000/PING?_token=local-dev-token"
echo "🔗 Dashboard: http://localhost:8000/dashboard"
echo "🔗 Health:    http://localhost:8000/health"
echo "🔗 Redis:     localhost:6379"
echo ""
echo "💡 Unlimited local requests — no cloud rate limits."
