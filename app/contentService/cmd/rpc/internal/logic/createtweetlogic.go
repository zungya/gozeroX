package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lib/pq"
	"gozeroX/app/contentService/cmd/rpc/internal/svc"
	"gozeroX/app/contentService/cmd/rpc/pb"
	"gozeroX/app/contentService/model"
	"gozeroX/app/usercenter/cmd/rpc/usercenter"
	"gozeroX/pkg/idgen"

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

// CreateTweet 创建推文
func (l *CreateTweetLogic) CreateTweet(in *pb.CreateTweetReq) (*pb.CreateTweetResp, error) {
	// 1. 生成雪花ID作为业务主键
	snowTid, err := idgen.GenID()
	if err != nil {
		logx.Errorf("CreateTweet generate snowflake id error: %v", err)
		return &pb.CreateTweetResp{
			Code: 500,
			Msg:  "生成推文ID失败",
		}, nil
	}

	// 2. 构建推文对象
	now := time.Now().UnixMilli() // 毫秒时间戳
	tweet := &model.Tweet{
		SnowTid:      snowTid,
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

	// 3. 插入数据库
	_, err = l.svcCtx.TweetModel.Insert(l.ctx, tweet)
	if err != nil {
		logx.Errorf("CreateTweet insert tweet error: %v", err)
		return &pb.CreateTweetResp{
			Code: 500,
			Msg:  "发布推文失败",
		}, nil
	}

	// 4. 存入Redis缓存
	if err := l.svcCtx.SetTweetToCache(l.ctx, snowTid, tweet); err != nil {
		logx.Errorf("CreateTweet SetTweetToCache error, snowTid:%d, err:%v", snowTid, err)
	}

	// 5. 异步更新用户推文数
	go func() {
		if err := l.svcCtx.IncrUserPostCount(context.Background(), in.Uid, 1); err != nil {
			logx.Errorf("CreateTweet IncrUserPostCount error, uid:%d, err:%v", in.Uid, err)
		}
	}()

	// 5.5 异步发送推文入库事件到 Kafka（推荐系统用）
	go func() {
		if err := l.sendRecommendIndexMessage(tweet); err != nil {
			logx.Errorf("CreateTweet sendRecommendIndexMessage error, snowTid:%d, err:%v", snowTid, err)
		}
	}()

	// 6. 获取用户信息（用于填充 nickname 和 avatar）
	userResp, err := l.svcCtx.UserCenterRpc.GetUserInfo(l.ctx, &usercenter.GetUserInfoReq{Uid: in.Uid})
	if err != nil {
		logx.Errorf("CreateTweet GetUserInfo error, uid:%d, err:%v", in.Uid, err)
		// 用户信息获取失败，仍然返回推文（只是没有用户信息）
		return &pb.CreateTweetResp{
			Code:  0,
			Msg:   "success",
			Tweet: l.svcCtx.BuildTweet(tweet),
		}, nil
	}

	// 7. 构建返回结果（包含用户信息）
	pbTweet := l.svcCtx.BuildTweet(tweet)
	if userResp.Code == 0 && userResp.UserInfo != nil {
		pbTweet.Nickname = userResp.UserInfo.Nickname
		pbTweet.Avatar = userResp.UserInfo.Avatar
	}

	return &pb.CreateTweetResp{
		Code:  0,
		Msg:   "success",
		Tweet: pbTweet,
	}, nil
}

// sendRecommendIndexMessage 发送推文入库消息到 Kafka recommend_tweet topic
func (l *CreateTweetLogic) sendRecommendIndexMessage(tweet *model.Tweet) error {
	message := map[string]interface{}{
		"action":     "index_tweet",
		"snow_tid":   tweet.SnowTid,
		"uid":        tweet.Uid,
		"content":    tweet.Content,
		"media_urls": tweet.MediaUrls,
		"tags":       tweet.Tags,
		"created_at": tweet.CreatedAt,
	}

	body, err := json.Marshal(message)
	if err != nil {
		return err
	}

	pusher := l.svcCtx.GetPusher("recommend_tweet")
	return pusher.PushWithKey(context.Background(), fmt.Sprintf("%d", tweet.SnowTid), string(body))
}
