package logic

import (
	"context"
	"gozeroX/app/contentService/model"
	"sync"

	"gozeroX/app/contentService/cmd/rpc/internal/svc"
	"gozeroX/app/contentService/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type BatchGetTweetsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewBatchGetTweetsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BatchGetTweetsLogic {
	return &BatchGetTweetsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// BatchGetTweets 3. 批量推文查询（仅返回公开的）
func (l *BatchGetTweetsLogic) BatchGetTweets(in *pb.BatchGetTweetsReq) (*pb.BatchGetTweetsResp, error) {
	// 1. 参数校验
	if len(in.Tids) == 0 {
		return &pb.BatchGetTweetsResp{
			Code:   0,
			Msg:    "success",
			Tweets: []*pb.Tweet{},
		}, nil
	}

	// 2. 去重
	tidSet := make(map[int64]struct{})
	for _, tid := range in.Tids {
		tidSet[tid] = struct{}{}
	}
	uniqueTids := make([]int64, 0, len(tidSet))
	for tid := range tidSet {
		uniqueTids = append(uniqueTids, tid)
	}

	// 3. 批量从缓存获取（适配拆分后的三类缓存）
	cachedTweets := make(map[int64]*model.Tweet)
	missTids := make([]int64, 0)

	var mu sync.Mutex
	var wg sync.WaitGroup

	// 并发从缓存获取
	for _, tid := range uniqueTids {
		wg.Add(1)
		go func(tid int64) {
			defer wg.Done()

			// 调用svc层的合并缓存方法
			tweet, err := l.svcCtx.GetTweetFromCache(l.ctx, tid)
			if err == nil {
				// 缓存命中，且只返回公开且未删除的推文
				if tweet.IsPublic && !tweet.IsDeleted {
					mu.Lock()
					cachedTweets[tid] = tweet
					mu.Unlock()
				}
			} else {
				mu.Lock()
				missTids = append(missTids, tid)
				mu.Unlock()
			}
		}(tid)
	}

	wg.Wait()

	logx.Infof("批量查询推文: 总请求=%d, 唯一TID=%d, 缓存命中=%d, 未命中=%d",
		len(in.Tids), len(uniqueTids), len(cachedTweets), len(missTids))

	// 4. 如果没有缓存未命中的，直接返回
	if len(missTids) == 0 {
		return &pb.BatchGetTweetsResp{
			Code:   0,
			Msg:    "success",
			Tweets: l.buildTweetsFromMap(cachedTweets),
		}, nil
	}

	// 5. 批量查询数据库（未命中的）
	dbTweets, err := l.svcCtx.TweetModel.FindBatchByTids(l.ctx, missTids)
	if err != nil {
		logx.Errorf("Batch find tweets errorx: %v", err)
		return nil, err
	}

	// 6. 将数据库结果写入缓存（异步，改为调用SplitTweetToCache）
	go func() {
		for _, tweet := range dbTweets {
			// 只缓存公开且未删除的推文
			if tweet.IsPublic && !tweet.IsDeleted {
				if err := l.svcCtx.SetTweetToCache(context.Background(), tweet.Tid, tweet); err != nil {
					logx.Errorf("BatchGetTweets SplitTweetToCache errorx, tid:%d, err:%v", tweet.Tid, err)
				}
			}
		}
	}()

	// 7. 合并缓存和数据库结果
	for _, tweet := range dbTweets {
		// 只返回公开且未删除的推文
		if tweet.IsPublic && !tweet.IsDeleted {
			cachedTweets[tweet.Tid] = tweet
		}
	}

	// 8. 构建响应（使用 svcCtx.BuildTweet）
	return &pb.BatchGetTweetsResp{
		Code:   0,
		Msg:    "success",
		Tweets: l.buildTweetsFromMap(cachedTweets),
	}, nil
}

// buildTweetsFromMap 从 map 构建推文列表（使用 svcCtx.BuildTweet）
func (l *BatchGetTweetsLogic) buildTweetsFromMap(tweetMap map[int64]*model.Tweet) []*pb.Tweet {
	tweets := make([]*pb.Tweet, 0, len(tweetMap))
	for _, tweet := range tweetMap {
		tweets = append(tweets, l.svcCtx.BuildTweet(tweet))
	}
	return tweets
}
