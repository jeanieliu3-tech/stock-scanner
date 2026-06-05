# ---- Stage 1: Build frontend ----
FROM node:20-alpine AS frontend-builder
WORKDIR /build/frontend
COPY frontend/package.json frontend/package-lock.json* ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# ---- Stage 2: Build backend ----
FROM golang:1.22-alpine AS backend-builder
WORKDIR /build/backend
COPY backend/go.mod backend/go.sum* ./
RUN go mod download
COPY backend/ ./
# 前端构建产物在 ../frontend/dist，复制到 ./static 供 embed 使用
COPY --from=frontend-builder /build/frontend/dist ./static
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o backend_linux .

# ---- Stage 3: Runtime (with static files) ----
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Shanghai
WORKDIR /app
COPY --from=backend-builder /build/backend/backend_linux .
# Copy frontend static files for disk serving
COPY --from=frontend-builder /build/frontend/dist ./static
RUN chmod +x backend_linux
EXPOSE 10000
CMD ["./backend_linux"]
