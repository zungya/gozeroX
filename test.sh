#!/bin/bash
# gozeroX 集成测试脚本
# 使用方法: bash test.sh
# 前提: docker compose -f docker-compose-env.yml up -d 已启动，所有 Go 服务已启动
# 覆盖: 5 个服务共 24 个接口 + Python 召回直连
# 设计: 使用 3 个用户 (A/B/C) 交叉互动，确保通知、点赞列表可被验证

# ============ 配置 ============
NO_PROXY="--noproxy '*'"
BASE_USER="http://localhost:1001/usercenter/v1"
BASE_CONTENT="http://localhost:1002/contentService/v1"
BASE_INTERACT="http://localhost:1003/interactService/v1"
BASE_NOTICE="http://localhost:1004/noticeService/v1"
BASE_RECOMMEND="http://localhost:1005/recommendService/v1"

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m'

PASS_COUNT=0
FAIL_COUNT=0
SKIP_COUNT=0

pass() { echo -e "${GREEN}[PASS]${NC} $1"; PASS_COUNT=$((PASS_COUNT+1)); }
fail() { echo -e "${RED}[FAIL]${NC} $1"; FAIL_COUNT=$((FAIL_COUNT+1)); }
skip() { echo -e "${YELLOW}[SKIP]${NC} $1"; SKIP_COUNT=$((SKIP_COUNT+1)); }

separator() { echo ""; echo "============================================"; echo "$1"; echo "============================================"; }

# 提取 JSON 字段的辅助函数
json_val() { echo "$1" | python3 -c "import sys,json; print(json.load(sys.stdin)$2)" 2>/dev/null; }

# 注册用户并返回 token/uid（$1=变量前缀）
register_user() {
    local PREFIX="$1"
    local MOBILE="139$((RANDOM % 10000))"
    MOBILE="${MOBILE}0000"
    MOBILE="${MOBILE:0:11}"

    local RESP=$(curl $NO_PROXY -s -X POST "$BASE_USER/user/register" \
      -H "Content-Type: application/json" \
      -d "{\"mobile\":\"$MOBILE\",\"password\":\"test1234\"}")

    local CODE=$(json_val "$RESP" ".get('code',-1)")
    if [ "$CODE" = "0" ]; then
        eval "${PREFIX}_TOKEN=$(json_val "$RESP" ".get('accessToken','')")"
        eval "${PREFIX}_UID=$(json_val "$RESP" ".get('userInfo',{}).get('uid',0)")"
        eval "${PREFIX}_MOBILE=$MOBILE"
        return 0
    else
        # 注册失败，尝试登录
        local LOGIN_RESP=$(curl $NO_PROXY -s -X POST "$BASE_USER/user/login" \
          -H "Content-Type: application/json" \
          -d "{\"mobile\":\"$MOBILE\",\"password\":\"test1234\"}")
        eval "${PREFIX}_TOKEN=$(json_val "$LOGIN_RESP" ".get('accessToken','')")"
        eval "${PREFIX}_UID=$(json_val "$LOGIN_RESP" ".get('userInfo',{}).get('uid',0)")"
        eval "${PREFIX}_MOBILE=$MOBILE"
        return 0
    fi
}

# ============ 1. 注册用户 A ============
separator "1. 注册用户 A (usercenter)"

register_user UA
echo "  用户A: uid=$UA_UID, mobile=$UA_MOBILE"

if [ -n "$UA_TOKEN" ] && [ "$UA_TOKEN" != "" ] && [ "$UA_TOKEN" != "None" ]; then
    pass "用户A 就绪, uid: $UA_UID"
else
    fail "用户A 注册/登录失败"
    exit 1
fi

UA_AUTH="Authorization: Bearer $UA_TOKEN"

# ============ 2. 注册用户 B ============
separator "2. 注册用户 B (usercenter)"

register_user UB
echo "  用户B: uid=$UB_UID, mobile=$UB_MOBILE"

if [ -n "$UB_TOKEN" ] && [ "$UB_TOKEN" != "" ] && [ "$UB_TOKEN" != "None" ]; then
    pass "用户B 就绪, uid: $UB_UID"
else
    fail "用户B 注册/登录失败"
    exit 1
fi

UB_AUTH="Authorization: Bearer $UB_TOKEN"

# ============ 3. 注册用户 C ============
separator "3. 注册用户 C (usercenter)"

register_user UC
echo "  用户C: uid=$UC_UID, mobile=$UC_MOBILE"

