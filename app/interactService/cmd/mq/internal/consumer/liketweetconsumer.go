package consumer

import (
	"context"
	"encoding/json"
	"fmt"

	"gozeroX/app/contentService/cmd/rpc/content"
	"gozeroX/app/interactService/cmd/mq/internal/svc"
	"gozeroX/app/interactService/model"

	"github.com/zeromicro/go-zero/core/logx"
)

// LikeTweetConsumer 推文点赞消息消费者
type LikeTweetConsumer struct {
	svcCtx *svc.ServiceContext
}

func NewLikeTweetConsumer(svcCtx *svc.ServiceContext) *LikeTweetConsumer {
	return &LikeTweetConsumer{svcCtx: svcCtx}
}

func (c *LikeTweetConsumer) Consume(ctx context.Context, key, value string) error {
	logx.Infof("LikeTweetConsumer received message, key:%s", key)

	var msg map[string]interface{}
	if err := json.Unmarshal([]byte(value), &msg); err != nil {
		logx.Errorf("LikeTweetConsumer json unmarshal error: %v, value:%s", err, value)
		return err
	}

	action, _ := msg["action"].(string)
	switch action {
	case "create_like_tweet":
		return c.handleCreate(ctx, msg)
	case "update_like_tweet":
		return c.handleUpdate(ctx, msg)
	default:
		logx.Infof("LikeTweetConsumer unknown action: %s", action)
		return fmt.Errorf("unknown action: %s", action)
	}
}

func (c *LikeTweetConsumer) handleCreate(ctx context.Context, msg map[string]interface{}) error {
	like := &model.LikesTweet{
		SnowLikesId: toInt64(msg["snow_likes_id"]),
		Uid:         toInt64(msg["uid"]),
		SnowTid:     toInt64(msg["snow_tid"]),
		Status:      toInt64(msg["status"]),
	}

	_, err := c.svcCtx.LikesTweetModel.Insert(ctx, like)
	if err != nil {
		logx.Errorf("LikeTweetConsumer insert error, snowLikesId:%d, err:%v", like.SnowLikesId, err)
		return err
	}

	logx.Infof("LikeTweetConsumer insert success, snowLikesId:%d, uid:%d, snowTid:%d", like.SnowLikesId, like.Uid, like.SnowTid)

	// 更新推文的点赞计数（新建 = +1）
	c.updateTweetLikeCount(ctx, toInt64(msg["snow_tid"]), 1)

	// 更新用户最后点赞时间（用于增量同步优化）
	c.updateUserLikeSync(ctx, msg)

	return nil
}

func (c *LikeTweetConsumer) handleUpdate(ctx context.Context, msg map[string]interface{}) error {
	like := &model.LikesTweet{
		SnowLikesId: toInt64(msg["snow_likes_id"]),
		Uid:         toInt64(msg["uid"]),
		SnowTid:     toInt64(msg["snow_tid"]),
		Status:      toInt64(msg["status"]),
	}

	err := c.svcCtx.LikesTweetModel.Update(ctx, like)
	if err != nil {
		logx.Errorf("LikeTweetConsumer update error, snowLikesId:%d, err:%v", like.SnowLikesId, err)
		return err
	}

	logx.Infof("LikeTweetConsumer update success, snowLikesId:%d, status:%d", like.SnowLikesId, like.Status)

	// 更新推文的点赞计数（status=1 → +1，status=0 → -1）
	delta := int64(1)
	if toInt64(msg["status"]) == 0 {
		delta = -1
	}
	c.updateTweetLikeCount(ctx, toInt64(msg["snow_tid"]), delta)

	// 更新用户最后点赞时间（用于增量同步优化）
	c.updateUserLikeSync(ctx, msg)

	return nil
}

// updateUserLikeSync 更新用户最后点赞时间
func (c *LikeTweetConsumer) updateUserLikeSync(ctx context.Context, msg map[string]interface{}) {
	uid := toInt64(msg["uid"])
	updatedAt := toInt64(msg["updated_at"])
	if uid == 0 || updatedAt == 0 {
		return
	}
	if err := c.svcCtx.UserLikeSyncModel.Upsert(ctx, uid, updatedAt); err != nil {
		logx.Errorf("LikeTweetConsumer updateUserLikeSync error, uid:%d, err:%v", uid, err)
	}
}

// updateTweetLikeCount 通过 RPC 更新推文的点赞计数
func (c *LikeTweetConsumer) updateTweetLikeCount(ctx context.Context, snowTid int64, delta int64) {
	if snowTid == 0 {
		return
	}
	resp, err := c.svcCtx.ContentServiceRpc.UpdateTweetStats(ctx, &content.UpdateTweetStatsReq{
		SnowTid:    snowTid,
		UpdateType: 1, // 1=like_count
		Delta:      delta,
	})
	if err != nil {
		logx.Errorf("LikeTweetConsumer updateTweetLikeCount RPC error, snowTid:%d, delta:%d, err:%v", snowTid, delta, err)
		return
	}
	if resp.Code != 0 {
		logx.Errorf("LikeTweetConsumer updateTweetLikeCount RPC failed, snowTid:%d, delta:%d, code:%d, msg:%s", snowTid, delta, resp.Code, resp.Msg)
	}
}
