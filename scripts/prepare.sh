#!/bin/bash
set -Eeuo pipefail

COZE_WORKSPACE_PATH="${COZE_WORKSPACE_PATH:-$(pwd)}"
export GOPROXY=https://goproxy.cn,direct

cd "${COZE_WORKSPACE_PATH}"

# Install Go if not present
if ! command -v go &> /dev/null; then
    echo "Installing Go..."
    if [ ! -f /usr/local/go/bin/go ]; then
        curl -fsSL https://dl.google.com/go/go1.22.10.linux-amd64.tar.gz -o /tmp/go.tar.gz
        tar -C /usr/local -xzf /tmp/go.tar.gz
        rm -f /tmp/go.tar.gz
    fi
    export PATH="$PATH:/usr/local/go/bin"
fi

echo "Installing frontend dependencies..."
cd frontend
pnpm install --prefer-frozen-lockfile --prefer-offline --loglevel debug --reporter=append-only

echo "Installing Go dependencies..."
cd "${COZE_WORKSPACE_PATH}/backend"
go mod tidy

echo "Build Go backend..."
CGO_ENABLED=0 go build -o server .

echo "Prepare completed successfully!"
