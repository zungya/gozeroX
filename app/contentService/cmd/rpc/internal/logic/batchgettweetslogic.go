package logic

import (
	"context"
	"sync"

	"gozeroX/app/contentService/cmd/rpc/internal/svc"
	"gozeroX/app/contentService/cmd/rpc/pb"
	"gozeroX/app/contentService/model"
	"gozeroX/app/usercenter/cmd/rpc/usercenter"

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

// BatchGetTweets 批量推文查询（仅返回公开的）
func (l *BatchGetTweetsLogic) BatchGetTweets(in *pb.BatchGetTweetsReq) (*pb.BatchGetTweetsResp, error) {
	// 1. 参数校验
	if len(in.SnowTids) == 0 {
		return &pb.BatchGetTweetsResp{
			Code:   0,
			Msg:    "success",
			Tweets: []*pb.Tweet{},
		}, nil
	}

	// 2. 去重
	tidSet := make(map[int64]struct{})
	for _, tid := range in.SnowTids {
		tidSet[tid] = struct{}{}
	}
	uniqueTids := make([]int64, 0, len(tidSet))
	for tid := range tidSet {
		uniqueTids = append(uniqueTids, tid)
	}

	// 3. 批量从缓存获取推文
	cachedTweets := make(map[int64]*model.Tweet)
	missTids := make([]int64, 0)

	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, tid := range uniqueTids {
		wg.Add(1)
		go func(tid int64) {
			defer wg.Done()

			tweet, err := l.svcCtx.GetTweetFromCache(l.ctx, tid)
			if err == nil {
				if tweet.IsPublic && tweet.Status == 0 {
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

	l.Infof("批量查询推文: 总请求=%d, 唯一TID=%d, 缓存命中=%d, 未命中=%d",
		len(in.SnowTids), len(uniqueTids), len(cachedTweets), len(missTids))

	// 4. 如果没有缓存未命中的，直接返回
	if len(missTids) == 0 {
		return l.buildRespWithUserInfo(cachedTweets)
	}

	// 5. 批量查询数据库（未命中的）
	dbTweets, err := l.svcCtx.TweetModel.FindBatchBySnowTids(l.ctx, missTids)
	if err != nil {
		l.Errorf("Batch find tweets error: %v", err)
		return nil, err
	}

	// 6. 将数据库结果写入缓存（异步）
	go func() {
		for _, tweet := range dbTweets {
			if tweet.IsPublic && tweet.Status == 0 {
				if err := l.svcCtx.SetTweetToCache(context.Background(), tweet.SnowTid, tweet); err != nil {
					l.Errorf("BatchGetTweets SetTweetToCache error, snowTid:%d, err:%v", tweet.SnowTid, err)
				}
			}
		}
	}()

	// 7. 合并缓存和数据库结果
	for _, tweet := range dbTweets {
		if tweet.IsPublic && tweet.Status == 0 {
			cachedTweets[tweet.SnowTid] = tweet
		}
	}

	// 8. 返回结果（包含用户信息）
	return l.buildRespWithUserInfo(cachedTweets)
}

// buildRespWithUserInfo 构建响应并填充用户信息
func (l *BatchGetTweetsLogic) buildRespWithUserInfo(tweetMap map[int64]*model.Tweet) (*pb.BatchGetTweetsResp, error) {
	if len(tweetMap) == 0 {
		return &pb.BatchGetTweetsResp{
			Code:   0,
			Msg:    "success",
			Tweets: []*pb.Tweet{},
		}, nil
	}

	// 1. 收集所有 uid
	uidSet := make(map[int64]struct{})
	for _, tweet := range tweetMap {
		uidSet[tweet.Uid] = struct{}{}
	}
	uids := make([]int64, 0, len(uidSet))
	for uid := range uidSet {
		uids = append(uids, uid)
	}

	// 2. 批量获取用户信息
	userBriefResp, err := l.svcCtx.UserCenterRpc.BatchGetUserBrief(l.ctx, &usercenter.BatchUserBriefReq{
		Uids: uids,
	})
	if err != nil {
		l.Errorf("BatchGetUserBrief error: %v", err)
		// 用户信息获取失败，仍然返回推文（只是没有用户信息）
		return &pb.BatchGetTweetsResp{
			Code:   0,
			Msg:    "success",
			Tweets: l.buildTweetsFromMap(tweetMap, nil),
		}, nil
	}

	// 3. 构建 uid -> UserBrief 的映射
	userMap := make(map[int64]*usercenter.UserBrief)
	for _, user := range userBriefResp.Users {
		userMap[user.Uid] = user
	}

	// 4. 构建推文列表（包含用户信息）
	return &pb.BatchGetTweetsResp{
		Code:   0,
		Msg:    "success",
		Tweets: l.buildTweetsFromMap(tweetMap, userMap),
	}, nil
}

// buildTweetsFromMap 从 map 构建推文列表
func (l *BatchGetTweetsLogic) buildTweetsFromMap(tweetMap map[int64]*model.Tweet, userMap map[int64]*usercenter.UserBrief) []*pb.Tweet {
	tweets := make([]*pb.Tweet, 0, len(tweetMap))
	for _, tweet := range tweetMap {
		pbTweet := l.svcCtx.BuildTweet(tweet)

		// 填充用户信息
		if userMap != nil {
			if user, ok := userMap[tweet.Uid]; ok {
				pbTweet.Nickname = user.Nickname
				pbTweet.Avatar = user.Avatar
			}
		}

		tweets = append(tweets, pbTweet)
	}
	return tweets
}
