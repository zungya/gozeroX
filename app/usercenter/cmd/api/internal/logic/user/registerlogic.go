// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package user

import (
	"context"
	"gozeroX/app/usercenter/cmd/rpc/usercenter"

	"gozeroX/app/usercenter/cmd/api/internal/svc"
	"gozeroX/app/usercenter/cmd/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type RegisterLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// register
func NewRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterLogic {
	return &RegisterLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RegisterLogic) Register(req *types.RegisterReq) (resp *types.RegisterResp, err error) {
	// todo: add your logic here and delete this line
	rpcResp, err := l.svcCtx.UserCenterRpc.Register(l.ctx, &usercenter.RegisterReq{
		Mobile:   req.Mobile,
		Password: req.Password,
	})
	if err != nil {
		return nil, err
	}

	// 2. 转换响应
	return &types.RegisterResp{
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
