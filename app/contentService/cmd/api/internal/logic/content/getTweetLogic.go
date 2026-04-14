package content

import (
	"context"

	"gozeroX/app/contentService/cmd/api/internal/svc"
	"gozeroX/app/contentService/cmd/api/internal/types"
	"gozeroX/app/contentService/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetTweetLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetTweetLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetTweetLogic {
	return &GetTweetLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetTweetLogic) GetTweet(req *types.GetTweetReq) (resp *types.GetTweetResp, err error) {
	rpcResp, err := l.svcCtx.ContentServiceRpc.GetTweetBySnowTid(l.ctx, &pb.GetTweetBySnowTidReq{
		SnowTid: req.SnowTid,
	})
	if err != nil {
		return nil, err
	}

	if rpcResp.Code != 0 {
		return &types.GetTweetResp{
			Code: rpcResp.Code,
			Msg:  rpcResp.Msg,
		}, nil
	}

	t := rpcResp.Tweet
	return &types.GetTweetResp{
		Code: 0,
		Msg:  "success",
		Data: types.Tweet{
			SnowTid:      t.SnowTid,
			Uid:          t.Uid,
			Content:      t.Content,
			MediaUrls:    t.MediaUrls,
			Tags:         t.Tags,
			IsPublic:     t.IsPublic,
			CreatedAt:    t.CreatedAt,
			LikeCount:    t.LikeCount,
			CommentCount: t.CommentCount,
			Status:       t.Status,
			Nickname:     t.Nickname,
			Avatar:       t.Avatar,
		},
	}, nil
}
