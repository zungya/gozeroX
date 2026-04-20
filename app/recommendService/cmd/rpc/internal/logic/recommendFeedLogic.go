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
	Uid   int64 `json:"uid"`
	Limit int64 `json:"limit"`
}

type PythonRecallResp struct {
	Code int64  `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		TweetIds []int64 `json:"tweet_ids"`
	} `json:"data"`
}

// RecommendFeed 推荐首页 Feed
// 缓存策略：使用 Redis List 维护用户预取推文 ID 队列
// 1. 队列够用 → 多取候选（2*limit），过滤已删除后取 limit 个有效推文
// 2. 队列不够 → 调 Python 召回 5*limit，合并后过滤，多余有效 ID 存回队列
func (l *RecommendFeedLogic) RecommendFeed(in *pb.RecommendFeedReq) (*pb.RecommendFeedResp, error) {
	cacheKey := fmt.Sprintf("%d", in.Uid)

	// 1. 从 Redis List 获取预存的推文 ID
	cachedIds, err := l.svcCtx.CacheManager.LRange(l.ctx, "recommend", "feed", cacheKey, 0, -1)
	if err != nil {
		l.Errorf("RecommendFeed LRange error, uid:%d, err:%v", in.Uid, err)
		cachedIds = []int64{}
	}

	// 2. 收集候选 ID（多取以补偿已删除推文）
	var candidateIds []int64
	consumedCount := int64(0) // 从缓存消耗的 ID 数量
	calledPython := false

	if int64(len(cachedIds)) >= in.Limit {
		// 缓存够用：多取候选，至少 2*limit（上限为缓存总量）
		fetchCount := in.Limit * 2
		if fetchCount > int64(len(cachedIds)) {
			fetchCount = int64(len(cachedIds))
		}
		candidateIds = cachedIds[:fetchCount]
		consumedCount = fetchCount
	} else {
		// 缓存不够：调 Python 召回 5*limit
		fetchLimit := in.Limit * 5
		recallResp, recallErr := l.callPythonRecall(in.Uid, fetchLimit)
		if recallErr != nil {
			l.Errorf("RecommendFeed callPythonRecall error, uid:%d, err:%v", in.Uid, recallErr)
			// 降级：尝试用缓存中的 ID
			if len(cachedIds) == 0 {
				return &pb.RecommendFeedResp{
					Code: errorx.ErrCodeRecommendServiceUnavailable,
					Msg:  errorx.GetMsg(errorx.ErrCodeRecommendServiceUnavailable),
				}, nil
			}
			candidateIds = cachedIds
			consumedCount = int64(len(cachedIds))
		} else {
			// 清空旧缓存
			_ = l.svcCtx.CacheManager.Del(l.ctx, "recommend", "feed", cacheKey)
			// 合并：缓存 + Python 召回
			candidateIds = make([]int64, 0, len(cachedIds)+len(recallResp.Data.TweetIds))
			candidateIds = append(candidateIds, cachedIds...)
			candidateIds = append(candidateIds, recallResp.Data.TweetIds...)
			consumedCount = int64(len(cachedIds)) // 缓存全部消耗
			calledPython = true
		}
	}

	// 3. 消费缓存中已使用的 ID
	if consumedCount > 0 && !calledPython {
		if err := l.svcCtx.CacheManager.LTrim(l.ctx, "recommend", "feed", cacheKey, consumedCount, -1); err != nil {
			l.Errorf("RecommendFeed LTrim error, uid:%d, err:%v", in.Uid, err)
		}
	}

	// 4. 空候选
	if len(candidateIds) == 0 {
		return &pb.RecommendFeedResp{Code: 0, Msg: "success"}, nil
	}

	// 5. 批量获取推文详情（BatchGetTweets 会过滤掉已删除/非公开的）
	tweetMap, err := l.batchGetTweetDetails(candidateIds)
	if err != nil {
		l.Errorf("RecommendFeed batchGetTweetDetails error, err:%v", err)
		return &pb.RecommendFeedResp{
			Code: errorx.ErrCodeRPCError,
			Msg:  errorx.GetMsg(errorx.ErrCodeRPCError),
		}, nil
	}

	// 6. 按候选顺序构建推文列表，取前 limit 个有效的
	tweets := l.buildTweetInfoList(candidateIds, tweetMap, int(in.Limit))

	// 7. 将未使用的有效 ID 存回缓存（仅 Python 召回路径需要）
	if calledPython {
		var extraValidIds []int64
		returnedCount := 0
		for _, tid := range candidateIds {
			if _, ok := tweetMap[tid]; ok {
				returnedCount++
				if returnedCount > int(in.Limit) {
					extraValidIds = append(extraValidIds, tid)
				}
			}
		}
		if len(extraValidIds) > 0 {
			go l.cacheExtraIds(context.Background(), in.Uid, extraValidIds)
		}
	}

	return &pb.RecommendFeedResp{
		Code:   0,
		Msg:    "success",
		Tweets: tweets,
	}, nil
}

// cacheExtraIds 异步将多余的推文 ID 存入 Redis List
func (l *RecommendFeedLogic) cacheExtraIds(ctx context.Context, uid int64, ids []int64) {
	cacheKey := fmt.Sprintf("%d", uid)
	if err := l.svcCtx.CacheManager.RPush(ctx, "recommend", "feed", cacheKey, ids...); err != nil {
		l.Errorf("cacheExtraIds RPush error, uid:%d, err:%v", uid, err)
		return
	}
	// 设置 5 分钟过期
	_ = l.svcCtx.CacheManager.Expire(ctx, "recommend", "feed", cacheKey, 300)
}

// callPythonRecall 调用 Python 推荐召回接口
func (l *RecommendFeedLogic) callPythonRecall(uid, limit int64) (*PythonRecallResp, error) {
	reqBody := PythonRecallReq{
		Uid:   uid,
		Limit: limit,
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

// buildTweetInfoList 按召回顺序构建 TweetInfo 列表，最多取 limit 个有效推文
func (l *RecommendFeedLogic) buildTweetInfoList(tweetIds []int64, tweetMap map[int64]*content.Tweet, limit int) []*pb.TweetInfo {
	var tweets []*pb.TweetInfo
	for _, tid := range tweetIds {
		if len(tweets) >= limit {
			break
		}
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
