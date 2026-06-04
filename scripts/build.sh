#!/bin/bash
set -Eeuo pipefail

COZE_WORKSPACE_PATH="${COZE_WORKSPACE_PATH:-$(pwd)}"
export GOPROXY=https://goproxy.cn,direct

# Install Go if not present
if ! command -v go &> /dev/null; then
    if [ ! -f /usr/local/go/bin/go ]; then
        echo "Installing Go..."
        curl -fsSL https://dl.google.com/go/go1.22.10.linux-amd64.tar.gz -o /tmp/go.tar.gz
        tar -C /usr/local -xzf /tmp/go.tar.gz
        rm -f /tmp/go.tar.gz
    fi
    export PATH="$PATH:/usr/local/go/bin"
fi

cd "${COZE_WORKSPACE_PATH}"

# Build Vue frontend
echo "Building Vue frontend..."
cd frontend
pnpm install --prefer-frozen-lockfile --prefer-offline
pnpm run build

# Copy built assets to backend's static directory
echo "Copying frontend build to backend/static..."
rm -rf "${COZE_WORKSPACE_PATH}/backend/static"
cp -r dist "${COZE_WORKSPACE_PATH}/backend/static"

# Build Go backend
echo "Building Go backend..."
cd "${COZE_WORKSPACE_PATH}/backend"
CGO_ENABLED=0 go build -o server .

echo "Build completed successfully!"
