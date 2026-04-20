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
		l.Errorf("UpdateUserStats error, uid:%d, type:%d, delta:%d, err:%v", in.Uid, in.UpdateType, in.Delta, err)
		return &pb.UpdateUserStatsResp{
			Code: 500,
			Msg:  "更新统计失败",
		}, nil
	}
	return &pb.UpdateUserStatsResp{
		Code: 0,
		Msg:  "success",
	}, nil
}