if [ -n "$UC_TOKEN" ] && [ "$UC_TOKEN" != "" ] && [ "$UC_TOKEN" != "None" ]; then
    pass "用户C 就绪, uid: $UC_UID"
else
    fail "用户C 注册/登录失败"
    exit 1
fi

UC_AUTH="Authorization: Bearer $UC_TOKEN"

# ============ 4. 获取用户信息 ============
separator "4. 获取用户信息 (usercenter)"

DETAIL_RESP=$(curl $NO_PROXY -s -X POST "$BASE_USER/user/detail" \
  -H "Content-Type: application/json" \
  -H "$UA_AUTH" \
  -d "{\"uid\":$UA_UID}")

echo "$DETAIL_RESP" | python3 -m json.tool 2>/dev/null || echo "$DETAIL_RESP"

DETAIL_UID=$(json_val "$DETAIL_RESP" ".get('userInfo',{}).get('uid',0)")
if [ "$DETAIL_UID" -gt 0 ] 2>/dev/null; then
    DETAIL_NICK=$(json_val "$DETAIL_RESP" ".get('userInfo',{}).get('nickname','')")
    DETAIL_POSTS=$(json_val "$DETAIL_RESP" ".get('userInfo',{}).get('postCount',0)")
    pass "获取用户A信息成功, nickname: $DETAIL_NICK, postCount: $DETAIL_POSTS"
else
    fail "获取用户信息失败"
fi

# ============ 5. 用户A 发布推文 ============
separator "5. 用户A 发布推文 (contentService)"

TWEET_RESP=$(curl $NO_PROXY -s -X POST "$BASE_CONTENT/createTweet" \
  -H "Content-Type: application/json" \
  -H "$UA_AUTH" \
  -d "{\"content\":\"Test tweet from A at $(date +%H:%M:%S)\",\"mediaUrls\":[],\"tags\":[\"test\",\"自动化\"],\"isPublic\":true}")

echo "$TWEET_RESP" | python3 -m json.tool 2>/dev/null || echo "$TWEET_RESP"

TWEET_CODE=$(json_val "$TWEET_RESP" ".get('code',-1)")
SNOW_TID=$(json_val "$TWEET_RESP" ".get('data',{}).get('snowTid','0')")

if [ "$TWEET_CODE" = "0" ] && [ "$SNOW_TID" != "0" ] && [ -n "$SNOW_TID" ]; then
    pass "用户A 发布推文成功, snowTid: $SNOW_TID"
else
    TWEET_MSG=$(json_val "$TWEET_RESP" ".get('msg','')")
    fail "发布推文失败, code: $TWEET_CODE, msg: $TWEET_MSG"
fi

# ============ 6. 获取推文详情 getTweet ============
separator "6. 获取推文详情 (contentService)"

if [ -n "$SNOW_TID" ] && [ "$SNOW_TID" != "0" ]; then
    GET_TWEET_RESP=$(curl $NO_PROXY -s "$BASE_CONTENT/getTweet?snowTid=$SNOW_TID" \
      -H "$UA_AUTH")

    echo "$GET_TWEET_RESP" | python3 -m json.tool 2>/dev/null || echo "$GET_TWEET_RESP"

    GT_CODE=$(json_val "$GET_TWEET_RESP" ".get('code',-1)")
    GT_SNOW_TID=$(json_val "$GET_TWEET_RESP" ".get('data',{}).get('snowTid','')")
    GT_CONTENT=$(json_val "$GET_TWEET_RESP" ".get('data',{}).get('content','')")

    if [ "$GT_CODE" = "0" ] && [ "$GT_SNOW_TID" = "$SNOW_TID" ] && [ -n "$GT_CONTENT" ] && [ "$GT_CONTENT" != "None" ]; then
        pass "getTweet 成功, snowTid: $GT_SNOW_TID, content 非空"
    else
        fail "getTweet 失败或数据不匹配, code: $GT_CODE, snowTid: $GT_SNOW_TID (expected: $SNOW_TID)"
    fi
else
    skip "getTweet（没有有效推文ID）"
fi

# ============ 7. 推文列表 ============
separator "7. 推文列表 (contentService)"

LIST_RESP=$(curl $NO_PROXY -s "$BASE_CONTENT/listTweets?queryUid=$UA_UID&limit=5" \
  -H "$UA_AUTH")

echo "$LIST_RESP" | python3 -m json.tool 2>/dev/null || echo "$LIST_RESP"

