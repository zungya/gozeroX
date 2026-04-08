package interaction

import (
	"context"

	"gozeroX/app/interactService/cmd/api/internal/svc"
	"gozeroX/app/interactService/cmd/api/internal/types"
	"gozeroX/app/interactService/cmd/rpc/pb"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserLikesAllLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetUserLikesAllLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserLikesAllLogic {
	return &GetUserLikesAllLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetUserLikesAllLogic) GetUserLikesAll(req *types.GetUserlikesAllReq) (resp *types.GetUserLikesAllResp, err error) {
	uid, ok := l.ctx.Value("user_id").(int64)
	if !ok {
		return nil, errors.New("无法获取用户ID")
	}

	rpcResp, err := l.svcCtx.InteractService.GetUserAllLikes(l.ctx, &pb.GetUserAllLikesReq{
		Uid:    uid,
		Cursor: req.LikesCursor,
	})
	if err != nil {
		return nil, err
	}

	if rpcResp.Code != 0 {
		return &types.GetUserLikesAllResp{
			Code:    int(rpcResp.Code),
			Message: rpcResp.Msg,
		}, nil
	}

	// 转换推文点赞
	tweetLikes := make([]types.UserTweetLike, 0, len(rpcResp.TweetLikes))
	for _, t := range rpcResp.TweetLikes {
		tweetLikes = append(tweetLikes, types.UserTweetLike{
			SnowTid:     t.SnowTid,
			SnowLikesId: t.SnowLikesId,
			Status:      t.Status,
		})
	}

	// 转换评论点赞
	commentLikes := make([]types.UserCommentLike, 0, len(rpcResp.CommentLikes))
	for _, c := range rpcResp.CommentLikes {
		commentLikes = append(commentLikes, types.UserCommentLike{
			SnowCid:     c.SnowCid,
			SnowLikesId: c.SnowLikesId,
			Status:      c.Status,
		})
	}

	return &types.GetUserLikesAllResp{
		Code:             0,
		Message:          "success",
		LikesForTweets:   tweetLikes,
		LikesForComments: commentLikes,
	}, nil
}
