#!/bin/bash
# 注册 20 个用户
source "$(dirname "$0")/common.sh"

check_services

echo -e "${GREEN}[1/4]${NC} 注册用户..."

> "$TOKENS_FILE"

for i in $(seq 1 20); do
    mobile=$(printf "138%08d" "$((i + 100))")
    result=$(register_user "$mobile" "test123456")
    token=$(echo "$result" | cut -d'|' -f1)
    uid=$(echo "$result" | cut -d'|' -f2)
    if [ -z "$token" ]; then
        echo "  用户$i 注册失败"
        echo "" >> "$TOKENS_FILE"
        continue
    fi
    echo "$token" >> "$TOKENS_FILE"
    echo "  用户$i uid=$uid"
done

echo "  注册完成"
