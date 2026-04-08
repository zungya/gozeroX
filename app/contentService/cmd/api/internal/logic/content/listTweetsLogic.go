package content

import (
	"context"

	"gozeroX/app/contentService/cmd/api/internal/svc"
	"gozeroX/app/contentService/cmd/api/internal/types"
	"gozeroX/app/contentService/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListTweetsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListTweetsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListTweetsLogic {
	return &ListTweetsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListTweetsLogic) ListTweets(req *types.ListTweetsReq) (resp *types.ListTweetsResp, err error) {
	// 从 JWT 获取当前用户ID，判断是否查看自己的主页
	// 查看自己的主页可以看到私密推文，查看别人主页只能看公开推文
	currentUid, _ := l.ctx.Value("user_id").(int64)
	isOwnProfile := currentUid > 0 && currentUid == req.QueryUid

	rpcResp, err := l.svcCtx.ContentServiceRpc.ListTweetsUid(l.ctx, &pb.ListTweetsUidReq{
		QueryUid: req.QueryUid,
		IsPublic: isOwnProfile, // true=包含私密推文，false=仅公开
		Cursor:   req.Cursor,
		Limit:    req.Limit,
		Sort:     req.Sort,
	})
	if err != nil {
		return nil, err
	}

	if rpcResp.Code != 0 {
		return &types.ListTweetsResp{
			Code: rpcResp.Code,
			Msg:  rpcResp.Msg,
		}, nil
	}

	tweets := make([]types.Tweet, 0, len(rpcResp.Tweets))
	for _, t := range rpcResp.Tweets {
		tweets = append(tweets, types.Tweet{
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
		})
	}

	return &types.ListTweetsResp{
		Code:  0,
		Msg:   "success",
		Data:  tweets,
		Total: rpcResp.Total,
	}, nil
}
