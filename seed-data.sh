#!/bin/bash
# gozeroX 数据注入脚本
# 注册用户、创建推文、点赞、评论，形成小规模社交数据
# 使用方法: bash seed-data.sh
# 前提: 所有服务已启动 (bash start-local.sh)

set -e

GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m'

BASE_USER="http://localhost:1001/usercenter/v1"
BASE_CONTENT="http://localhost:1002/contentService/v1"
BASE_INTERACT="http://localhost:1003/interactService/v1"

# ========== 工具函数 ==========

# 注册用户，返回 "accessToken|uid"
register_user() {
    local mobile=$1
    local password=$2
    local resp=$(curl -s -X POST "$BASE_USER/user/register" \
        -H "Content-Type: application/json" \
        -d "{\"mobile\":\"$mobile\",\"password\":\"$password\"}")
    local token=$(echo "$resp" | python3 -c "import sys,json; print(json.load(sys.stdin)['accessToken'])" 2>/dev/null)
    local uid=$(echo "$resp" | python3 -c "import sys,json; print(json.load(sys.stdin)['userInfo']['uid'])" 2>/dev/null)
    echo "${token}|${uid}"
}

# 创建推文，返回 snowTid
create_tweet() {
    local token=$1
    local content=$2
    local tags=$3
    local resp=$(curl -s -X POST "$BASE_CONTENT/createTweet" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $token" \
        -d "{\"content\":\"$content\",\"mediaUrls\":[],\"tags\":$tags,\"isPublic\":true}")
    echo "$resp" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['snowTid'])" 2>/dev/null
}

# 点赞推文，返回 snowLikesId
like_tweet() {
    local token=$1
    local target_id=$2
    local resp=$(curl -s -X POST "$BASE_INTERACT/like" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $token" \
        -d "{\"isCreated\":0,\"snowLikesId\":0,\"targetType\":0,\"targetId\":$target_id,\"status\":1}")
    echo "$resp" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['snowLikesId'])" 2>/dev/null
}

# 点赞评论
like_comment() {
    local token=$1
    local target_id=$2
    local snow_tid=$3
    local resp=$(curl -s -X POST "$BASE_INTERACT/like" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $token" \
        -d "{\"isCreated\":0,\"snowLikesId\":0,\"targetType\":1,\"targetId\":$target_id,\"snowTid\":$snow_tid,\"status\":1}")
    echo "$resp" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['snowLikesId'])" 2>/dev/null
}

# 创建评论，返回 snowCid
create_comment() {
    local token=$1
    local snow_tid=$2
    local content=$3
    local resp=$(curl -s -X POST "$BASE_INTERACT/createComment" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $token" \
        -d "{\"snowTid\":$snow_tid,\"content\":\"$content\",\"parentId\":0,\"rootId\":0}")
    echo "$resp" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['snowCid'])" 2>/dev/null
}

# 创建回复，返回 snowCid
create_reply() {
    local token=$1
    local snow_tid=$2
    local content=$3
    local parent_id=$4
    local root_id=$5
    local resp=$(curl -s -X POST "$BASE_INTERACT/createComment" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $token" \
        -d "{\"snowTid\":$snow_tid,\"content\":\"$content\",\"parentId\":$parent_id,\"rootId\":$root_id}")
    echo "$resp" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['snowCid'])" 2>/dev/null
}

# ========== 检查服务 ==========

echo -e "${YELLOW}检查服务状态...${NC}"
for PORT in 1001 1002 1003; do
    if ! lsof -iTCP:$PORT -sTCP:LISTEN -P &>/dev/null; then
        echo "端口 :$PORT 未监听，请先启动服务"
        exit 1
    fi
done
echo "服务就绪"
echo ""

# ========== 1. 注册用户 (10个) ==========

echo -e "${GREEN}[1/5]${NC} 注册用户..."

# 声明数组
declare -a TOKENS
declare -a UIDS
MOBILE_PREFIXS=("138" "139" "136" "137" "158" "159" "188" "187" "135" "133")

for i in $(seq 0 9); do
    mobile="${MOBILE_PREFIXS[$i]}0000000$((i+1))"
    password="test123456"
    result=$(register_user "$mobile" "$password" "user$((i+1))")
    token=$(echo "$result" | cut -d'|' -f1)
    uid=$(echo "$result" | cut -d'|' -f2)
    if [ -z "$token" ] || [ "$token" = "" ]; then
        echo "  用户$((i+1)) 注册失败，跳过"
        TOKENS+=("")
        continue
    fi
    TOKENS+=("$token")
    UIDS+=("$uid")
    echo "  用户$((i+1)) uid=$uid"
done

# 有效用户数
VALID_COUNT=0
for t in "${TOKENS[@]}"; do
    [ -n "$t" ] && VALID_COUNT=$((VALID_COUNT+1))
