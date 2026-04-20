#!/bin/bash
# seed-gen 公共模块：API 调用函数和全局配置
# 被 run.sh 和各子脚本 source

BASE_USER="http://localhost:1001/usercenter/v1"
BASE_CONTENT="http://localhost:1002/contentService/v1"
BASE_INTERACT="http://localhost:1003/interactService/v1"

TOKENS_FILE="/tmp/seed-tokens.txt"
TWEETS_FILE="/tmp/seed-tweets.txt"
COMMENTS_FILE="/tmp/seed-comments.txt"

GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m'

# ========== 工具函数 ==========

register_user() {
    local mobile=$1
    local password=$2
    local resp=$(curl -s -X POST "$BASE_USER/user/register" \
        -H "Content-Type: application/json" \
        -d "{\"mobile\":\"$mobile\",\"password\":\"$password\"}")
    local token=$(echo "$resp" | python3 -c "import sys,json; print(json.load(sys.stdin).get('accessToken',''))" 2>/dev/null)
    local uid=$(echo "$resp" | python3 -c "import sys,json; print(json.load(sys.stdin).get('userInfo',{}).get('uid',''))" 2>/dev/null)
    echo "${token}|${uid}"
}

create_tweet() {
    local token=$1
    local content=$2
    local tags=$3
    [ -z "$tags" ] && tags="[]"
    # 用 python3 安全构建 JSON，避免 content 中的引号/特殊字符破坏 JSON 结构
    local json_data=$(python3 -c "
import json, sys
tags_str = sys.argv[2]
try:
    tags_obj = json.loads(tags_str)
except:
    tags_obj = []
print(json.dumps({'content': sys.argv[1], 'mediaUrls': [], 'tags': tags_obj, 'isPublic': True}, ensure_ascii=False))
" "$content" "$tags")
    local resp=$(curl -s -X POST "$BASE_CONTENT/createTweet" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $token" \
        -d "$json_data")
    echo "$resp" | python3 -c "import sys,json; d=json.load(sys.stdin).get('data',{}); print(d.get('snowTid',''))" 2>/dev/null
}

like_tweet() {
    local token=$1
    local target_id=$2
    curl -s -X POST "$BASE_INTERACT/like" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $token" \
        -d "{\"isCreated\":0,\"snowLikesId\":0,\"targetType\":0,\"targetId\":$target_id,\"status\":1}" > /dev/null 2>&1
}

create_comment() {
    local token=$1
    local snow_tid=$2
    local content=$3
    local json_data=$(python3 -c "
import json, sys
print(json.dumps({'snowTid': int(sys.argv[1]), 'content': sys.argv[2], 'parentId': 0, 'rootId': 0}, ensure_ascii=False))
" "$snow_tid" "$content")
    local resp=$(curl -s -X POST "$BASE_INTERACT/createComment" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $token" \
        -d "$json_data")
    echo "$resp" | python3 -c "import sys,json; d=json.load(sys.stdin).get('data',{}); print(d.get('snowCid',''))" 2>/dev/null
}

# 读取 token 数组：TOKENS[i]="token"
load_tokens() {
    TOKENS=()
    if [ -f "$TOKENS_FILE" ]; then
        while IFS= read -r line; do
            TOKENS+=("$line")
        done < "$TOKENS_FILE"
    fi
}

# 读取推文 ID 数组
load_tweets() {
    TWEET_IDS=()
    if [ -f "$TWEETS_FILE" ]; then
        while IFS= read -r line; do
            TWEET_IDS+=("$line")
        done < "$TWEETS_FILE"
    fi
}

# 读取评论数组
load_comments() {
    COMMENT_ENTRIES=()
    if [ -f "$COMMENTS_FILE" ]; then
        while IFS= read -r line; do
            COMMENT_ENTRIES+=("$line")
        done < "$COMMENTS_FILE"
    fi
}

# 批量创建推文：参数为 token_index 和推文数据数组（"content|tags" 格式）
# 返回写入到 TWEETS_FILE
batch_create_tweets() {
    local token_idx=$1
    shift
    local entries=("$@")
    local token="${TOKENS[$token_idx]}"
    local count=0

    for entry in "${entries[@]}"; do
        local content=$(echo "$entry" | cut -d'|' -f1)
        local tags=$(echo "$entry" | cut -d'|' -f2)
        [ -z "$tags" ] && tags="[]"

        local tid=$(create_tweet "$token" "$content" "$tags")
        if [ -n "$tid" ]; then
            echo "$tid" >> "$TWEETS_FILE"
            count=$((count+1))
        fi
    done
    echo "$count"
}

# 检查服务状态
check_services() {
    for PORT in 1001 1002 1003; do
        if ! lsof -iTCP:$PORT -sTCP:LISTEN -P &>/dev/null; then
            echo "端口 :$PORT 未监听，请先启动服务"
            exit 1
        fi
    done
}