LIST_CODE=$(json_val "$LIST_RESP" ".get('code',-1)")
if [ "$LIST_CODE" = "0" ]; then
    TOTAL=$(json_val "$LIST_RESP" ".get('total',0)")
    ITEM_COUNT=$(json_val "$LIST_RESP" ".get('data',[]).__len__()")
    pass "推文列表查询成功, total: $TOTAL, 本页: $ITEM_COUNT 条"
else
    fail "推文列表查询失败"
fi

# ============ 8. 用户B 点赞用户A的推文 ============
separator "8. 用户B 点赞用户A的推文 (interactService)"

SNOW_LIKES_ID_B=""

if [ -n "$SNOW_TID" ] && [ "$SNOW_TID" != "0" ]; then
    LIKE_RESP=$(curl $NO_PROXY -s -X POST "$BASE_INTERACT/like" \
      -H "Content-Type: application/json" \
      -H "$UB_AUTH" \
      -d "{\"isCreated\":0,\"snowLikesId\":\"0\",\"targetType\":0,\"targetId\":\"$SNOW_TID\",\"status\":1}")

    echo "$LIKE_RESP" | python3 -m json.tool 2>/dev/null || echo "$LIKE_RESP"

    LIKE_CODE=$(json_val "$LIKE_RESP" ".get('code',-1)")
    if [ "$LIKE_CODE" = "0" ]; then
        SNOW_LIKES_ID_B=$(json_val "$LIKE_RESP" ".get('data',{}).get('snowLikesId','0')")
        LIKE_STATUS=$(json_val "$LIKE_RESP" ".get('data',{}).get('status',-1)")
        pass "用户B 点赞A的推文成功, snowLikesId: $SNOW_LIKES_ID_B, status: $LIKE_STATUS"
    else
        LIKE_MSG=$(json_val "$LIKE_RESP" ".get('message','')")
        fail "用户B 点赞失败, code: $LIKE_CODE, message: $LIKE_MSG"
    fi
else
    skip "用户B 点赞（没有有效推文ID）"
fi

# ============ 9. 用户B 取消点赞 ============
separator "9. 用户B 取消点赞 (interactService)"

if [ -n "$SNOW_LIKES_ID_B" ] && [ "$SNOW_LIKES_ID_B" != "0" ]; then
    UNLIKE_RESP=$(curl $NO_PROXY -s -X POST "$BASE_INTERACT/like" \
      -H "Content-Type: application/json" \
      -H "$UB_AUTH" \
      -d "{\"isCreated\":1,\"snowLikesId\":\"$SNOW_LIKES_ID_B\",\"targetType\":0,\"targetId\":\"$SNOW_TID\",\"status\":0}")

    echo "$UNLIKE_RESP" | python3 -m json.tool 2>/dev/null || echo "$UNLIKE_RESP"

    UNLIKE_CODE=$(json_val "$UNLIKE_RESP" ".get('code',-1)")
    if [ "$UNLIKE_CODE" = "0" ]; then
        UNLIKE_STATUS=$(json_val "$UNLIKE_RESP" ".get('data',{}).get('status',-1)")
        pass "用户B 取消点赞成功, status: $UNLIKE_STATUS"
    else
        fail "用户B 取消点赞失败, code: $UNLIKE_CODE"
    fi
else
    skip "用户B 取消点赞（没有有效的点赞记录ID）"
fi

# ============ 10. 用户B 再次点赞（保留状态）============
separator "10. 用户B 再次点赞 (interactService)"

if [ -n "$SNOW_LIKES_ID_B" ] && [ "$SNOW_LIKES_ID_B" != "0" ]; then
    RELIKE_RESP=$(curl $NO_PROXY -s -X POST "$BASE_INTERACT/like" \
      -H "Content-Type: application/json" \
      -H "$UB_AUTH" \
      -d "{\"isCreated\":1,\"snowLikesId\":\"$SNOW_LIKES_ID_B\",\"targetType\":0,\"targetId\":\"$SNOW_TID\",\"status\":1}")

    echo "$RELIKE_RESP" | python3 -m json.tool 2>/dev/null || echo "$RELIKE_RESP"

    RELIKE_CODE=$(json_val "$RELIKE_RESP" ".get('code',-1)")
    if [ "$RELIKE_CODE" = "0" ]; then
        pass "用户B 再次点赞成功"
    else
        fail "用户B 再次点赞失败, code: $RELIKE_CODE"
    fi
