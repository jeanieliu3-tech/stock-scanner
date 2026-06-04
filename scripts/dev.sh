#!/bin/bash
set -Eeuo pipefail

PORT=5000
COZE_WORKSPACE_PATH="${COZE_WORKSPACE_PATH:-$(pwd)}"
DEPLOY_RUN_PORT="${DEPLOY_RUN_PORT:-${PORT}}"
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

kill_port_if_listening() {
    local pids
    pids=$(ss -H -lntp 2>/dev/null | awk -v port="$1" '$4 ~ ":"port"$"' | grep -o 'pid=[0-9]*' | cut -d= -f2 | paste -sd' ' - || true)
    if [[ -z "${pids}" ]]; then
      echo "Port $1 is free."
      return
    fi
    echo "Port $1 in use by PIDs: ${pids} (SIGKILL)"
    echo "${pids}" | xargs -I {} kill -9 {}
    sleep 1
}

echo "Clearing ports before start."
kill_port_if_listening 3000
kill_port_if_listening "${DEPLOY_RUN_PORT}"

# Start Go backend on port 3000
echo "Starting Go backend on port 3000..."
cd "${COZE_WORKSPACE_PATH}/backend"
BACKEND_PORT=3000 nohup ./server > /app/work/logs/bypass/go-backend.log 2>&1 &
echo "Go backend started (PID: $!)"

# Start Vite dev server on DEPLOY_RUN_PORT
echo "Starting Vite dev server on port ${DEPLOY_RUN_PORT}..."
cd "${COZE_WORKSPACE_PATH}/frontend"
nohup pnpm run dev --port "${DEPLOY_RUN_PORT}" --host 0.0.0.0 > /app/work/logs/bypass/vite-frontend.log 2>&1 &
echo "Vite dev server started (PID: $!)"

echo "Waiting for services to be ready..."
for i in $(seq 1 30); do
    if curl -s --max-time 2 "http://localhost:${DEPLOY_RUN_PORT}" > /dev/null 2>&1; then
        echo "Frontend ready on port ${DEPLOY_RUN_PORT}"
        break
    fi
    sleep 1
done

echo "Dev environment started!"
echo "  Frontend: http://localhost:${DEPLOY_RUN_PORT}"
echo "  Backend:  http://localhost:3000"
