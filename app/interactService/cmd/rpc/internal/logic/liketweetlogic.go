package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"gozeroX/app/interactService/model"
	"gozeroX/pkg/idgen"
	"time"

	"gozeroX/app/interactService/cmd/rpc/internal/svc"
	"gozeroX/app/interactService/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type LikeTweetLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewLikeTweetLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LikeTweetLogic {
	return &LikeTweetLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// LikeTweet 推文点赞/取消点赞（write-behind模式）
// 设计原则：
// 1. 第一次操作：生成ID → 发Kafka → 更新推文点赞数缓存
// 2. 第二次操作：发Kafka → 更新推文点赞数缓存
// 用户点赞关系由前端本地存储，登录时通过 GetUserAllLikes 获取
func (l *LikeTweetLogic) LikeTweet(in *pb.LikeTweetReq) (*pb.LikeTweetResp, error) {
	now := time.Now().UnixMilli()

	var snowLikesId int64

	if in.IsCreated == 0 {
		// 第一次操作：生成新的点赞记录
		newId, err := idgen.GenID()
		if err != nil {
			logx.Errorf("LikeTweet generate snowflake id errorx: %v", err)
			return &pb.LikeTweetResp{
				Code: 120401,
				Msg:  "生成点赞ID失败",
			}, nil
		}
		snowLikesId = newId
	} else {
		// 第二次操作：直接使用前端传来的 snowLikesId
		snowLikesId = in.SnowLikesId
	}

	// 发送消息到 Kafka（异步落库）
	likeRecord := &model.LikesTweet{
		SnowLikesId: snowLikesId,
		Uid:         in.Uid,
		SnowTid:     in.SnowTid,
		Status:      in.Status,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := l.sendLikeTweetMessage(likeRecord, in.IsCreated == 0); err != nil {
		logx.Errorf("LikeTweet send queue message errorx, err:%v", err)
	}

	// 更新推文点赞数缓存
	delta := 1
	if in.Status == 0 {
		delta = -1
	}
	go l.updateTweetLikeCount(in.SnowTid, delta)

	// 返回点赞信息
	return &pb.LikeTweetResp{
		Code: 0,
		Msg:  "success",
		Like: &pb.LikeTweetInfo{
			SnowLikesId: snowLikesId,
			Uid:         in.Uid,
			SnowTid:     in.SnowTid,
			Status:      in.Status,
			UpdateTime:  now,
		},
	}, nil
}

// sendLikeTweetMessage 发送点赞消息到 Kafka
func (l *LikeTweetLogic) sendLikeTweetMessage(like *model.LikesTweet, isNew bool) error {
	action := "update_like_tweet"
	if isNew {
		action = "create_like_tweet"
	}

	message := map[string]interface{}{
		"action":        action,
		"snow_likes_id": like.SnowLikesId,
		"uid":           like.Uid,
		"snow_tid":      like.SnowTid,
		"status":        like.Status,
		"created_at":    like.CreatedAt,
		"updated_at":    like.UpdatedAt,
	}

	body, err := json.Marshal(message)
	if err != nil {
		return err
	}

	pusher := l.svcCtx.GetPusher("like_tweet")
	return pusher.PushWithKey(l.ctx, fmt.Sprintf("%d", like.SnowLikesId), string(body))
}

// updateTweetLikeCount 更新推文点赞数缓存
func (l *LikeTweetLogic) updateTweetLikeCount(snowTid int64, delta int) {
	_, err := l.svcCtx.CacheManager.HIncrBy(l.ctx, "tweet", "info", snowTid, "like_count", delta)
	if err != nil {
		logx.Errorf("updateTweetLikeCount errorx, snowTid:%d, delta:%d, err:%v", snowTid, delta, err)
	}
}