else
    skip "用户B 再次点赞（没有有效的点赞记录ID）"
fi

# ============ 11. 用户C 点赞用户A的推文 ============
separator "11. 用户C 点赞用户A的推文 (interactService)"

SNOW_LIKES_ID_C=""

if [ -n "$SNOW_TID" ] && [ "$SNOW_TID" != "0" ]; then
    LIKE_C_RESP=$(curl $NO_PROXY -s -X POST "$BASE_INTERACT/like" \
      -H "Content-Type: application/json" \
      -H "$UC_AUTH" \
      -d "{\"isCreated\":0,\"snowLikesId\":\"0\",\"targetType\":0,\"targetId\":\"$SNOW_TID\",\"status\":1}")

    echo "$LIKE_C_RESP" | python3 -m json.tool 2>/dev/null || echo "$LIKE_C_RESP"

    LIKE_C_CODE=$(json_val "$LIKE_C_RESP" ".get('code',-1)")
    if [ "$LIKE_C_CODE" = "0" ]; then
        SNOW_LIKES_ID_C=$(json_val "$LIKE_C_RESP" ".get('data',{}).get('snowLikesId','0')")
        pass "用户C 点赞A的推文成功, snowLikesId: $SNOW_LIKES_ID_C"
    else
        fail "用户C 点赞失败"
    fi
else
    skip "用户C 点赞（没有有效推文ID）"
fi

# ============ 12. 用户B 发表根评论（对A的推文）============
separator "12. 用户B 发表根评论 (interactService)"

SNOW_CID=""

if [ -n "$SNOW_TID" ] && [ "$SNOW_TID" != "0" ]; then
    COMMENT_RESP=$(curl $NO_PROXY -s -X POST "$BASE_INTERACT/createComment" \
      -H "Content-Type: application/json" \
      -H "$UB_AUTH" \
      -d "{\"snowTid\":\"$SNOW_TID\",\"content\":\"B comments on A's tweet!\",\"parentId\":\"0\",\"rootId\":\"0\"}")

    echo "$COMMENT_RESP" | python3 -m json.tool 2>/dev/null || echo "$COMMENT_RESP"

    COMMENT_CODE=$(json_val "$COMMENT_RESP" ".get('code',-1)")
    if [ "$COMMENT_CODE" = "0" ]; then
        SNOW_CID=$(json_val "$COMMENT_RESP" ".get('data',{}).get('snowCid','0')")
        pass "用户B 发表根评论成功, snowCid: $SNOW_CID"
    else
        COMMENT_MSG=$(json_val "$COMMENT_RESP" ".get('message','')")
        fail "用户B 发表根评论失败, code: $COMMENT_CODE, message: $COMMENT_MSG"
    fi
else
    skip "用户B 发表根评论（没有有效推文ID）"
fi

# ============ 13. 获取推文评论列表 ============
separator "13. 获取推文评论列表 (interactService)"

if [ -n "$SNOW_TID" ] && [ "$SNOW_TID" != "0" ]; then
    GET_COMMENTS_RESP=$(curl $NO_PROXY -s "$BASE_INTERACT/getComments?snowTid=$SNOW_TID&limit=10" \
      -H "$UA_AUTH")

    echo "$GET_COMMENTS_RESP" | python3 -m json.tool 2>/dev/null || echo "$GET_COMMENTS_RESP"

    GC_CODE=$(json_val "$GET_COMMENTS_RESP" ".get('code',-1)")
    if [ "$GC_CODE" = "0" ]; then
        GC_TOTAL=$(json_val "$GET_COMMENTS_RESP" ".get('total',0)")
        GC_COUNT=$(json_val "$GET_COMMENTS_RESP" ".get('data',[]).__len__()")
        pass "获取评论列表成功, total: $GC_TOTAL, 本页: $GC_COUNT 条"
    else
        fail "获取评论列表失败, code: $GC_CODE"
    fi
else
    skip "获取评论列表（没有有效推文ID）"
fi

# ============ 14. 用户C 发表回复（对B的评论）============
separator "14. 用户C 回复用户B的评论 (interactService)"

REPLY_CID=""

