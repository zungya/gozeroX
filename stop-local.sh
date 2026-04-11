#!/bin/bash
# gozeroX 本地停止脚本
# 使用方法: bash stop-local.sh

GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

echo "停止所有 Go 微服务..."

# 优先通过端口查找并杀死进程（更可靠）
KILLED=0
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

# 清理 PID 文件
rm -f .local-pids

if [ $KILLED -gt 0 ]; then
    echo ""
    echo -e "${GREEN}已停止 $KILLED 个服务${NC}"
else
    echo -e "${RED}没有发现运行中的服务${NC}"
fi
