package consumer

import (
	"context"
	"encoding/json"
	"fmt"

	"gozeroX/app/interactService/cmd/mq/internal/svc"
	"gozeroX/app/interactService/model"

	"github.com/zeromicro/go-zero/core/logx"
)

// LikeCommentConsumer 评论点赞消息消费者
type LikeCommentConsumer struct {
	svcCtx *svc.ServiceContext
}

func NewLikeCommentConsumer(svcCtx *svc.ServiceContext) *LikeCommentConsumer {
	return &LikeCommentConsumer{svcCtx: svcCtx}
}

func (c *LikeCommentConsumer) Consume(ctx context.Context, key, value string) error {
	logx.Infof("LikeCommentConsumer received message, key:%s", key)

	var msg map[string]interface{}
	if err := json.Unmarshal([]byte(value), &msg); err != nil {
		logx.Errorf("LikeCommentConsumer json unmarshal error: %v, value:%s", err, value)
		return err
	}

	action, _ := msg["action"].(string)
	switch action {
	case "create_like_comment":
		return c.handleCreate(ctx, msg)
	case "update_like_comment":
		return c.handleUpdate(ctx, msg)
	default:
		logx.Errorf("LikeCommentConsumer unknown action: %s", action)
		return fmt.Errorf("unknown action: %s", action)
	}
}

func (c *LikeCommentConsumer) handleCreate(ctx context.Context, msg map[string]interface{}) error {
	like := &model.LikesComment{
		SnowLikesId: toInt64(msg["snow_likes_id"]),
		Uid:         toInt64(msg["uid"]),
		SnowCid:     toInt64(msg["snow_cid"]),
		SnowTid:     toInt64(msg["snow_tid"]),
		Status:      toInt64(msg["status"]),
	}

	_, err := c.svcCtx.LikesCommentModel.Insert(ctx, like)
	if err != nil {
		logx.Errorf("LikeCommentConsumer insert error, snowLikesId:%d, err:%v", like.SnowLikesId, err)
		return err
	}

	logx.Infof("LikeCommentConsumer insert success, snowLikesId:%d, uid:%d, snowCid:%d", like.SnowLikesId, like.Uid, like.SnowCid)

	// 更新评论的点赞计数（新建 = +1）
	c.updateCommentLikeCount(ctx, toInt64(msg["snow_cid"]), 1)

	// 更新用户最后点赞时间（用于增量同步优化）
	c.updateUserLikeSync(ctx, msg)

	return nil
}

func (c *LikeCommentConsumer) handleUpdate(ctx context.Context, msg map[string]interface{}) error {
	like := &model.LikesComment{
		SnowLikesId: toInt64(msg["snow_likes_id"]),
		Uid:         toInt64(msg["uid"]),
		SnowCid:     toInt64(msg["snow_cid"]),
		SnowTid:     toInt64(msg["snow_tid"]),
		Status:      toInt64(msg["status"]),
	}

	err := c.svcCtx.LikesCommentModel.Update(ctx, like)
	if err != nil {
		logx.Errorf("LikeCommentConsumer update error, snowLikesId:%d, err:%v", like.SnowLikesId, err)
		return err
	}

	logx.Infof("LikeCommentConsumer update success, snowLikesId:%d, status:%d", like.SnowLikesId, like.Status)

	// 更新评论的点赞计数（status=1 → +1，status=0 → -1）
	delta := int64(1)
	if toInt64(msg["status"]) == 0 {
		delta = -1
	}
	c.updateCommentLikeCount(ctx, toInt64(msg["snow_cid"]), delta)

	// 更新用户最后点赞时间（用于增量同步优化）
	c.updateUserLikeSync(ctx, msg)

	return nil
}

// updateCommentLikeCount 更新评论的点赞计数（先尝试 comment 表，再尝试 reply 表）
func (c *LikeCommentConsumer) updateCommentLikeCount(ctx context.Context, snowCid int64, delta int64) {
	if snowCid == 0 {
		return
	}
	// 先尝试更新 comment 表
	err := c.svcCtx.CommentModel.UpdateCount(ctx, snowCid, 1, delta)
	if err != nil {
		// comment 表没找到，尝试 reply 表
		err = c.svcCtx.ReplyModel.UpdateCount(ctx, snowCid, 1, delta)
		if err != nil {
			logx.Errorf("LikeCommentConsumer updateCommentLikeCount error, snowCid:%d, delta:%d, err:%v", snowCid, delta, err)
		}
	}
}

// updateUserLikeSync 更新用户最后点赞时间
func (c *LikeCommentConsumer) updateUserLikeSync(ctx context.Context, msg map[string]interface{}) {
	uid := toInt64(msg["uid"])
	updatedAt := toInt64(msg["updated_at"])
	if uid == 0 || updatedAt == 0 {
		return
	}
	if err := c.svcCtx.UserLikeSyncModel.Upsert(ctx, uid, updatedAt); err != nil {
		logx.Errorf("LikeCommentConsumer updateUserLikeSync error, uid:%d, err:%v", uid, err)
	}
}