if [ -n "$SNOW_CID" ] && [ "$SNOW_CID" != "0" ]; then
    REPLY_RESP=$(curl $NO_PROXY -s -X POST "$BASE_INTERACT/createComment" \
      -H "Content-Type: application/json" \
      -H "$UC_AUTH" \
      -d "{\"snowTid\":\"$SNOW_TID\",\"content\":\"C replies to B's comment!\",\"parentId\":\"$SNOW_CID\",\"rootId\":\"$SNOW_CID\"}")

    echo "$REPLY_RESP" | python3 -m json.tool 2>/dev/null || echo "$REPLY_RESP"

    REPLY_CODE=$(json_val "$REPLY_RESP" ".get('code',-1)")
    if [ "$REPLY_CODE" = "0" ]; then
        REPLY_CID=$(json_val "$REPLY_RESP" ".get('data',{}).get('snowCid','0')")
        pass "用户C 回复B的评论成功, replyCid: $REPLY_CID"
    else
        REPLY_MSG=$(json_val "$REPLY_RESP" ".get('message','')")
        fail "用户C 回复失败, code: $REPLY_CODE, message: $REPLY_MSG"
    fi
else
    skip "用户C 回复（没有有效根评论ID）"
fi

# ============ 15. 获取回复列表 ============
separator "15. 获取回复列表 (interactService)"

if [ -n "$SNOW_CID" ] && [ "$SNOW_CID" != "0" ]; then
    GET_REPLIES_RESP=$(curl $NO_PROXY -s "$BASE_INTERACT/getReplies?rootCid=$SNOW_CID&limit=10" \
      -H "$UB_AUTH")

    echo "$GET_REPLIES_RESP" | python3 -m json.tool 2>/dev/null || echo "$GET_REPLIES_RESP"

    GR_CODE=$(json_val "$GET_REPLIES_RESP" ".get('code',-1)")
    if [ "$GR_CODE" = "0" ]; then
        GR_TOTAL=$(json_val "$GET_REPLIES_RESP" ".get('total',0)")
        pass "获取回复列表成功, total: $GR_TOTAL"
    else
        fail "获取回复列表失败, code: $GR_CODE"
    fi
else
    skip "获取回复列表（没有有效根评论ID）"
fi

# ============ 16. 用户B 点赞用户C的评论 ============
separator "16. 用户B 点赞用户C的评论 (interactService)"

SNOW_LIKES_COMMENT_ID=""

if [ -n "$REPLY_CID" ] && [ "$REPLY_CID" != "0" ]; then
    LIKE_COMMENT_RESP=$(curl $NO_PROXY -s -X POST "$BASE_INTERACT/like" \
      -H "Content-Type: application/json" \
      -H "$UB_AUTH" \
      -d "{\"isCreated\":0,\"snowLikesId\":\"0\",\"targetType\":1,\"targetId\":\"$REPLY_CID\",\"snowTid\":\"$SNOW_TID\",\"status\":1,\"isReply\":1}")

    echo "$LIKE_COMMENT_RESP" | python3 -m json.tool 2>/dev/null || echo "$LIKE_COMMENT_RESP"

    LC_CODE=$(json_val "$LIKE_COMMENT_RESP" ".get('code',-1)")
    if [ "$LC_CODE" = "0" ]; then
        SNOW_LIKES_COMMENT_ID=$(json_val "$LIKE_COMMENT_RESP" ".get('data',{}).get('snowLikesId','0')")
        pass "用户B 点赞C的评论成功, snowLikesId: $SNOW_LIKES_COMMENT_ID"
    else
        fail "用户B 点赞评论失败, code: $LC_CODE"
    fi
else
    skip "用户B 点赞评论（没有有效评论ID）"
fi

# ============ 17. 验证评论计数 =1（bugfix 回归测试：不应翻倍）============
separator "17. 验证推文评论计数 (contentService)"

if [ -n "$SNOW_TID" ] && [ "$SNOW_TID" != "0" ]; then
    sleep 2
    CC_RESP=$(curl $NO_PROXY -s "$BASE_CONTENT/getTweet?snowTid=$SNOW_TID" \
      -H "$UA_AUTH")

    echo "$CC_RESP" | python3 -m json.tool 2>/dev/null || echo "$CC_RESP"

    CC_CODE=$(json_val "$CC_RESP" ".get('code',-1)")
    COMMENT_COUNT=$(json_val "$CC_RESP" ".get('data',{}).get('commentCount',-1)")
    if [ "$CC_CODE" = "0" ]; then
        if [ "$COMMENT_COUNT" = "1" ]; then
            pass "评论计数正确: commentCount=$COMMENT_COUNT (1 条根评论，无翻倍)"
        else
            fail "评论计数异常: commentCount=$COMMENT_COUNT (expected 1, 旧 bug 可能复现)"
        fi
    else
        fail "验证评论计数失败, getTweet 返回 code: $CC_CODE"
    fi
