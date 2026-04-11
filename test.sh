#!/bin/bash
# gozeroX 本地测试脚本
# 使用方法: bash test.sh
# 前提: docker compose -f docker-compose-env.yml up -d 已启动，所有 Go 服务已启动

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
info() { echo -e "${YELLOW}[INFO]${NC} $1"; SKIP_COUNT=$((SKIP_COUNT+1)); }

separator() { echo ""; echo "============================================"; echo "$1"; echo "============================================"; }

# 提取 JSON 字段的辅助函数
json_val() { echo "$1" | python3 -c "import sys,json; print(json.load(sys.stdin)$2)" 2>/dev/null; }

# ============ 1. 注册 ============
separator "1. 用户注册"

MOBILE="1870018700$((RANDOM % 100))"
REGISTER_RESP=$(curl $NO_PROXY -s -w "\n%{http_code}" -X POST "$BASE_USER/user/register" \
  -H "Content-Type: application/json" \
  -d "{\"mobile\":\"$MOBILE\",\"password\":\"test1234\",\"nickname\":\"tester\"}")

HTTP_CODE=$(echo "$REGISTER_RESP" | tail -1)
BODY=$(echo "$REGISTER_RESP" | sed '$d')

if [ "$HTTP_CODE" = "200" ]; then
    CODE=$(json_val "$BODY" ".get('code',-1)")
    if [ "$CODE" = "0" ]; then
        pass "注册成功 (手机号: $MOBILE)"
    else
        MSG=$(json_val "$BODY" ".get('message','')")
        fail "注册失败, code: $CODE, msg: $MSG (已知bug: PostgreSQL LastInsertId)"
    fi
else
    fail "HTTP $HTTP_CODE"
fi

# ============ 2. 登录 ============
separator "2. 用户登录"

LOGIN_RESP=$(curl $NO_PROXY -s -X POST "$BASE_USER/user/login" \
  -H "Content-Type: application/json" \
  -d '{"mobile":"13800138000","password":"test1234"}')

echo "$LOGIN_RESP" | python3 -m json.tool 2>/dev/null || echo "$LOGIN_RESP"

TOKEN=$(json_val "$LOGIN_RESP" ".get('accessToken','')")

if [ -n "$TOKEN" ] && [ "$TOKEN" != "" ] && [ "$TOKEN" != "None" ]; then
    pass "登录成功，获取到 Token"
    info "Token: ${TOKEN:0:40}..."
else
    fail "登录失败，无法获取 Token，后续测试跳过"
    exit 1
fi

AUTH_HEADER="Authorization: Bearer $TOKEN"

# ============ 3. 获取用户信息 ============
separator "3. 获取用户信息"

DETAIL_RESP=$(curl $NO_PROXY -s -X POST "$BASE_USER/user/detail" \
  -H "Content-Type: application/json" \
  -H "$AUTH_HEADER" \
  -d '{"uid":1}')

echo "$DETAIL_RESP" | python3 -m json.tool 2>/dev/null || echo "$DETAIL_RESP"

UID=$(json_val "$DETAIL_RESP" ".get('userInfo',{}).get('uid',0)")
if [ "$UID" -gt 0 ] 2>/dev/null; then
    pass "获取用户信息成功, uid: $UID"
else
    fail "获取用户信息失败"
fi

# ============ 4. 发推文 ============
separator "4. 发推文"

TWEET_RESP=$(curl $NO_PROXY -s -X POST "$BASE_CONTENT/createTweet" \
  -H "Content-Type: application/json" \
  -H "$AUTH_HEADER" \
  -d '{"content":"Hello from local test!","mediaUrls":[],"tags":["test"],"isPublic":true}')

echo "$TWEET_RESP" | python3 -m json.tool 2>/dev/null || echo "$TWEET_RESP"

