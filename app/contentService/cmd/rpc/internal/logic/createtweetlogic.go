package logic

import (
	"context"
	"fmt"
	"github.com/lib/pq"
	"gozeroX/app/contentService/model"
	"time"

	"gozeroX/app/contentService/cmd/rpc/internal/svc"
	"gozeroX/app/contentService/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateTweetLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateTweetLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateTweetLogic {
	return &CreateTweetLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// CreateTweet 4. 创建推文（供API调用）
func (l *CreateTweetLogic) CreateTweet(in *pb.CreateTweetReq) (*pb.CreateTweetResp, error) {
	// todo: add your logic here and delete this line
	// 1. 参数校验
	if err := l.validateParams(in); err != nil {
		return &pb.CreateTweetResp{
			Code: 400,
			Msg:  err.Error(),
		}, nil
	}

	// 2. 构建推文对象
	tweet := &model.Tweet{
		Uid:          in.Uid,
		Content:      in.Content,
		MediaUrls:    pq.StringArray(in.MediaUrls),
		Tags:         pq.StringArray(in.Tags),
		IsPublic:     in.IsPublic,
		CreatedAt:    time.Now(),
		IsDeleted:    false,
		LikeCount:    0,
		CommentCount: 0,
	}

	// 3. 插入数据库
	result, err := l.svcCtx.TweetModel.Insert(l.ctx, tweet)
	if err != nil {
		logx.Errorf("Insert tweet error: %v", err)
		return nil, err
	}

	// 4. 获取自增ID
	tid, err := result.LastInsertId()
	if err != nil {
		logx.Errorf("Get last insert id error: %v", err)
		return nil, err
	}
	tweet.Tid = tid

	// ✅ 5. 存入缓存 - 模块名"tweet"，过期1小时（完全模仿user风格）
	_ = l.svcCtx.CacheManager.Set(l.ctx, "tweet", "info", tid, tweet, 3600)

	// 6. 返回结果
	return &pb.CreateTweetResp{
		Code:  0,
		Msg:   "success",
		Tweet: l.svcCtx.BuildTweet(tweet), // 使用 svcCtx 的 Build 方法
	}, nil
}

// 参数校验
func (l *CreateTweetLogic) validateParams(in *pb.CreateTweetReq) error {
	if in.Uid == 0 {
		return fmt.Errorf("用户ID不能为空")
	}
	if len(in.Content) == 0 {
		return fmt.Errorf("推文内容不能为空")
	}
	if len(in.Content) > 1000 {
		return fmt.Errorf("推文内容不能超过1000字")
	}
	if len(in.MediaUrls) > 9 {
		return fmt.Errorf("最多只能上传9张图片")
	}
	if len(in.Tags) > 10 {
		return fmt.Errorf("最多只能添加10个标签")
	}
	return nil
}
