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
	// todo: add your logic here and delete this line
	// 1. 从 JWT 中获取当前登录用户ID
	currentUid, ok := l.ctx.Value("uid").(int64)
	if !ok || currentUid == 0 {
		return &types.CreateTweetResp{
			Code: 401,
			Msg:  "未登录或登录已过期",
		}, nil
	}

	// 2. 权限校验：只能发布自己的推文
	if currentUid != req.Uid {
		return &types.CreateTweetResp{
			Code: 403,
			Msg:  "无权发布他人的推文",
		}, nil
	}

	// 3. 调用 RPC
	rpcResp, err := l.svcCtx.ContentServiceRpc.CreateTweet(l.ctx, &pb.CreateTweetReq{
		Uid:       req.Uid,
		Content:   req.Content,
		MediaUrls: req.MediaUrls,
		Tags:      req.Tags,
		IsPublic:  req.IsPublic,
	})
	if err != nil {
		logx.Errorf("CreateTweet RPC error: %v", err)
		return nil, err
	}

	// 4. 转换响应
	if rpcResp.Code != 0 {
		return &types.CreateTweetResp{
			Code: rpcResp.Code,
			Msg:  rpcResp.Msg,
		}, nil
	}

	return &types.CreateTweetResp{
		Code: 0,
		Msg:  "发布成功",
		Data: types.Tweet{
			Tid:          rpcResp.Tweet.Tid,
			Uid:          rpcResp.Tweet.Uid,
			Content:      rpcResp.Tweet.Content,
			MediaUrls:    rpcResp.Tweet.MediaUrls,
			Tags:         rpcResp.Tweet.Tags,
			IsPublic:     rpcResp.Tweet.IsPublic,
			CreatedAt:    rpcResp.Tweet.CreatedAt,
			IsDeleted:    rpcResp.Tweet.IsDeleted,
			LikeCount:    rpcResp.Tweet.LikeCount,
			CommentCount: rpcResp.Tweet.CommentCount,
		},
	}, nil
}