done
echo "  注册完成: $VALID_COUNT 个用户"
echo ""

# ========== 2. 创建推文 (每用户 3-5 条) ==========

echo -e "${GREEN}[2/5]${NC} 创建推文..."

declare -a ALL_TWEETS  # 所有推文ID
TWEET_CONTENTS=(
    "今天天气真好，适合写代码！ #daily"
    "分享一个 Go 微服务开发的小技巧 #golang #microservice"
    "推荐系统真的很重要，用户体验提升很多 #recommend"
    "刚学会用 Kafka 做异步处理，效率提升不少 #kafka"
    "Docker 部署真方便，一键启动所有服务 #docker"
    "Redis 缓存策略需要仔细设计，不然容易踩坑 #redis"
    "PostgreSQL 的 JSON 查询功能很强大 #postgresql"
    "微服务之间的 RPC 通信用 gRPC 真的高效 #grpc"
    "JWT 鉴权在微服务架构中的最佳实践 #jwt"
    "goctl 代码生成工具太好用了，省了很多时间 #gozero"
    "今天读了一篇关于分布式事务的好文章 #distributed"
    "推荐一本好书：《Designing Data-Intensive Applications》#reading"
    "API 网关在微服务架构中的作用不可忽视 #gateway"
    "单元测试覆盖率要提上去了 #testing"
    "CI/CD 流水线搭建完成，自动化部署真香 #cicd"
    "消息队列的消费者幂等性处理很重要 #idempotent"
    "服务注册与发现用 Etcd 挺好用的 #etcd"
    "日志聚合方案选型：ELK vs Loki #logging"
    "链路追踪对排查微服务问题帮助很大 #tracing"
    "灰度发布策略小结 #canary"
    "这个社交 App 做的不错啊 #social"
    "大家有什么好的监控方案推荐吗 #monitoring"
    "周末出去爬山了，风景很美 #weekend"
    "学 Rust 中，所有权机制真的很有意思 #rust"
    "代码 Review 的重要性再怎么强调都不过分 #codereview"
    "新项目准备用 Clean Architecture #architecture"
    "缓存击穿、穿透、雪崩的区别和解决方案 #cache"
    "数据库索引优化的几个常见技巧 #database"
    "今天团建，大家都很开心 #team"
    "开了一个新的技术博客，欢迎关注 #blog"
    "关于微服务拆分粒度的思考 #design"
    "用了 Go 1.22 的新特性，range over func 真方便 #go"
    "K8s 部署踩坑记录 #k8s"
    "前端 React 和 Vue 怎么选？ #frontend"
    "最近在研究向量数据库 #vectordb"
    "技术债务需要定期清理 #techdebt"
    "高并发下的限流策略比较 #ratelimit"
    "今天推文发完了 #done"
    "系统稳定性建设的一些经验 #reliability"
    "学好数据结构和算法真的很重要 #algorithm"
    "云原生应用开发的几点体会 #cloudnative"
    "写好文档也是一种能力 #docs"
    "Go 的错误处理哲学 #errorhandling"
    "对 Serverless 架构的一些看法 #serverless"
    "性能优化从监控开始 #performance"
    "设计模式在日常开发中的应用 #patterns"
    "代码简洁之道 #cleancode"
    "关于技术选型的几个原则 #techchoice"
    "开源社区贡献指南 #opensource"
    "分布式系统的一致性问题 #consistency"
)

