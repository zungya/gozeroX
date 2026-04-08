package content

import (
	"context"

	"gozeroX/app/contentService/cmd/api/internal/svc"
	"gozeroX/app/contentService/cmd/api/internal/types"
	"gozeroX/app/contentService/cmd/rpc/pb"

	"github.com/pkg/errors"
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
	uid, ok := l.ctx.Value("user_id").(int64)
	if !ok {
		return nil, errors.New("无法获取用户ID")
	}

	rpcResp, err := l.svcCtx.ContentServiceRpc.DeleteTweet(l.ctx, &pb.DeleteTweetReq{
		SnowTid: req.SnowTid,
		Uid:     uid,
	})
	if err != nil {
		return nil, err
	}

	return &types.DeleteTweetResp{
		Code: rpcResp.Code,
		Msg:  rpcResp.Msg,
	}, nil
}
