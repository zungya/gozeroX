#!/bin/bash
# gozeroX 本地启动脚本
# 使用方法: bash start-local.sh
# 前提: docker compose -f docker-compose-env.yml up -d 已启动

set -e

GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

# 检查基础设施
if ! docker ps --format '{{.Names}}' | grep -q redis; then
    echo -e "${RED}错误: Docker 基础设施未启动${NC}"
    echo "请先运行: docker compose -f docker-compose-env.yml up -d"
    exit 1
fi

echo "启动所有 Go 微服务..."

# RPC 服务（必须用 -local.yaml，连接 localhost:36379 Redis）
echo -e "${GREEN}[RPC]${NC} 启动 usercenter-rpc (:2001)"
go run app/usercenter/cmd/rpc/usercenter.go -f app/usercenter/cmd/rpc/etc/usercenter-local.yaml &
PID_RPC_USER=$!

echo -e "${GREEN}[RPC]${NC} 启动 contentService-rpc (:2002)"
go run app/contentService/cmd/rpc/contentservice.go -f app/contentService/cmd/rpc/etc/contentservice-local.yaml &
PID_RPC_CONTENT=$!

echo -e "${GREEN}[RPC]${NC} 启动 interactService-rpc (:2003)"
go run app/interactService/cmd/rpc/interactservice.go -f app/interactService/cmd/rpc/etc/interactservice-local.yaml &
PID_RPC_INTERACT=$!

echo -e "${GREEN}[RPC]${NC} 启动 noticeService-rpc (:2004)"
go run app/noticeService/cmd/rpc/notice.go -f app/noticeService/cmd/rpc/etc/noticeservice-local.yaml &
PID_RPC_NOTICE=$!

echo -e "${GREEN}[RPC]${NC} 启动 recommendService-rpc (:2005)"
go run app/recommendService/cmd/rpc/recommendservice.go -f app/recommendService/cmd/rpc/etc/recommendservice-local.yaml &
PID_RPC_RECOMMEND=$!

# 等待 RPC 服务启动
echo "等待 RPC 服务就绪..."
sleep 8

# API 服务（用默认 yaml，只连 RPC）
echo -e "${GREEN}[API]${NC} 启动 usercenter-api (:1001)"
go run app/usercenter/cmd/api/usercenter.go -f app/usercenter/cmd/api/etc/usercenter.yaml &
PID_API_USER=$!

echo -e "${GREEN}[API]${NC} 启动 contentService-api (:1002)"
go run app/contentService/cmd/api/content.go -f app/contentService/cmd/api/etc/content-api.yaml &
PID_API_CONTENT=$!

echo -e "${GREEN}[API]${NC} 启动 interactService-api (:1003)"
go run app/interactService/cmd/api/interaction.go -f app/interactService/cmd/api/etc/interaction-api.yaml &
PID_API_INTERACT=$!

echo -e "${GREEN}[API]${NC} 启动 noticeService-api (:1004)"
go run app/noticeService/cmd/api/notice.go -f app/noticeService/cmd/api/etc/notice-api.yaml &
PID_API_NOTICE=$!

echo -e "${GREEN}[API]${NC} 启动 recommendService-api (:1005)"
go run app/recommendService/cmd/api/recommend.go -f app/recommendService/cmd/api/etc/recommend-api.yaml &
PID_API_RECOMMEND=$!

# 等待 API 服务启动
echo "等待 API 服务就绪..."
sleep 6

# 验证
echo ""
echo "============================================"
echo "服务状态检查"
echo "============================================"
FAIL=0
for PORT in 2001 2002 2003 2004 2005 1001 1002 1003 1004 1005; do
    if lsof -iTCP:$PORT -sTCP:LISTEN -P &>/dev/null; then
        NAME=$(lsof -iTCP:$PORT -sTCP:LISTEN -P 2>/dev/null | tail -1 | awk '{print $1}')
        echo -e "  ${GREEN}✓${NC} $NAME :$PORT"
    else
        echo -e "  ${RED}✗${NC} 端口 :$PORT 未监听"
        FAIL=$((FAIL+1))
    fi
done

if [ $FAIL -eq 0 ]; then
    echo ""
    echo -e "${GREEN}全部 10 个服务启动成功！${NC}"
    echo "运行测试: bash test.sh"
else
    echo ""
    echo -e "${RED}$FAIL 个服务启动失败，请检查日志${NC}"
fi

# 保存 PID 到文件，方便 stop-local.sh 使用
cat > .local-pids <<EOF
$PID_RPC_USER
$PID_RPC_CONTENT
$PID_RPC_INTERACT
$PID_RPC_NOTICE
$PID_RPC_RECOMMEND
$PID_API_USER
$PID_API_CONTENT
$PID_API_INTERACT
$PID_API_NOTICE
$PID_API_RECOMMEND
EOF

echo "PID 已保存到 .local-pids（可用 bash stop-local.sh 停止）"
