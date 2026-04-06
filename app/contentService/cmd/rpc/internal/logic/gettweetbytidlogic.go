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

func (l *GetTweetByTidLogic) GetTweetByTid(in *pb.GetTweetByTidReq) (*pb.GetTweetByTidResp, error) {
	// ✅ 1. 从缓存合并获取完整推文
	cachedTweet, err := l.svcCtx.GetTweetFromCache(l.ctx, in.Tid)
	if err == nil {
		logx.Infof("缓存命中: tweet:%d", in.Tid)
		return &pb.GetTweetByTidResp{
			Code:  0,
			Msg:   "success",
			Tweet: l.svcCtx.BuildTweet(cachedTweet),
		}, nil
	}
	logx.Infof("缓存未命中: tweet:%d, err:%v", in.Tid, err)

	// 2. 查数据库
	tweet, err := l.svcCtx.TweetModel.FindOne(l.ctx, in.Tid)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return &pb.GetTweetByTidResp{
				Code: 404,
				Msg:  "推文不存在",
			}, nil
		}
		logx.Errorf("Find tweet errorx: %v", err)
		return nil, err
	}

	// 3. 异步存入缓存（调用通用拆分方法）
	go func() {
		if err := l.svcCtx.SetTweetToCache(context.Background(), in.Tid, tweet); err != nil {
			logx.Errorf("SplitTweetToCache errorx, tid:%d, err:%v", in.Tid, err)
			// 缓存存储失败不影响主流程，仅打日志
		}
	}()

	return &pb.GetTweetByTidResp{
		Code:  0,
		Msg:   "success",
		Tweet: l.svcCtx.BuildTweet(tweet),
	}, nil
}