TWEET_CODE=$(json_val "$TWEET_RESP" ".get('code',-1)")
SNOW_TID=$(json_val "$TWEET_RESP" ".get('data',{}).get('snowTid',0)")

if [ "$TWEET_CODE" = "0" ] && [ "$SNOW_TID" != "0" ] && [ -n "$SNOW_TID" ]; then
    pass "发推文成功, snowTid: $SNOW_TID"
else
    MSG=$(json_val "$TWEET_RESP" ".get('msg','')")
    fail "发推文失败, code: $TWEET_CODE, msg: $MSG"
    info "如果提示'生成推文ID失败'，检查 pkg/idgen 雪花算法是否正确初始化"
fi

# ============ 5. 推文列表 ============
separator "5. 推文列表"

LIST_RESP=$(curl $NO_PROXY -s "$BASE_CONTENT/listTweets?queryUid=1&limit=5" \
  -H "$AUTH_HEADER")

echo "$LIST_RESP" | python3 -m json.tool 2>/dev/null || echo "$LIST_RESP"

LIST_CODE=$(json_val "$LIST_RESP" ".get('code',-1)")
if [ "$LIST_CODE" = "0" ]; then
    TOTAL=$(json_val "$LIST_RESP" ".get('total',0)")
    pass "推文列表查询成功, 共 $TOTAL 条"
else
    fail "推文列表查询失败"
fi

# ============ 6. 点赞推文 ============
separator "6. 点赞推文 (interactService)"

if [ -n "$SNOW_TID" ] && [ "$SNOW_TID" != "0" ]; then
    LIKE_RESP=$(curl $NO_PROXY -s -X POST "$BASE_INTERACT/like" \
      -H "Content-Type: application/json" \
      -H "$AUTH_HEADER" \
      -d "{\"isCreated\":0,\"snowLikesId\":\"0\",\"targetType\":0,\"targetId\":\"$SNOW_TID\",\"status\":1}")

    echo "$LIKE_RESP" | python3 -m json.tool 2>/dev/null || echo "$LIKE_RESP"

    LIKE_CODE=$(json_val "$LIKE_RESP" ".get('code',-1)")
    if [ "$LIKE_CODE" = "0" ]; then
        pass "点赞成功"
    else
        fail "点赞失败, code: $LIKE_CODE"
    fi
else
    info "跳过点赞测试（没有有效推文ID）"
fi

# ============ 7. 发表评论 ============
separator "7. 发表评论 (interactService)"

if [ -n "$SNOW_TID" ] && [ "$SNOW_TID" != "0" ]; then
    COMMENT_RESP=$(curl $NO_PROXY -s -X POST "$BASE_INTERACT/createComment" \
      -H "Content-Type: application/json" \
      -H "$AUTH_HEADER" \
      -d "{\"snowTid\":\"$SNOW_TID\",\"content\":\"Nice post!\",\"parentId\":\"0\",\"rootId\":\"0\"}")

    echo "$COMMENT_RESP" | python3 -m json.tool 2>/dev/null || echo "$COMMENT_RESP"

    COMMENT_CODE=$(json_val "$COMMENT_RESP" ".get('code',-1)")
    if [ "$COMMENT_CODE" = "0" ]; then
        pass "评论成功"
    else
        fail "评论失败, code: $COMMENT_CODE"
    fi
else
    info "跳过评论测试（没有有效推文ID）"
fi

# ============ 8. 获取通知 ============
separator "8. 获取通知列表"

NOTICE_RESP=$(curl $NO_PROXY -s "$BASE_NOTICE/getNotices?limit=10" \
  -H "$AUTH_HEADER")

echo "$NOTICE_RESP" | python3 -m json.tool 2>/dev/null || echo "$NOTICE_RESP"

NOTICE_CODE=$(json_val "$NOTICE_RESP" ".get('code',-1)")
if [ "$NOTICE_CODE" = "0" ]; then
    pass "获取通知列表成功"
