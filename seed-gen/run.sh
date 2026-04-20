#!/bin/bash
# seed-gen 主控脚本
# 一键生成全部测试数据：20用户 + 300推文 + 600点赞 + 200评论
#
# 使用方法:
#   bash seed-gen/run.sh
#
# 前提: 所有服务已启动 (bash start-local.sh)

set -e

GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m'

SEED_DIR="$(cd "$(dirname "$0")" && pwd)"

echo -e "${YELLOW}========== gozeroX 测试数据生成 ==========${NC}"
echo ""

# 1. 注册用户
bash "$SEED_DIR/01-register.sh"
echo ""

# 2. 创建推文（9大类约300条）
bash "$SEED_DIR/02-tweets.sh"
echo ""

echo -e "${YELLOW}等待 MQ 消费推文数据 (2秒)...${NC}"
sleep 2

# 3. 点赞（600条，正态分布）
bash "$SEED_DIR/03-likes.sh"
echo ""

echo -e "${YELLOW}等待 MQ 消费点赞数据 (2秒)...${NC}"
sleep 2

# 4. 评论（200条，集中在部分推文）
bash "$SEED_DIR/04-comments.sh"
echo ""

echo -e "${YELLOW}等待 MQ 消费评论数据 (3秒)...${NC}"
sleep 3

# 汇总
TOKEN_COUNT=$(grep -c '.' /tmp/seed-tokens.txt 2>/dev/null || echo 0)
TWEET_COUNT=$(wc -l < /tmp/seed-tweets.txt 2>/dev/null || echo 0)
COMMENT_COUNT=$(wc -l < /tmp/seed-comments.txt 2>/dev/null || echo 0)

echo ""
echo "============================================"
echo -e "${GREEN}测试数据生成完成！${NC}"
echo "============================================"
echo "  用户:   $TOKEN_COUNT 个"
echo "  推文:   $TWEET_COUNT 条"
echo "  点赞:   ~600 次"
echo "  评论:   $COMMENT_COUNT 条"
echo ""
echo "测试账号: mobile=13800000101 ~ 13800000120, password=test123456"
