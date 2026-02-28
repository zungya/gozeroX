// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package content

import (
	"context"
	"gozeroX/app/contentService/cmd/rpc/pb"

	"gozeroX/app/contentService/cmd/api/internal/svc"
	"gozeroX/app/contentService/cmd/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteTweetLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteTweetLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteTweetLogic {
	return &DeleteTweetLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteTweetLogic) DeleteTweet(req *types.DeleteTweetReq) (resp *types.DeleteTweetResp, err error) {
	// todo: add your logic here and delete this line
	// 1. 从 JWT 中获取当前登录用户ID
	currentUid, ok := l.ctx.Value("uid").(int64)
	if !ok || currentUid == 0 {
		return &types.DeleteTweetResp{
			Code: 401,
			Msg:  "未登录或登录已过期",
		}, nil
	}

	// 2. 权限校验：只能删除自己的推文
	if currentUid != req.Uid {
		return &types.DeleteTweetResp{
			Code: 403,
			Msg:  "无权删除他人的推文",
		}, nil
	}

	// 3. 调用 RPC
	rpcResp, err := l.svcCtx.ContentServiceRpc.DeleteTweet(l.ctx, &pb.DeleteTweetReq{
		Tid: req.Tid,
		Uid: req.Uid,
	})
	if err != nil {
		logx.Errorf("DeleteTweet RPC error: %v", err)
		return nil, err
	}

	// 4. 返回响应
	return &types.DeleteTweetResp{
		Code: rpcResp.Code,
		Msg:  rpcResp.Msg,
	}, nil
}
