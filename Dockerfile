# ===== 构建阶段 =====
FROM golang:1.26.1-alpine AS builder

RUN apk add --no-cache git gcc musl-dev

WORKDIR /app

# 复制整个项目（go.work + 各子模块的 go.mod/go.sum 都在 app/ 和 pkg/ 里）
COPY go.work go.work.sum ./
COPY pkg/ pkg/
COPY app/ app/

# 构建参数：服务入口路径
ARG BUILD_SERVICE

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /app/server ${BUILD_SERVICE}

# ===== 运行阶段 =====
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone

WORKDIR /app

COPY --from=builder /app/server .

CMD ["./server", "-f", "etc/service.yaml"]
