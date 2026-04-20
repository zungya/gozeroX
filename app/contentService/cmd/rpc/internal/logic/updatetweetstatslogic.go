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
// 仅更新数据库，Redis 缓存由各 RPC 层乐观更新
func (l *UpdateTweetStatsLogic) UpdateTweetStats(in *pb.UpdateTweetStatsReq) (*pb.UpdateTweetStatsResp, error) {
	if err := l.svcCtx.TweetModel.UpdateCount(l.ctx, in.SnowTid, in.UpdateType, in.Delta); err != nil {
		l.Errorf("UpdateTweetStats error, snowTid:%d, type:%d, delta:%d, err:%v", in.SnowTid, in.UpdateType, in.Delta, err)
		return &pb.UpdateTweetStatsResp{
			Code: 500,
			Msg:  "更新统计失败",
		}, nil
	}

	return &pb.UpdateTweetStatsResp{
		Code: 0,
		Msg:  "success",
	}, nil
}
