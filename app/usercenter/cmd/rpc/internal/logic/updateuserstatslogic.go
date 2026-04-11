package logic

import (
	"context"

	"gozeroX/app/usercenter/cmd/rpc/internal/svc"
	"gozeroX/app/usercenter/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateUserStatsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateUserStatsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateUserStatsLogic {
	return &UpdateUserStatsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 更新用户统计字段（follow_count/fans_count/post_count）
func (l *UpdateUserStatsLogic) UpdateUserStats(in *pb.UpdateUserStatsReq) (*pb.UpdateUserStatsResp, error) {
	// 1. 更新数据库
	if err := l.svcCtx.UserModel.UpdateStatsWithValues(l.ctx, in.Uid, in.UpdateType, in.Delta); err != nil {
		logx.Errorf("UpdateUserStats error, uid:%d, type:%d, delta:%d, err:%v", in.Uid, in.UpdateType, in.Delta, err)
		return &pb.UpdateUserStatsResp{
			Code: 500,
			Msg:  "更新统计失败",
		}, nil
	}

	// 2. 更新 Redis 缓存中对应的字段
	var cacheField string
	switch in.UpdateType {
	case 1:
		cacheField = "follow_count"
	case 2:
		cacheField = "fans_count"
	case 3:
		cacheField = "post_count"
	default:
		return &pb.UpdateUserStatsResp{Code: 0, Msg: "success"}, nil
	}

	if _, err := l.svcCtx.CacheManager.HIncrBy(l.ctx, "user", "info", in.Uid, cacheField, int(in.Delta)); err != nil {
		logx.Errorf("UpdateUserStats cache HIncrBy error, uid:%d, field:%s, err:%v", in.Uid, cacheField, err)
		// 缓存更新失败不影响主流程，DB 已更新成功
	}

	return &pb.UpdateUserStatsResp{
		Code: 0,
		Msg:  "success",
	}, nil
}
