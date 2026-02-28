package logic

import (
	"context"
	"gozeroX/app/contentService/model"

	"errors"

	"gozeroX/app/contentService/cmd/rpc/internal/svc"
	"gozeroX/app/contentService/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetTweetByTidLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetTweetByTidLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetTweetByTidLogic {
	return &GetTweetByTidLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// GetTweetByTid 2. 单条推文查询
func (l *GetTweetByTidLogic) GetTweetByTid(in *pb.GetTweetByTidReq) (*pb.GetTweetByTidResp, error) {
	// ✅ 1. 从缓存获取
	var cachedTweet model.Tweet
	err := l.svcCtx.CacheManager.Get(l.ctx, "tweet", "info", in.Tid, &cachedTweet)
	if err == nil {
		logx.Infof("缓存命中: tweet:%d", in.Tid)
		return &pb.GetTweetByTidResp{
			Code:  0,
			Msg:   "success",
			Tweet: l.svcCtx.BuildTweet(&cachedTweet),
		}, nil
	}

	// 2. 查数据库
	tweet, err := l.svcCtx.TweetModel.FindOne(l.ctx, in.Tid)
	if err != nil {
		// ✅ 使用 errors.Is 判断是否为 ErrNotFound
		if errors.Is(err, model.ErrNotFound) {
			return &pb.GetTweetByTidResp{
				Code: 404,
				Msg:  "推文不存在",
			}, nil
		}
		logx.Errorf("Find tweet error: %v", err)
		return nil, err
	}

	// 3. 存入缓存
	_ = l.svcCtx.CacheManager.Set(l.ctx, "tweet", "info", in.Tid, tweet, 3600)

	return &pb.GetTweetByTidResp{
		Code:  0,
		Msg:   "success",
		Tweet: l.svcCtx.BuildTweet(tweet),
	}, nil
}
