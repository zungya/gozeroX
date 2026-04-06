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

func (l *ListTweetsLogic) ListTweets(req *types.ListTweetsUidReq) (resp *types.ListTweetsUidResp, err error) {
	// todo: add your logic here and delete this line
	// 1. 从 JWT 中获取当前登录用户ID（如果有）
	currentUid, _ := l.ctx.Value("uid").(int64)

	// 2. 判断是否为查看自己的主页
	isOwn := currentUid > 0 && currentUid == req.Uid

	// 3. 调用 RPC
	rpcResp, err := l.svcCtx.ContentServiceRpc.ListTweetsUid(l.ctx, &pb.ListTweetsUidReq{
		QueryUid: req.Uid,
		IsPublic: isOwn, // true: 自己的主页(查所有), false: 别人主页(只查公开)
		Page:     req.Page,
		Size:     req.Size,
		Sort:     l.convertSort(req.Sort),
	})
	if err != nil {
		logx.Errorf("ListTweetsUid RPC errorx: %v", err)
		return nil, err
	}

	// 4. 转换响应 - ✅ 直接使用 types.ListData 和 types.Pagination
	list := make([]types.Tweet, 0, len(rpcResp.Tweets))
	for _, t := range rpcResp.Tweets {
		list = append(list, types.Tweet{
			Tid:          t.Tid,
			Uid:          t.Uid,
			Content:      t.Content,
			MediaUrls:    t.MediaUrls,
			Tags:         t.Tags,
			IsPublic:     t.IsPublic,
			CreatedAt:    t.CreatedAt,
			IsDeleted:    t.IsDeleted,
			LikeCount:    t.LikeCount,
			CommentCount: t.CommentCount,
		})
	}

	return &types.ListTweetsUidResp{
		Code: rpcResp.Code,
		Msg:  rpcResp.Msg,
		Data: types.ListData{ // ✅ 直接用 types.ListData
			List: list,
			Pagination: types.Pagination{ // ✅ 直接用 types.Pagination
				Page:  rpcResp.Pagination.Page,
				Size:  rpcResp.Pagination.Size,
				Total: rpcResp.Pagination.Total,
			},
		},
	}, nil
}

// 转换排序参数
func (l *ListTweetsLogic) convertSort(sort string) pb.SortType {
	switch sort {
	case "created_at_desc":
		return pb.SortType_CREATED_AT_DESC
	case "created_at_asc":
		return pb.SortType_CREATED_AT_ASC
	default:
		return pb.SortType_CREATED_AT_DESC
	}
}
