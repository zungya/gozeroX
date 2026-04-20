#!/bin/bash
# Kafka Topic 预创建脚本
# 由 Kafka 容器启动时自动调用

TOPICS=(
    "comment_create"
    "like_tweet"
    "like_comment"
    "notice"
    "recommend_tweet"
    "recommend_interaction"
    "comment_status_sync"
)

echo "========== 创建 Kafka Topics =========="

# 等待 Kafka broker 就绪
for i in $(seq 1 60); do
    if /opt/kafka/bin/kafka-topics.sh --bootstrap-server kafka:9092 --list > /dev/null 2>&1; then
        break
    fi
    sleep 1
done

# 创建 topics
for topic in "${TOPICS[@]}"; do
    /opt/kafka/bin/kafka-topics.sh --bootstrap-server kafka:9092 \
        --create --if-not-exists \
        --topic "$topic" \
        --partitions 3 \
        --replication-factor 1 2>/dev/null
done

echo "========== Kafka Topics 创建完成 =========="
