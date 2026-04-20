#!/bin/bash
# gozeroX 全量清理脚本
# 清理日志、Prometheus、Grafana、Kafka、Redis、PostgreSQL 数据
# 使用方法: bash clean-all.sh

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m'

echo -e "${YELLOW}⚠  此脚本将清理以下数据：${NC}"
echo "  - 日志文件 (logs/)"
echo "  - Prometheus 数据 (data/prometheus/)"
echo "  - Grafana 数据 (data/grafana/)"
echo "  - Kafka 数据 (data/kafka/)"
echo "  - Redis 数据 (data/redis/)"
echo "  - PostgreSQL 数据 (data/postgresql/)"
echo ""
read -p "确认清理？(y/N): " CONFIRM
if [ "$CONFIRM" != "y" ] && [ "$CONFIRM" != "Y" ]; then
    echo "已取消"
    exit 0
fi

# 1. 先停止所有 Go 服务
echo ""
echo -e "${GREEN}[1/3]${NC} 停止 Go 微服务..."
bash stop-local.sh 2>/dev/null || true

# 2. 停止 Docker 基础设施
echo -e "${GREEN}[2/3]${NC} 停止 Docker 基础设施..."
docker compose -f docker-compose-env.yml down 2>/dev/null || true

# 3. 清理数据
echo -e "${GREEN}[3/3]${NC} 清理数据..."

# 日志
rm -rf logs/
echo "  ✓ 日志已清理"

# Prometheus
rm -rf data/prometheus/data/*
echo "  ✓ Prometheus 数据已清理"

# Grafana
rm -rf data/grafana/data/*
echo "  ✓ Grafana 数据已清理"

# Kafka
rm -rf data/kafka/data/*
echo "  ✓ Kafka 数据已清理"

# Redis
rm -rf data/redis/data/*
echo "  ✓ Redis 数据已清理"

# PostgreSQL（需要删除整个数据目录，重启时 init 脚本会重建）
rm -rf data/postgresql/data/*
echo "  ✓ PostgreSQL 数据已清理"

echo ""
echo -e "${GREEN}全部清理完成！${NC}"
echo "重新启动: docker compose -f docker-compose-env.yml up -d && bash start-local.sh"
