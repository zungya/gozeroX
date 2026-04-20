package recommend

import (
	"context"
	"errors"

	"gozeroX/app/recommendService/cmd/api/internal/svc"
	"gozeroX/app/recommendService/cmd/api/internal/types"
	"gozeroX/app/recommendService/cmd/rpc/recommend"

	"github.com/zeromicro/go-zero/core/logx"
)

type FeedLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewFeedLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FeedLogic {
	return &FeedLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *FeedLogic) Feed(req *types.RecommendFeedReq) (resp *types.RecommendFeedResp, err error) {
	// 从 JWT context 获取 uid
	uid, ok := l.ctx.Value("user_id").(int64)
	if !ok {
		return nil, errors.New("无法获取用户ID")
	}

	// 调用 recommendService RPC
	rpcResp, err := l.svcCtx.RecommendRpc.RecommendFeed(l.ctx, &recommend.RecommendFeedReq{
		Uid:   uid,
		Limit: req.Limit,
	})
	if err != nil {
		l.Errorf("Feed RecommendRpc.RecommendFeed error, uid:%d, err:%v", uid, err)
		return &types.RecommendFeedResp{
			Code: 500,
			Msg:  "推荐服务调用失败",
		}, nil
	}

	// 转换 RPC 响应为 API 响应
	var tweets []types.TweetInfo
	for _, t := range rpcResp.Tweets {
		tweets = append(tweets, types.TweetInfo{
			SnowTid:      t.SnowTid,
			Uid:          t.Uid,
			Content:      t.Content,
			MediaUrls:    t.MediaUrls,
			Tags:         t.Tags,
			LikeCount:    t.LikeCount,
			CommentCount: t.CommentCount,
			CreatedAt:    t.CreatedAt,
			Nickname:     t.Nickname,
			Avatar:       t.Avatar,
		})
	}

	return &types.RecommendFeedResp{
		Code: rpcResp.Code,
		Msg:  rpcResp.Msg,
		Data: tweets,
	}, nil
}
