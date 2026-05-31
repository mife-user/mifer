# ============================================================
# Mifer — AI Agent Bot Docker 镜像
# ============================================================
# 多阶段构建：Go 编译 → 最小运行时镜像
# ============================================================

# ---- 构建阶段 ----
FROM golang:1.25-alpine AS builder

# 安装构建依赖（git 用于 go mod download，gcc 用于 CGO）
RUN apk add --no-cache git gcc musl-dev

WORKDIR /build

# 先复制依赖文件，利用 Docker 缓存层
COPY go.mod go.sum ./
RUN go mod download

# 复制源码并编译
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /build/mifer ./cmd/main
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /build/mcp-demo ./cmd/mcp-demo

# ---- 运行阶段 ----
FROM alpine:3.21

# 安装运行时依赖：ca-certificates（HTTPS 请求）、tzdata（时区）
RUN apk add --no-cache ca-certificates tzdata

# 创建非 root 用户
RUN adduser -D -h /app mifer

WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /build/mifer /app/mifer
COPY --from=builder /build/mcp-demo /app/mcp-demo

# 创建数据和配置目录
RUN mkdir -p /app/config /app/memory /app/.mifer/plans /app/.mifer/skills && \
    chown -R mifer:mifer /app

# 切换到非 root 用户
USER mifer

# 暴露 HTTP 服务端口
EXPOSE 15555

# 环境变量默认值
ENV MIFER_ENV=prod

# 启动 HTTP 服务模式
ENTRYPOINT ["./mifer", "serve"]