else
    skip "验证评论计数（没有有效推文ID）"
fi

# ============ 18. 用户A 删除自己的推文 ============
separator "18. 用户A 删除自己的推文 (contentService)"

if [ -n "$SNOW_TID" ] && [ "$SNOW_TID" != "0" ]; then
    DELETE_RESP=$(curl $NO_PROXY -s -X DELETE "$BASE_CONTENT/deleteTweet?snowTid=$SNOW_TID" \
      -H "Content-Type: application/json" \
      -H "$UA_AUTH")

    echo "$DELETE_RESP" | python3 -m json.tool 2>/dev/null || echo "$DELETE_RESP"

    DEL_CODE=$(json_val "$DELETE_RESP" ".get('code',-1)")
    if [ "$DEL_CODE" = "0" ]; then
        pass "用户A 删除推文成功"
    else
        DEL_MSG=$(json_val "$DELETE_RESP" ".get('msg','')")
        fail "用户A 删除推文失败, code: $DEL_CODE, msg: $DEL_MSG"
    fi
else
    skip "用户A 删除推文（没有有效推文ID）"
fi

# ============ 等待 MQ 消费者异步写入 PostgreSQL ============
separator "等待异步写入... (MQ → PostgreSQL)"
echo "  Write-Behind 模式，等待 Kafka 消费者处理消息..."
sleep 8

# ============ 19. 用户B 获取点赞列表（验证异步持久化）============
separator "19. 用户B 获取点赞列表 (interactService)"

LIKES_ALL_RESP=$(curl $NO_PROXY -s "$BASE_INTERACT/getUserLikesAll?likesCursor=0" \
  -H "$UB_AUTH")

echo "$LIKES_ALL_RESP" | python3 -m json.tool 2>/dev/null || echo "$LIKES_ALL_RESP"

LA_CODE=$(json_val "$LIKES_ALL_RESP" ".get('code',-1)")
if [ "$LA_CODE" = "0" ]; then
    LA_TWEET_COUNT=$(json_val "$LIKES_ALL_RESP" ".get('likesForTweets',[]).__len__()")
    LA_COMMENT_COUNT=$(json_val "$LIKES_ALL_RESP" ".get('likesForComments',[]).__len__()")
    if [ "$LA_TWEET_COUNT" -ge 1 ] 2>/dev/null; then
        pass "用户B 点赞列表正常, 推文点赞: $LA_TWEET_COUNT, 评论点赞: $LA_COMMENT_COUNT"
    else
        fail "用户B 点赞列表为空（推文点赞: $LA_TWEET_COUNT），异步写入可能未完成"
    fi
else
    fail "获取用户B点赞失败, code: $LA_CODE"
fi

# ============ 20. 用户C 获取点赞列表 ============
separator "20. 用户C 获取点赞列表 (interactService)"

LIKES_C_RESP=$(curl $NO_PROXY -s "$BASE_INTERACT/getUserLikesAll?likesCursor=0" \
  -H "$UC_AUTH")

echo "$LIKES_C_RESP" | python3 -m json.tool 2>/dev/null || echo "$LIKES_C_RESP"

LC_ALL_CODE=$(json_val "$LIKES_C_RESP" ".get('code',-1)")
if [ "$LC_ALL_CODE" = "0" ]; then
    LC_TWEET_COUNT=$(json_val "$LIKES_C_RESP" ".get('likesForTweets',[]).__len__()")
    if [ "$LC_TWEET_COUNT" -ge 1 ] 2>/dev/null; then
        pass "用户C 点赞列表正常, 推文点赞: $LC_TWEET_COUNT"
    else
        fail "用户C 点赞列表为空（推文点赞: $LC_TWEET_COUNT），异步写入可能未完成"
    fi
else
    fail "获取用户C点赞失败, code: $LC_ALL_CODE"
fi

# ============ 21. 用户A 获取通知列表（验证通知产生）============
separator "21. 用户A 获取通知列表 (noticeService)"

NOTICE_RESP=$(curl $NO_PROXY -s "$BASE_NOTICE/getNotices?limit=10" \
  -H "$UA_AUTH")

echo "$NOTICE_RESP" | python3 -m json.tool 2>/dev/null || echo "$NOTICE_RESP"