idx=0
for i in $(seq 0 9); do
    [ -z "${TOKENS[$i]}" ] && continue
    # 每用户创建 3-5 条推文
    count=$((3 + RANDOM % 3))
    for j in $(seq 1 $count); do
        if [ $idx -ge ${#TWEET_CONTENTS[@]} ]; then
            idx=0
        fi
        content="${TWEET_CONTENTS[$idx]}"
        tags=$(echo "$content" | grep -oE '#[^ ]+' | sed 's/^#/"/;s/$/"/' | tr '\n' ',' | sed 's/,$//' | sed 's/^/[/;s/$/]/')
        if [ -z "$tags" ]; then
            tags="[]"
        fi
        # 去掉 hashtags
        clean_content=$(echo "$content" | sed 's/ #[^ ]*//g')

        tid=$(create_tweet "${TOKENS[$i]}" "$clean_content" "$tags")
        if [ -n "$tid" ]; then
            ALL_TWEETS+=("$tid")
            echo "  用户$((i+1)) 推文#$j tid=$tid"
        fi
        idx=$((idx+1))
    done
done

echo "  创建完成: ${#ALL_TWEETS[@]} 条推文"
echo "  等待 MQ 消费推文..."
sleep 1
echo ""

echo -e "${GREEN}[3/5]${NC} 点赞推文..."

LIKE_COUNT=0
for i in $(seq 0 9); do
    [ -z "${TOKENS[$i]}" ] && continue
    # 每用户随机点赞 5-10 条推文
    count=$((5 + RANDOM % 6))
    for j in $(seq 1 $count); do
        # 随机选一条推文（不点赞自己的）
        while true; do
            rand_idx=$((RANDOM % ${#ALL_TWEETS[@]}))
            tid="${ALL_TWEETS[$rand_idx]}"
            # 简单跳过自己的推文（约 1/10 概率是自己）
            break
        done
        lid=$(like_tweet "${TOKENS[$i]}" "$tid")
        if [ -n "$lid" ]; then
            LIKE_COUNT=$((LIKE_COUNT+1))
        fi
    done
done

echo "  点赞完成: $LIKE_COUNT 次"
echo "  等待 MQ 消费点赞..."
sleep 1
echo ""

echo -e "${GREEN}[4/5]${NC} 创建评论..."

declare -a ALL_COMMENTS  # "tid|cid" 格式
COMMENT_TEXTS=(
    "写得很好，学习了！"
    "赞同你的观点"
    "有道理，我也遇到过类似的情况"
    "分享得不错，谢谢"
    "这个思路很新颖"
    "补充一点：实践比理论更重要"
    "推荐相关的资料可以参考"
    "期待更多分享"
    "已收藏，回头细看"
    "和我的想法不谋而合"
    "这个坑我也踩过"
    "不错不错，继续加油"
)

COMMENT_COUNT=0
for tid in "${ALL_TWEETS[@]}"; do
    # 每条推文随机 0-3 条评论
    count=$((RANDOM % 4))
    for j in $(seq 1 $count); do
        # 随机选一个用户评论
        user_idx=$((RANDOM % 10))
        [ -z "${TOKENS[$user_idx]}" ] && continue

        text_idx=$((RANDOM % ${#COMMENT_TEXTS[@]}))
        cid=$(create_comment "${TOKENS[$user_idx]}" "$tid" "${COMMENT_TEXTS[$text_idx]}")
        if [ -n "$cid" ]; then
            ALL_COMMENTS+=("$tid|$cid")
            COMMENT_COUNT=$((COMMENT_COUNT+1))
        fi
    done
done

echo "  评论完成: $COMMENT_COUNT 条"
echo "  等待 MQ 消费评论..."
sleep 1
echo ""

echo -e "${GREEN}[5/5]${NC} 创建回复 & 点赞评论..."

REPLY_TEXTS=(
    "谢谢支持！"
    "确实是这样"
    "我也这么觉得"
    "好的，回头我整理一下"
    "一起交流学习"
)

REPLY_COUNT=0
COMMENT_LIKE_COUNT=0

for entry in "${ALL_COMMENTS[@]}"; do
    tid=$(echo "$entry" | cut -d'|' -f1)
    cid=$(echo "$entry" | cut -d'|' -f2)

    # 50% 概率有回复
    if [ $((RANDOM % 2)) -eq 0 ]; then
        user_idx=$((RANDOM % 10))
        [ -z "${TOKENS[$user_idx]}" ] && continue
        reply_idx=$((RANDOM % ${#REPLY_TEXTS[@]}))
        rid=$(create_reply "${TOKENS[$user_idx]}" "$tid" "${REPLY_TEXTS[$reply_idx]}" "$cid" "$cid")
        if [ -n "$rid" ]; then
            REPLY_COUNT=$((REPLY_COUNT+1))
        fi
    fi

    # 30% 概率给评论点赞
    if [ $((RANDOM % 10)) -lt 3 ]; then
        user_idx=$((RANDOM % 10))
        [ -z "${TOKENS[$user_idx]}" ] && continue
        lid=$(like_comment "${TOKENS[$user_idx]}" "$cid" "$tid")
        if [ -n "$lid" ]; then
            COMMENT_LIKE_COUNT=$((COMMENT_LIKE_COUNT+1))
        fi
    fi
done

echo "  回复完成: $REPLY_COUNT 条"
echo "  评论点赞完成: $COMMENT_LIKE_COUNT 次"
echo ""

# ========== 等待 MQ 消费 ==========

echo -e "${YELLOW}等待 MQ 消费者处理消息 (5秒)...${NC}"
sleep 5

# ========== 汇总 ==========

echo ""
echo "============================================"
echo -e "${GREEN}数据注入完成！${NC}"
echo "============================================"
echo "  用户:      $VALID_COUNT 个"
echo "  推文:      ${#ALL_TWEETS[@]} 条"
echo "  推文点赞:  $LIKE_COUNT 次"
echo "  评论:      $COMMENT_COUNT 条"
echo "  回复:      $REPLY_COUNT 条"
echo "  评论点赞:  $COMMENT_LIKE_COUNT 次"
echo ""
echo "测试账号: mobile=13800000001 ~ 13800000010, password=test123456"
