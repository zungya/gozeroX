package content

import (
	"context"

	"gozeroX/app/contentService/cmd/api/internal/svc"
	"gozeroX/app/contentService/cmd/api/internal/types"
	"gozeroX/app/contentService/cmd/rpc/pb"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
)

type CreateTweetLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateTweetLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateTweetLogic {
	return &CreateTweetLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateTweetLogic) CreateTweet(req *types.CreateTweetReq) (resp *types.CreateTweetResp, err error) {
	uid, ok := l.ctx.Value("user_id").(int64)
	if !ok {
		return nil, errors.New("无法获取用户ID")
	}

	rpcResp, err := l.svcCtx.ContentServiceRpc.CreateTweet(l.ctx, &pb.CreateTweetReq{
		Uid:       uid,
		Content:   req.Content,
		MediaUrls: req.MediaUrls,
		Tags:      req.Tags,
		IsPublic:  req.IsPublic,
	})
	if err != nil {
		return nil, err
	}

	if rpcResp.Code != 0 {
		return &types.CreateTweetResp{
			Code: rpcResp.Code,
			Msg:  rpcResp.Msg,
		}, nil
	}

	t := rpcResp.Tweet
	return &types.CreateTweetResp{
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
