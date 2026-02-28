package logic

import (
	"context"

	"gozeroX/app/contentService/cmd/rpc/internal/svc"
	"gozeroX/app/contentService/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListTweetsUidLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListTweetsUidLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListTweetsUidLogic {
	return &ListTweetsUidLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// ListTweetsUid 1. 用户主页推文列表（带权限过滤）
func (l *ListTweetsUidLogic) ListTweetsUid(in *pb.ListTweetsUidReq) (*pb.ListTweetsUidResp, error) {
	// 1. 参数校验和默认值处理
	if in.Page <= 0 {
		in.Page = 1
	}
	if in.Size <= 0 {
		in.Size = 10 // 默认每页10条
	}
	if in.Size > 100 {
		in.Size = 100 // 限制最大100条
	}

	// 2. 构建查询条件
	var isPublic *bool
	// 如果 is_public 为 true，表示要查询所有（公开+不公开）
	// 如果 is_public 为 false，表示只查询公开的
	if !in.IsPublic {
		// 只查询公开的
		trueVal := true
		isPublic = &trueVal
	}
	// 如果 in.IsPublic == true，isPublic 保持为 nil，表示查询所有（不区分公开/私密）

	// 3. 处理排序
	sortField := "created_at"
	sortOrder := "DESC"
	if in.Sort == pb.SortType_CREATED_AT_ASC {
		sortOrder = "ASC"
	}

	// 4. 查询数据库
	tweets, total, err := l.svcCtx.TweetModel.FindByUid(l.ctx, in.QueryUid, isPublic, in.Page, in.Size, sortField, sortOrder)
	if err != nil {
		logx.Errorf("Find tweets by uid error: %v", err)
		return nil, err
	}

	// 5. 转换为 PB 对象
	pbTweets := make([]*pb.Tweet, 0, len(tweets))
	for _, tweet := range tweets {
		pbTweets = append(pbTweets, l.svcCtx.BuildTweet(tweet))
	}

	// 6. 返回结果
	return &pb.ListTweetsUidResp{
		Code:   0,
		Msg:    "success",
		Tweets: pbTweets,
		Pagination: &pb.Pagination{
			Page:  in.Page,
			Size:  in.Size,
			Total: total,
		},
	}, nil
}