NOTICE_CODE=$(json_val "$NOTICE_RESP" ".get('code',-1)")
if [ "$NOTICE_CODE" = "0" ]; then
    LIKE_NOTICES=$(json_val "$NOTICE_RESP" ".get('likeNotices',[]).__len__()")
    COMMENT_NOTICES=$(json_val "$NOTICE_RESP" ".get('commentNotices',[]).__len__()")
    UNREAD=$(json_val "$NOTICE_RESP" ".get('unreadCount',0)")
    TOTAL_NOTICES=$((LIKE_NOTICES + COMMENT_NOTICES))
    if [ "$TOTAL_NOTICES" -ge 1 ] 2>/dev/null; then
        pass "用户A 通知列表正常, 点赞通知: $LIKE_NOTICES, 评论通知: $COMMENT_NOTICES, 未读: $UNREAD"
    else
        fail "用户A 通知列表为空（点赞: $LIKE_NOTICES, 评论: $COMMENT_NOTICES），异步写入可能未完成"
    fi
else
    fail "获取通知列表失败, code: $NOTICE_CODE"
fi

# ============ 22. 用户B 获取通知列表 ============
separator "22. 用户B 获取通知列表 (noticeService)"

NOTICE_B_RESP=$(curl $NO_PROXY -s "$BASE_NOTICE/getNotices?limit=10" \
  -H "$UB_AUTH")

echo "$NOTICE_B_RESP" | python3 -m json.tool 2>/dev/null || echo "$NOTICE_B_RESP"

NOTICE_B_CODE=$(json_val "$NOTICE_B_RESP" ".get('code',-1)")
if [ "$NOTICE_B_CODE" = "0" ]; then
    LIKE_B_NOTICES=$(json_val "$NOTICE_B_RESP" ".get('likeNotices',[]).__len__()")
    COMMENT_B_NOTICES=$(json_val "$NOTICE_B_RESP" ".get('commentNotices',[]).__len__()")
    UNREAD_B=$(json_val "$NOTICE_B_RESP" ".get('unreadCount',0)")
    # B 被 C 的评论点赞通知（步骤15中B点赞C的评论不会通知B自己，但C回复B的评论会通知B）
    TOTAL_B_NOTICES=$((LIKE_B_NOTICES + COMMENT_B_NOTICES))
    if [ "$TOTAL_B_NOTICES" -ge 1 ] 2>/dev/null; then
        pass "用户B 通知列表正常, 点赞通知: $LIKE_B_NOTICES, 评论通知: $COMMENT_B_NOTICES, 未读: $UNREAD_B"
    else
        fail "用户B 通知列表为空（点赞: $LIKE_B_NOTICES, 评论: $COMMENT_B_NOTICES），异步写入可能未完成"
    fi
else
    fail "获取用户B通知列表失败, code: $NOTICE_B_CODE"
fi

# ============ 23. 用户A 获取未读数 ============
separator "23. 用户A 获取未读数 (noticeService)"

UNREAD_RESP=$(curl $NO_PROXY -s "$BASE_NOTICE/getUnreadCount" \
  -H "$UA_AUTH")

echo "$UNREAD_RESP" | python3 -m json.tool 2>/dev/null || echo "$UNREAD_RESP"

UNREAD_CODE=$(json_val "$UNREAD_RESP" ".get('code',-1)")
if [ "$UNREAD_CODE" = "0" ]; then
    LIKE_UNREAD=$(json_val "$UNREAD_RESP" ".get('likeUnread',0)")
    COMMENT_UNREAD=$(json_val "$UNREAD_RESP" ".get('commentUnread',0)")
    TOTAL_UNREAD=$(json_val "$UNREAD_RESP" ".get('totalUnread',0)")
    if [ "$TOTAL_UNREAD" -ge 1 ] 2>/dev/null; then
        pass "用户A 未读数正常, like: $LIKE_UNREAD, comment: $COMMENT_UNREAD, total: $TOTAL_UNREAD"
    else
        fail "用户A 未读数为0，通知可能未持久化"
    fi
else
    fail "获取未读数失败, code: $UNREAD_CODE"
fi

# ============ 24. 用户A 标记全部已读 ============
separator "24. 用户A 标记全部已读 (noticeService)"

MARK_RESP=$(curl $NO_PROXY -s -X POST "$BASE_NOTICE/markRead" \
  -H "Content-Type: application/json" \
  -H "$UA_AUTH" \
  -d '{"noticeType":0}')

