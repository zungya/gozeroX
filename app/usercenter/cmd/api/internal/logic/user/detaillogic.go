// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package user

import (
	"context"
	"gozeroX/app/usercenter/cmd/rpc/usercenter"

	"gozeroX/app/usercenter/cmd/api/internal/svc"
	"gozeroX/app/usercenter/cmd/api/internal/types"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
)

type DetailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// get user info
func NewDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DetailLogic {
	return &DetailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DetailLogic) Detail(req *types.UserInfoReq) (resp *types.UserInfoResp, err error) {
	// todo: add your logic here and delete this line
	userId, ok := l.ctx.Value("user_id").(int64)
	if !ok {
		return nil, errors.New("无法获取用户ID")
	}

	// 2. 如果请求中传了 uid，且当前用户有权限，可以用传的 uid
	// 这里简单处理：如果 req.Uid 不为 0，用 req.Uid，否则用 token 里的
	queryUid := userId
	if req.Uid != 0 {
		// TODO: 校验权限（比如管理员才能查别人）
		queryUid = req.Uid
	}

	// 3. 调用 RPC 获取用户信息
	rpcResp, err := l.svcCtx.UserCenterRpc.GetUserInfo(l.ctx, &usercenter.GetUserInfoReq{
		Uid: queryUid,
	})
	if err != nil {
		return nil, err
	}

	// 4. 转换响应
	return &types.UserInfoResp{
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
