package logic

import (
	"context"

	"gozeroX/app/contentService/cmd/rpc/internal/svc"
	"gozeroX/app/contentService/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateTweetStatsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateTweetStatsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateTweetStatsLogic {
	return &UpdateTweetStatsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 更新推文统计字段（like_count / comment_count）
func (l *UpdateTweetStatsLogic) UpdateTweetStats(in *pb.UpdateTweetStatsReq) (*pb.UpdateTweetStatsResp, error) {
	// 1. 更新数据库
	if err := l.svcCtx.TweetModel.UpdateCount(l.ctx, in.SnowTid, in.UpdateType, in.Delta); err != nil {
		logx.Errorf("UpdateTweetStats error, snowTid:%d, type:%d, delta:%d, err:%v", in.SnowTid, in.UpdateType, in.Delta, err)
		return &pb.UpdateTweetStatsResp{
			Code: 500,
			Msg:  "更新统计失败",
		}, nil
	}

	// 2. 更新 Redis 缓存中对应的字段
	var cacheField string
	switch in.UpdateType {
	case 1:
		cacheField = "like_count"
	case 2:
		cacheField = "comment_count"
	default:
		return &pb.UpdateTweetStatsResp{Code: 0, Msg: "success"}, nil
	}

	if _, err := l.svcCtx.CacheManager.HIncrBy(l.ctx, "tweet", "info", in.SnowTid, cacheField, int(in.Delta)); err != nil {
		logx.Errorf("UpdateTweetStats cache HIncrBy error, snowTid:%d, field:%s, err:%v", in.SnowTid, cacheField, err)
		// 缓存更新失败不影响主流程，DB 已更新成功
	}

	return &pb.UpdateTweetStatsResp{
		Code: 0,
		Msg:  "success",
	}, nil
}
