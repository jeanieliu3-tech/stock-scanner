# ---- Stage 1: Build frontend ----
FROM node:20-alpine AS frontend-builder
WORKDIR /build/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm install --legacy-peer-deps
COPY frontend/ ./
RUN npx vite build

# ---- Stage 2: Build backend ----
FROM golang:1.22-alpine AS backend-builder
RUN apk add --no-cache git
WORKDIR /build/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
RUN go mod verify
COPY backend/ ./
# 前端构建产物复制到 ./static 供磁盘 serving
COPY --from=frontend-builder /build/frontend/dist ./static
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -o backend_linux . 2>&1

# ---- Stage 3: Runtime ----
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Shanghai
WORKDIR /app
COPY --from=backend-builder /build/backend/backend_linux .
COPY --from=frontend-builder /build/frontend/dist ./static
RUN chmod +x backend_linux
EXPOSE 10000
CMD ["./backend_linux"]
