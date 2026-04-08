package logic

import (
	"context"

	"gozeroX/app/contentService/cmd/rpc/internal/svc"
	"gozeroX/app/contentService/cmd/rpc/pb"
	"gozeroX/app/usercenter/cmd/rpc/usercenter"

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

// ListTweetsUid 用户主页推文列表（游标分页）
func (l *ListTweetsUidLogic) ListTweetsUid(in *pb.ListTweetsUidReq) (*pb.ListTweetsUidResp, error) {
	// 1. 参数校验和默认值处理
	if in.Limit <= 0 {
		in.Limit = 10 // 默认每页10条
	}
	if in.Limit > 100 {
		in.Limit = 100 // 限制最大100条
	}

	// 2. 构建查询条件
	var isPublic *bool
	// 如果 is_public 为 false，表示只查询公开的
	if !in.IsPublic {
		trueVal := true
		isPublic = &trueVal
	}
	// 如果 in.IsPublic == true，isPublic 保持为 nil，表示查询所有（不区分公开/私密)

	// 3. 处理排序
	sortOrder := "DESC"
	if in.Sort == 1 {
		sortOrder = "ASC"
	}

	// 4. 查询数据库（使用游标分页）
	tweets, total, err := l.svcCtx.TweetModel.FindByUid(l.ctx, in.QueryUid, isPublic, in.Cursor, in.Limit, "created_at", sortOrder)
	if err != nil {
		logx.Errorf("ListTweetsUid find tweets error: %v", err)
		return nil, err
	}

	// 5. 获取用户信息（所有推文都属于同一个用户，只需调用一次）
	var nickname, avatar string
	userResp, err := l.svcCtx.UserCenterRpc.GetUserInfo(l.ctx, &usercenter.GetUserInfoReq{Uid: in.QueryUid})
	if err != nil {
		logx.Errorf("ListTweetsUid GetUserInfo error, uid:%d, err:%v", in.QueryUid, err)
		// 用户信息获取失败，仍然返回推文（只是没有用户信息）
	} else if userResp.Code == 0 && userResp.UserInfo != nil {
		nickname = userResp.UserInfo.Nickname
		avatar = userResp.UserInfo.Avatar
	}

	// 6. 转换为 PB 对象（填充用户信息）
	pbTweets := make([]*pb.Tweet, 0, len(tweets))
	for _, tweet := range tweets {
		pbTweet := l.svcCtx.BuildTweet(tweet)
		pbTweet.Nickname = nickname
		pbTweet.Avatar = avatar
		pbTweets = append(pbTweets, pbTweet)
	}

	// 7. 返回结果
	return &pb.ListTweetsUidResp{
		Code:   0,
		Msg:    "success",
		Tweets: pbTweets,
		Total:  total,
	}, nil
}
