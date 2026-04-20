#!/bin/bash
# 200 条评论，集中在部分推文上
source "$(dirname "$0")/common.sh"

load_tokens
load_tweets

echo -e "${GREEN}[4/4]${NC} 生成评论..."

if [ ${#TWEET_IDS[@]} -eq 0 ]; then
    echo "错误：未找到推文数据"
    exit 1
fi

USER_COUNT=${#TOKENS[@]}
TWEET_COUNT=${#TWEET_IDS[@]}
COMMENT_TOTAL=0

# 评论内容池
COMMENT_TEXTS=(
    "太赞了，写得很好！"
    "学到了，收藏了"
    "说的太对了吧"
    "这个观点很新颖，从没想过"
    "有道理，不过我觉得还有另一种可能"
    "我也遇到过类似的情况"
    "感谢分享，很有帮助"
    "笑死我了哈哈哈哈"
    "这个我举双手赞同"
    "评论区有没有懂哥详细解释一下"
    "已关注，期待更多内容"
    "好家伙这也太厉害了"
    "真实，太真实了"
    "你是懂xx的"
    "mark一下回头细看"
    "前排占座"
    "这也行？涨知识了"
    "建议置顶"
    "看完想打人，不是想打你是想打那个让你写这个的"
    "下次多写点这种内容！"
    "这个思路绝了"
    "补充一个冷知识"
    "同感！终于有人说出来了"
    "在？为什么不早发？等很久了"
    "这也太干货了吧"
)

# 评论集中在约 60-80 条推文上（约 25% 的推文）
# 从 300 条推文中随机选 70 条重点推文
FOCUS_COUNT=70
FOCUS_TWEET_INDICES=()
selected=" "
while [ ${#FOCUS_TWEET_INDICES[@]} -lt $FOCUS_COUNT ]; do
    idx=$((RANDOM % TWEET_COUNT))
    case "$selected" in
        *" $idx "*) continue ;;
    esac
    selected="$selected$idx "
    FOCUS_TWEET_INDICES+=("$idx")
done

# 为这 70 条推文分配评论数（正态分布）
# 10条推文各5-7条评论, 20条推文各3-4条, 40条推文各1-2条
# 总计约: 10*6 + 20*3.5 + 40*1.5 = 60+70+60 = 190，再加上一些随机 ≈ 200
> "$COMMENTS_FILE"

for i in $(seq 0 $((FOCUS_COUNT - 1))); do
    tweet_idx="${FOCUS_TWEET_INDICES[$i]}"
    tid="${TWEET_IDS[$tweet_idx]}"

    # 分配评论数
    if [ $i -lt 10 ]; then
        comment_count=$((5 + RANDOM % 3))   # 5-7
    elif [ $i -lt 30 ]; then
        comment_count=$((3 + RANDOM % 2))   # 3-4
    else
        comment_count=$((1 + RANDOM % 2))   # 1-2
    fi

    for j in $(seq 1 $comment_count); do
        user_idx=$((RANDOM % USER_COUNT))
        [ -z "${TOKENS[$user_idx]}" ] && continue

        text_idx=$((RANDOM % ${#COMMENT_TEXTS[@]}))
        cid=$(create_comment "${TOKENS[$user_idx]}" "$tid" "${COMMENT_TEXTS[$text_idx]}")
        if [ -n "$cid" ]; then
            echo "$tid|$cid" >> "$COMMENTS_FILE"
            COMMENT_TOTAL=$((COMMENT_TOTAL+1))
        fi
    done
done

echo "  评论完成: $COMMENT_TOTAL 条 (集中在 $FOCUS_COUNT 条推文上)"
