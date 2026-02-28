// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package user

import (
	"context"

	"gozeroX/app/usercenter/cmd/api/internal/svc"
	"gozeroX/app/usercenter/cmd/api/internal/types"
	"gozeroX/app/usercenter/cmd/rpc/pb" // RPC 的 pb

	"github.com/zeromicro/go-zero/core/logx"
)

type LoginLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// login
func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LoginLogic) Login(req *types.LoginReq) (resp *types.LoginResp, err error) {
	// 1. 调用 RPC
	rpcResp, err := l.svcCtx.UserCenterRpc.Login(l.ctx, &pb.LoginReq{
		Mobile:   req.Mobile,
		Password: req.Password,
	})
	if err != nil {
		return nil, err
	}

	// 2. 转换响应（RPC 的 pb 转为 API 的 types）
	return &types.LoginResp{
		AccessToken:  rpcResp.Token.AccessToken,
		AccessExpire: rpcResp.Token.AccessExpire,
		UserInfo: types.UserInfo{
			Uid:         rpcResp.UserInfo.Uid,
			Nickname:    rpcResp.UserInfo.Nickname,
			Avatar:      rpcResp.UserInfo.Avatar,
			Bio:         rpcResp.UserInfo.Bio,
			FollowCount: rpcResp.UserInfo.FollowCount,
			FansCount:   rpcResp.UserInfo.FansCount,
			PostCount:   rpcResp.UserInfo.PostCount,
		},
	}, nil
}