echo "$MARK_RESP" | python3 -m json.tool 2>/dev/null || echo "$MARK_RESP"

MARK_CODE=$(json_val "$MARK_RESP" ".get('code',-1)")
if [ "$MARK_CODE" = "0" ]; then
    pass "标记已读成功"
else
    fail "标记已读失败, code: $MARK_CODE"
fi

# ============ 25. 验证已读后未读数为0 ============
separator "25. 用户A 验证已读后未读数 (noticeService)"

UNREAD2_RESP=$(curl $NO_PROXY -s "$BASE_NOTICE/getUnreadCount" \
  -H "$UA_AUTH")

echo "$UNREAD2_RESP" | python3 -m json.tool 2>/dev/null || echo "$UNREAD2_RESP"

UNREAD2_CODE=$(json_val "$UNREAD2_RESP" ".get('code',-1)")
if [ "$UNREAD2_CODE" = "0" ]; then
    TOTAL_UNREAD2=$(json_val "$UNREAD2_RESP" ".get('totalUnread',0)")
    if [ "$TOTAL_UNREAD2" = "0" ] 2>/dev/null; then
        pass "标记已读后未读数为0，验证通过"
    else
        fail "标记已读后未读数仍为 $TOTAL_UNREAD2"
    fi
else
    fail "验证已读失败, code: $UNREAD2_CODE"
fi

# ============ 26. 推荐 Feed ============
separator "26. 推荐 Feed (recommendService)"

FEED_RESP=$(curl $NO_PROXY -s "$BASE_RECOMMEND/feed?limit=5" \
  -H "$UA_AUTH")

echo "$FEED_RESP" | python3 -m json.tool 2>/dev/null || echo "$FEED_RESP"

FEED_CODE=$(json_val "$FEED_RESP" ".get('code',-1)")
if [ "$FEED_CODE" = "0" ]; then
    FEED_COUNT=$(json_val "$FEED_RESP" ".get('data',[]).__len__()")
    pass "推荐 Feed 成功, 返回 $FEED_COUNT 条推文"
else
    FEED_MSG=$(json_val "$FEED_RESP" ".get('msg','')")
    fail "推荐 Feed 失败, code: $FEED_CODE, msg: $FEED_MSG"
fi

# ============ 27. Python 召回直连测试 ============
separator "27. Python 召回服务直连测试"

RECALL_RESP=$(curl $NO_PROXY -s -X POST "http://127.0.0.1:2006/api/v1/recall" \
  -H "Content-Type: application/json" \
  -d "{\"uid\":$UA_UID,\"limit\":5}" 2>/dev/null)

if [ -n "$RECALL_RESP" ] && [ "$RECALL_RESP" != "" ]; then
    echo "$RECALL_RESP" | python3 -m json.tool 2>/dev/null || echo "$RECALL_RESP"

    RECALL_CODE=$(json_val "$RECALL_RESP" ".get('code',-1)")
    if [ "$RECALL_CODE" = "0" ]; then
        TWEET_IDS=$(json_val "$RECALL_RESP" ".get('data',{}).get('tweet_ids',[]).__len__()")
        pass "Python 召回成功, 返回 $TWEET_IDS 个推文ID"
    else
        fail "Python 召回失败, code: $RECALL_CODE"
    fi
else
    skip "Python 召回服务未响应 (需要运行在 :2006)"
fi

# ============ 汇总 ============
separator "测试汇总"
TOTAL=$((PASS_COUNT + FAIL_COUNT + SKIP_COUNT))
echo -e "  总计: $TOTAL"
echo -e "  ${GREEN}PASS: $PASS_COUNT${NC}  ${RED}FAIL: $FAIL_COUNT${NC}  ${YELLOW}SKIP: $SKIP_COUNT${NC}"
echo ""
echo "测试用户:"
echo "  - 用户A (推文作者): uid=$UA_UID"
echo "  - 用户B (点赞+评论): uid=$UB_UID"
echo "  - 用户C (点赞+回复+评论点赞): uid=$UC_UID"
echo ""
echo "交互流程:"
echo "  - B和C 点赞 A的推文 → A 收到点赞通知"
echo "  - B 评论 A的推文 → A 收到评论通知"
echo "  - C 回复 B的评论 → B 收到回复通知"
echo "  - B 点赞 C的评论 → C 收到评论点赞通知"
echo "  - A 删除推文"
echo "  - 等待8秒后验证异步持久化数据"
