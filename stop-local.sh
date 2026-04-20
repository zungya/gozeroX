#!/bin/bash
# gozeroX 本地停止脚本
# 使用方法: bash stop-local.sh

GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

echo "停止所有 Go 微服务..."

KILLED=0

# 1. 通过端口停止 API/RPC 服务
for PORT in 1001 1002 1003 1004 1005 2001 2002 2003 2004 2005; do
    PIDS=$(lsof -iTCP:$PORT -sTCP:LISTEN -t 2>/dev/null)
    if [ -n "$PIDS" ]; then
        for PID in $PIDS; do
            NAME=$(ps -p $PID -o comm= 2>/dev/null || echo "unknown")
            echo -e "  ${GREEN}✓${NC} 停止 :$PORT (PID $PID $NAME)"
            kill $PID 2>/dev/null
            KILLED=$((KILLED+1))
        done
    fi
done

# 2. 通过 PID 文件停止 MQ 消费者（无监听端口）
if [ -f .local-pids ]; then
    LINE_NUM=0
    while read PID; do
        LINE_NUM=$((LINE_NUM+1))
        # MQ 进程在第 11、12 行（最后两行）
        if [ $LINE_NUM -ge 11 ]; then
            if kill -0 $PID 2>/dev/null; then
                echo -e "  ${GREEN}✓${NC} 停止 MQ 进程 (PID $PID)"
                kill $PID 2>/dev/null
                KILLED=$((KILLED+1))
            fi
        fi
    done < .local-pids
fi

# 清理 PID 文件
rm -f .local-pids

if [ $KILLED -gt 0 ]; then
    echo ""
    echo -e "${GREEN}已停止 $KILLED 个服务${NC}"
else
    echo -e "${RED}没有发现运行中的服务${NC}"
fi
