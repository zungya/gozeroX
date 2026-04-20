#!/bin/bash
# 创建全部 300 条推文
source "$(dirname "$0")/common.sh"

load_tokens
if [ ${#TOKENS[@]} -eq 0 ]; then
    echo "错误：未找到用户 token，请先运行 01-register.sh"
    exit 1
fi

USER_COUNT=${#TOKENS[@]}
> "$TWEETS_FILE"

# 加载各分类推文数据
source "$(dirname "$0")/02-tweets-game.sh"
source "$(dirname "$0")/02-tweets-life.sh"
source "$(dirname "$0")/02-tweets-media.sh"
source "$(dirname "$0")/02-tweets-sports-music.sh"
source "$(dirname "$0")/02-tweets-study-mix.sh"

echo -e "${GREEN}[2/4]${NC} 创建推文..."

TOTAL=0

# 分配推文给用户（轮询分配）
create_batch() {
    local data_array_name=$1
    local label=$2
    eval "local entries=(\"\${${data_array_name}[@]}\")"
    local count=0

    for entry in "${entries[@]}"; do
        local content=$(echo "$entry" | cut -d'|' -f1)
        local tags=$(echo "$entry" | cut -d'|' -f2)
        [ -z "$tags" ] && tags="[]"

        # 轮询选择用户
        local idx=$((TOTAL % USER_COUNT))
        local token="${TOKENS[$idx]}"
        [ -z "$token" ] && idx=0 && token="${TOKENS[0]}"

        local tid=$(create_tweet "$token" "$content" "$tags")
        if [ -n "$tid" ]; then
            echo "$tid" >> "$TWEETS_FILE"
            count=$((count+1))
        fi
        TOTAL=$((TOTAL+1))
    done
    echo "  $label: $count 条"
}

create_batch "TWEETS_GAME"        "游戏"
sleep 1
create_batch "TWEETS_LIFE"        "生活"
sleep 1
create_batch "TWEETS_MEDIA"       "影视动漫"
sleep 1
create_batch "TWEETS_FUNNY"       "搞笑"
sleep 1
create_batch "TWEETS_SPORTS"      "体育"
sleep 1
create_batch "TWEETS_MUSIC"       "音乐"
sleep 1
create_batch "TWEETS_SCIENCE"     "知识科普"
sleep 1
create_batch "TWEETS_STUDY"       "学习"
sleep 1
create_batch "TWEETS_MIX"         "跨类混合"

echo "  总计创建: $TOTAL 条推文"
