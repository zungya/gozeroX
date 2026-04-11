package logic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"gozeroX/pkg/errorx"
	"io"
	"net/http"
	"time"

	"gozeroX/app/contentService/cmd/rpc/content"
	"gozeroX/app/recommendService/cmd/rpc/internal/svc"
	"gozeroX/app/recommendService/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type RecommendFeedLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRecommendFeedLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RecommendFeedLogic {
	return &RecommendFeedLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// Python recall 接口的请求/响应结构
type PythonRecallReq struct {
	Uid    int64 `json:"uid"`
	Limit  int64 `json:"limit"`
	Cursor int64 `json:"cursor"`
}

type PythonRecallResp struct {
	Code int64  `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		TweetIds []int64 `json:"tweet_ids"`
		Cursor   int64   `json:"cursor"`
		HasMore  bool    `json:"has_more"`
	} `json:"data"`
}

// RecommendFeed 推荐首页 Feed
func (l *RecommendFeedLogic) RecommendFeed(in *pb.RecommendFeedReq) (*pb.RecommendFeedResp, error) {
	// 1. 从缓存获取推荐结果（一次缓存24个，前端每次请求8个）
	cacheTweets, cacheCursor, cacheHasMore, err := l.getRecommendFromCache(in.Uid, in.Cursor, in.Limit)
	if err == nil && len(cacheTweets) > 0 {
		return &pb.RecommendFeedResp{
			Code:    0,
			Msg:     "success",
			Tweets:  cacheTweets,
			Cursor:  cacheCursor,
			HasMore: cacheHasMore,
		}, nil
	}

	// 2. 缓存未命中，调用 Python recall 接口（请求3倍数量用于缓存）
	fetchLimit := in.Limit * 3
	if fetchLimit < 24 {
		fetchLimit = 24
	}
	recallResp, err := l.callPythonRecall(in.Uid, fetchLimit, in.Cursor)
	if err != nil {
		logx.Errorf("RecommendFeed callPythonRecall error, uid:%d, err:%v", in.Uid, err)
		return &pb.RecommendFeedResp{
			Code: errorx.ErrCodeRecommendServiceUnavailable,
			Msg:  errorx.GetMsg(errorx.ErrCodeRecommendServiceUnavailable),
		}, nil
	}

	// 3. Python 返回空结果（冷启动或无推荐）
	if len(recallResp.Data.TweetIds) == 0 {
		return &pb.RecommendFeedResp{
			Code:    0,
			Msg:     "success",
			Tweets:  nil,
			Cursor:  0,
			HasMore: false,
		}, nil
	}

	// 4. 调用 contentService RPC 批量获取推文详情
	tweetDetails, err := l.batchGetTweetDetails(recallResp.Data.TweetIds)
	if err != nil {
		logx.Errorf("RecommendFeed batchGetTweetDetails error, err:%v", err)
		return &pb.RecommendFeedResp{
			Code: errorx.ErrCodeRPCError,
			Msg:  errorx.GetMsg(errorx.ErrCodeRPCError),
		}, nil
	}

	// 5. 按召回顺序组装结果（Python 返回的顺序就是推荐排序）
	tweets := l.buildTweetInfoList(recallResp.Data.TweetIds, tweetDetails)

	// 6. 缓存推荐结果
	go l.cacheRecommendResult(context.Background(), in.Uid, in.Cursor, tweets, recallResp.Data.Cursor, recallResp.Data.HasMore)

	// 7. 截取用户请求的数量
	returnLimit := in.Limit
	if int(returnLimit) > len(tweets) {
		returnLimit = int64(len(tweets))
	}

	return &pb.RecommendFeedResp{
		Code:    0,
		Msg:     "success",
		Tweets:  tweets[:returnLimit],
		Cursor:  recallResp.Data.Cursor,
		HasMore: recallResp.Data.HasMore,
	}, nil
}

// callPythonRecall 调用 Python 推荐召回接口
func (l *RecommendFeedLogic) callPythonRecall(uid, limit, cursor int64) (*PythonRecallResp, error) {
	reqBody := PythonRecallReq{
		Uid:    uid,
		Limit:  limit,
		Cursor: cursor,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal recall request failed: %w", err)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Post(
		l.svcCtx.Config.PythonRecommend.RecallUrl,
		"application/json",
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return nil, fmt.Errorf("call python recall failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read recall response failed: %w", err)
	}

	var recallResp PythonRecallResp
	if err := json.Unmarshal(body, &recallResp); err != nil {
		return nil, fmt.Errorf("unmarshal recall response failed: %w", err)
	}

	if recallResp.Code != 0 {
		return nil, fmt.Errorf("python recall returned error: %s", recallResp.Msg)
	}

	return &recallResp, nil
}

// batchGetTweetDetails 批量获取推文详情
func (l *RecommendFeedLogic) batchGetTweetDetails(tweetIds []int64) (map[int64]*content.Tweet, error) {
	resp, err := l.svcCtx.ContentServiceRpc.BatchGetTweets(l.ctx, &content.BatchGetTweetsReq{
		SnowTids: tweetIds,
	})
	if err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		return nil, fmt.Errorf("BatchGetTweets returned code:%d, msg:%s", resp.Code, resp.Msg)
	}

	// 构建以 snow_tid 为 key 的 map，方便后续按顺序查找
	tweetMap := make(map[int64]*content.Tweet, len(resp.Tweets))
	for _, t := range resp.Tweets {
		tweetMap[t.SnowTid] = t
	}
	return tweetMap, nil
}

// buildTweetInfoList 按召回顺序构建 TweetInfo 列表
func (l *RecommendFeedLogic) buildTweetInfoList(tweetIds []int64, tweetMap map[int64]*content.Tweet) []*pb.TweetInfo {
	var tweets []*pb.TweetInfo
	for _, tid := range tweetIds {
		t, ok := tweetMap[tid]
		if !ok {
			continue // 推文可能已被删除，跳过
		}
		tweets = append(tweets, &pb.TweetInfo{
			SnowTid:      t.SnowTid,
			Uid:          t.Uid,
			Content:      t.Content,
			MediaUrls:    t.MediaUrls,
			Tags:         t.Tags,
			LikeCount:    t.LikeCount,
			CommentCount: t.CommentCount,
			CreatedAt:    t.CreatedAt,
			Nickname:     t.Nickname,
			Avatar:       t.Avatar,
		})
	}
	return tweets
}

// getRecommendFromCache 从 Redis 缓存获取推荐结果
func (l *RecommendFeedLogic) getRecommendFromCache(uid, cursor, limit int64) ([]*pb.TweetInfo, int64, bool, error) {
	// 缓存 key: recommend:feed:{uid}:{cursor}
	cacheKey := fmt.Sprintf("%d:%d", uid, cursor)

	// 从缓存读取推荐的 tweet_ids
	snowIds, err := l.svcCtx.CacheManager.SMembers(l.ctx, "recommend", "feed", cacheKey)
	if err != nil || len(snowIds) == 0 {
		return nil, 0, false, fmt.Errorf("cache miss")
	}

	// 获取缓存的 cursor 和 has_more
	fields, err := l.svcCtx.CacheManager.HGetAll(l.ctx, "recommend", "feed_meta", cacheKey)
	if err != nil || len(fields) == 0 {
		return nil, 0, false, fmt.Errorf("cache meta miss")
	}

	// SMembers 返回的就是 []int64，直接使用
	cachedIds := snowIds

	if len(cachedIds) == 0 {
		return nil, 0, false, fmt.Errorf("empty cached ids")
	}

	// 批量获取推文详情
	tweetMap, err := l.batchGetTweetDetails(cachedIds)
	if err != nil {
		return nil, 0, false, err
	}

	tweets := l.buildTweetInfoList(cachedIds, tweetMap)

	// 截取请求的数量
	returnLimit := int(limit)
	if returnLimit > len(tweets) {
		returnLimit = len(tweets)
	}

	var nextCursor int64
	var hasMore bool
	if v, ok := fields["cursor"]; ok {
		fmt.Sscanf(v, "%d", &nextCursor)
	}
	if v, ok := fields["has_more"]; ok {
		hasMore = v == "1"
	}

	return tweets[:returnLimit], nextCursor, hasMore, nil
}

// cacheRecommendResult 缓存推荐结果到 Redis
func (l *RecommendFeedLogic) cacheRecommendResult(ctx context.Context, uid, cursor int64, tweets []*pb.TweetInfo, nextCursor int64, hasMore bool) {
	cacheKey := fmt.Sprintf("%d:%d", uid, cursor)

	// 缓存推文 ID 列表
	var ids []int64
	for _, t := range tweets {
		ids = append(ids, t.SnowTid)
	}
	if len(ids) > 0 {
		if err := l.svcCtx.CacheManager.SAdd(ctx, "recommend", "feed", cacheKey, ids...); err != nil {
			logx.Errorf("cacheRecommendResult SAdd error, key:%s, err:%v", cacheKey, err)
			return
		}
		// 缓存 60 秒
		if err := l.svcCtx.CacheManager.Expire(ctx, "recommend", "feed", cacheKey, 60); err != nil {
			logx.Errorf("cacheRecommendResult Expire error, key:%s, err:%v", cacheKey, err)
		}
	}

	// 缓存 meta 信息（cursor 和 has_more）
	hasMoreStr := "0"
	if hasMore {
		hasMoreStr = "1"
	}
	metaFields := map[string]interface{}{
		"cursor":   fmt.Sprintf("%d", nextCursor),
		"has_more": hasMoreStr,
	}
	if err := l.svcCtx.CacheManager.HSetAll(ctx, "recommend", "feed_meta", cacheKey, metaFields); err != nil {
		logx.Errorf("cacheRecommendResult HSetAll error, key:%s, err:%v", cacheKey, err)
	}
	if err := l.svcCtx.CacheManager.Expire(ctx, "recommend", "feed_meta", cacheKey, 60); err != nil {
		logx.Errorf("cacheRecommendResult Expire meta error, key:%s, err:%v", cacheKey, err)
	}
}
