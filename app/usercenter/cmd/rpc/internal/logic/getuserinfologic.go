package logic

import (
	"context"
	"gozeroX/app/usercenter/model"

	"gozeroX/app/usercenter/cmd/rpc/internal/svc"
	"gozeroX/app/usercenter/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserInfoLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetUserInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserInfoLogic {
	return &GetUserInfoLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// GetUserInfo 获取用户信息
func (l *GetUserInfoLogic) GetUserInfo(in *pb.GetUserInfoReq) (*pb.GetUserInfoResp, error) {
	// todo: add your logic here and delete this line
	// ✅ 1. 从缓存获取 - 模块名"user"，ID为uid
	var cachedUser model.User
	err := l.svcCtx.CacheManager.Get(l.ctx, "user", "info", in.Uid, &cachedUser)
	if err == nil {
		logx.Infof("缓存命中: user:%d", in.Uid)
		return &pb.GetUserInfoResp{
			UserInfo: l.svcCtx.BuildUserInfo(&cachedUser),
		}, nil
	}

	// 2. 查数据库
	user, err := l.svcCtx.UserModel.FindOne(l.ctx, in.Uid)
	if err != nil {
		return nil, err
	}

	// ✅ 3. 存入缓存 - 模块名"user"，过期1小时
	_ = l.svcCtx.CacheManager.Set(l.ctx, "user", "info", in.Uid, user, 3600)

	return &pb.GetUserInfoResp{
		UserInfo: l.svcCtx.BuildUserInfo(user),
	}, nil
}
