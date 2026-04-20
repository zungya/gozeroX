#!/bin/bash
# 600 条点赞，正态分布
# 大部分推文 1-3 个赞，少数热门推文 10-25 个赞
source "$(dirname "$0")/common.sh"

load_tokens
load_tweets

echo -e "${GREEN}[3/4]${NC} 生成点赞..."

if [ ${#TWEET_IDS[@]} -eq 0 ]; then
    echo "错误：未找到推文数据"
    exit 1
fi

USER_COUNT=${#TOKENS[@]}
TWEET_COUNT=${#TWEET_IDS[@]}
LIKE_TOTAL=0

# 正态分布模拟：用权重数组决定每条推文获得多少赞
# 权重分布：10%推文拿高赞(10-25), 20%推文拿中赞(5-9), 70%推文拿低赞(1-3)
assign_likes() {
    local tid=$1
    local rand=$RANDOM
    if [ $rand -lt 3276 ]; then
        # ~10%: 热门推文 10-25 赞
        echo $((10 + RANDOM % 16))
    elif [ $rand -lt 9830 ]; then
        # ~20%: 中等推文 5-9 赞
        echo $((5 + RANDOM % 5))
    else
        # ~70%: 普通推文 1-3 赞
        echo $((1 + RANDOM % 3))
    fi
}

# 为每条推文分配点赞数
for i in $(seq 0 $((TWEET_COUNT - 1))); do
    tid="${TWEET_IDS[$i]}"
    like_count=$(assign_likes "$tid")

    # 随机选择不重复的用户点赞（用字符串代替 associative array）
    used_users=" "
    for j in $(seq 1 $like_count); do
        attempts=0
        while [ $attempts -lt 20 ]; do
            user_idx=$((RANDOM % USER_COUNT))
            [ -z "${TOKENS[$user_idx]}" ] && { attempts=$((attempts+1)); continue; }
            case "$used_users" in
                *" $user_idx "*) attempts=$((attempts+1)); continue ;;
            esac
            used_users="$used_users$user_idx "
            like_tweet "${TOKENS[$user_idx]}" "$tid"
            LIKE_TOTAL=$((LIKE_TOTAL+1))
            break
        done
    done

    # 每处理50条打印进度
    if [ $((i % 50)) -eq 49 ]; then
        echo "  进度: $((i+1))/$TWEET_COUNT 推文, $LIKE_TOTAL 次点赞"
    fi
done

echo "  点赞完成: $LIKE_TOTAL 次"
