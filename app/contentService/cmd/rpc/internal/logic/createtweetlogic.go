package logic

import (
	"context"
	"github.com/lib/pq"
	"gozeroX/app/contentService/model"
	"gozeroX/pkg/idgen"
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

func (l *CreateTweetLogic) CreateTweet(in *pb.CreateTweetReq) (*pb.CreateTweetResp, error) {

	// 2. 生成雪花ID作为业务主键
	snowTid, err := idgen.GenID()
	if err != nil {
		logx.Errorf("CreateTweet generate snowflake id errorx: %v", err)
		return &pb.CreateTweetResp{
			Code: 500,
			Msg:  "生成推文ID失败",
		}, nil
	}

	// 3. 构建推文对象（使用新的表结构）
	now := time.Now().UnixMilli() // 毫秒时间戳
	tweet := &model.Tweet{
		SnowTid:      snowTid, // 雪花ID作为业务主键
		Uid:          in.Uid,
		Content:      in.Content,
		MediaUrls:    pq.StringArray(in.MediaUrls),
		Tags:         pq.StringArray(in.Tags),
		IsPublic:     in.IsPublic,
		LikeCount:    0,
		CommentCount: 0,
		Status:       0, // 0-正常
		CreatedAt:    now,
	}

	// 4. 插入数据库（tid 是自增的，数据库会自动生成）
	result, err := l.svcCtx.TweetModel.Insert(l.ctx, tweet)
	if err != nil {
		logx.Errorf("Insert tweet errorx: %v", err)
		return &pb.CreateTweetResp{
			Code: 500,
			Msg:  "发布推文失败",
		}, nil
	}

	// 5. 获取自增ID（可选，如果需要可以获取）
	tid, err := result.LastInsertId()
	if err != nil {
		logx.Errorf("Get last insert id errorx: %v", err)
		// 即使获取失败也不影响业务，只是记录日志
	} else {
		tweet.Tid = tid
	}

	// 6. 存入Redis缓存（使用snowTid作为key）
	if err := l.svcCtx.SetTweetToCache(l.ctx, snowTid, tweet); err != nil {
		logx.Errorf("CreateTweet SetTweetToCache errorx, snowTid:%d, err:%v", snowTid, err)
	}

	// 7. 异步更新用户推文数
	go func() {
		if err := l.svcCtx.IncrUserTweetCount(context.Background()); err != nil {
			logx.Errorf("IncrUserTweetCount errorx, uid:%d, err:%v", in.Uid, err)
		}
	}()

	// 8. 返回结果
	return &pb.CreateTweetResp{
		Code:  0,
		Msg:   "success",
		Tweet: l.svcCtx.BuildTweet(tweet),
	}, nil
}