else
    fail "获取通知失败, code: $NOTICE_CODE"
fi

# ============ 9. 获取未读数 ============
separator "9. 获取未读数"

UNREAD_RESP=$(curl $NO_PROXY -s "$BASE_NOTICE/getUnreadCount" \
  -H "$AUTH_HEADER")

echo "$UNREAD_RESP" | python3 -m json.tool 2>/dev/null || echo "$UNREAD_RESP"

UNREAD_CODE=$(json_val "$UNREAD_RESP" ".get('code',-1)")
if [ "$UNREAD_CODE" = "0" ]; then
    TOTAL_UNREAD=$(json_val "$UNREAD_RESP" ".get('totalUnread',0)")
    pass "获取未读数成功, totalUnread: $TOTAL_UNREAD"
else
    fail "获取未读数失败, code: $UNREAD_CODE"
fi

# ============ 10. 标记已读 ============
separator "10. 标记已读"

MARK_RESP=$(curl $NO_PROXY -s -X POST "$BASE_NOTICE/markRead" \
  -H "Content-Type: application/json" \
  -H "$AUTH_HEADER" \
  -d '{"noticeType":0}')

echo "$MARK_RESP" | python3 -m json.tool 2>/dev/null || echo "$MARK_RESP"

MARK_CODE=$(json_val "$MARK_RESP" ".get('code',-1)")
if [ "$MARK_CODE" = "0" ]; then
    pass "标记已读成功"
else
    fail "标记已读失败, code: $MARK_CODE"
fi

# ============ 11. 推荐Feed ============
separator "11. 推荐Feed (recommendService + Python召回)"

FEED_RESP=$(curl $NO_PROXY -s "http://localhost:1005/recommendService/v1/feed?limit=5" \
  -H "$AUTH_HEADER")

echo "$FEED_RESP" | python3 -m json.tool 2>/dev/null || echo "$FEED_RESP"

FEED_CODE=$(json_val "$FEED_RESP" ".get('code',-1)")
if [ "$FEED_CODE" = "0" ]; then
    FEED_COUNT=$(json_val "$FEED_RESP" ".get('data',[]).__len__()")
    HAS_MORE=$(json_val "$FEED_RESP" ".get('hasMore',False)")
    pass "推荐Feed成功, 返回 $FEED_COUNT 条推文, hasMore: $HAS_MORE"
else
    FEED_MSG=$(json_val "$FEED_RESP" ".get('msg','')")
    fail "推荐Feed失败, code: $FEED_CODE, msg: $FEED_MSG"
fi

# ============ 12. Python召回直连测试 ============
separator "12. Python召回服务直连测试"

RECALL_RESP=$(curl $NO_PROXY -s -X POST "http://127.0.0.1:2006/api/v1/recall" \
  -H "Content-Type: application/json" \
  -d '{"uid":1,"limit":5,"cursor":0}')

echo "$RECALL_RESP" | python3 -m json.tool 2>/dev/null || echo "$RECALL_RESP"

RECALL_CODE=$(json_val "$RECALL_RESP" ".get('code',-1)")
if [ "$RECALL_CODE" = "0" ]; then
    TWEET_IDS=$(json_val "$RECALL_RESP" ".get('data',{}).get('tweet_ids',[]).__len__()")
    pass "Python召回成功, 返回 $TWEET_IDS 个推文ID"
else
    fail "Python召回失败, code: $RECALL_CODE"
fi

# ============ 汇总 ============
separator "测试汇总"
echo -e "  ${GREEN}PASS: $PASS_COUNT${NC}  ${RED}FAIL: $FAIL_COUNT${NC}  ${YELLOW}SKIP: $SKIP_COUNT${NC}"
echo ""
echo "说明:"
echo "  - 通知相关: 需要 MQ 消费者运行才能产生通知数据，否则为空"
echo "  - 推荐 Feed: 需要 Python 召回服务运行在 2006 端口"
