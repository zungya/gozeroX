package consumer

import (
	"context"
	"encoding/json"
	"fmt"

	"gozeroX/app/noticeService/cmd/mq/internal/svc"
	"gozeroX/app/noticeService/model"
	"gozeroX/pkg/idgen"

	"github.com/zeromicro/go-zero/core/logx"
)

// NoticeConsumer 通知消息消费者（统一处理点赞通知和评论通知）
type NoticeConsumer struct {
	svcCtx *svc.ServiceContext
}

func NewNoticeConsumer(svcCtx *svc.ServiceContext) *NoticeConsumer {
	return &NoticeConsumer{svcCtx: svcCtx}
}

func (c *NoticeConsumer) Consume(ctx context.Context, key, value string) error {
	logx.Infof("NoticeConsumer received message, key:%s", key)

	var msg map[string]interface{}
	if err := json.Unmarshal([]byte(value), &msg); err != nil {
		logx.Errorf("NoticeConsumer json unmarshal error: %v, value:%s", err, value)
		return err
	}

	action, _ := msg["action"].(string)
	switch action {
	// 点赞通知（聚合处理）
	case "like_tweet", "like_comment":
		return c.handleLike(ctx, msg)
	case "cancel_like_tweet", "cancel_like_comment":
		return c.handleCancelLike(ctx, msg)
	// 评论通知（逐条插入）
	case "comment_tweet", "reply_comment":
		return c.handleComment(ctx, msg)
	default:
		logx.Errorf("NoticeConsumer unknown action: %s", action)
		return fmt.Errorf("unknown action: %s", action)
	}
}

// ==================== 点赞通知（聚合） ====================

func (c *NoticeConsumer) handleLike(ctx context.Context, msg map[string]interface{}) error {
	recipientUid := toInt64(msg["recipient_uid"])
	targetType := toInt64(msg["target_type"])
	targetId := toInt64(msg["target_id"])
	likerUid := toInt64(msg["liker_uid"])
	snowTid := toInt64(msg["snow_tid"])
	rootId := toInt64(msg["root_id"])
	timestamp := toInt64(msg["timestamp"])

	// 1. 查找该 target+接收者是否已存在通知
	existing, err := c.svcCtx.NoticeLikeModel.FindByUidAndTarget(ctx, recipientUid, targetType, targetId)
	if err != nil && err != model.ErrNotFound {
		logx.Errorf("NoticeConsumer handleLike FindByUidAndTarget error: %v", err)
		return err
	}

	if existing != nil {
		// 2a. 已存在：更新 recent_uids（新的推到前面）、total_count 和 recent_count
		newRecent1 := likerUid
		newRecent2 := existing.RecentUid1
		newTotal := existing.TotalCount + 1
		newRecentCount := existing.RecentCount + 1
		return c.svcCtx.NoticeLikeModel.UpdateAggregation(ctx, existing.SnowNid, newRecent1, newRecent2, newTotal, newRecentCount)
	}

	// 2b. 不存在：创建新的聚合记录
	snowNid, err := idgen.GenID()
	if err != nil {
		logx.Errorf("NoticeConsumer handleLike GenID error: %v", err)
		return err
	}

	notice := &model.NoticeLike{
		SnowNid:     snowNid,
		TargetType:  targetType,
		TargetId:    targetId,
		SnowTid:     snowTid,
		RootId:      rootId,
		RecentUid1:  likerUid,
		RecentUid2:  0,
		Uid:         recipientUid,
		TotalCount:  1,
		RecentCount: 1,
		IsRead:      0,
		CreatedAt:   timestamp,
		UpdatedAt:   timestamp,
		Status:      0,
	}

	_, err = c.svcCtx.NoticeLikeModel.Insert(ctx, notice)
	if err != nil {
		// 唯一索引冲突（并发场景）：回退到查找+更新
		logx.Errorf("NoticeConsumer handleLike Insert error (may be unique conflict): %v", err)
		existing, findErr := c.svcCtx.NoticeLikeModel.FindByUidAndTarget(ctx, recipientUid, targetType, targetId)
		if findErr != nil {
			return findErr
		}
		newRecent1 := likerUid
		newRecent2 := existing.RecentUid1
		newTotal := existing.TotalCount + 1
		newRecentCount := existing.RecentCount + 1
		return c.svcCtx.NoticeLikeModel.UpdateAggregation(ctx, existing.SnowNid, newRecent1, newRecent2, newTotal, newRecentCount)
	}

	logx.Infof("NoticeConsumer handleLike success, recipientUid:%d, targetType:%d, targetId:%d", recipientUid, targetType, targetId)
	return nil
}

func (c *NoticeConsumer) handleCancelLike(ctx context.Context, msg map[string]interface{}) error {
	recipientUid := toInt64(msg["recipient_uid"])
	targetType := toInt64(msg["target_type"])
	targetId := toInt64(msg["target_id"])

	existing, err := c.svcCtx.NoticeLikeModel.FindByUidAndTarget(ctx, recipientUid, targetType, targetId)
	if err != nil {
		// 记录不存在，忽略取消操作
		if err == model.ErrNotFound {
			return nil
		}
		return err
	}

	newTotal := existing.TotalCount - 1
	if newTotal < 0 {
		newTotal = 0
	}

	newRecentCount := existing.RecentCount - 1
	if newRecentCount < 0 {
		newRecentCount = 0
	}

	return c.svcCtx.NoticeLikeModel.UpdateAggregation(ctx, existing.SnowNid, existing.RecentUid1, existing.RecentUid2, newTotal, newRecentCount)
}

// ==================== 评论通知（逐条插入） ====================

func (c *NoticeConsumer) handleComment(ctx context.Context, msg map[string]interface{}) error {
	snowNid, err := idgen.GenID()
	if err != nil {
		logx.Errorf("NoticeConsumer handleComment GenID error: %v", err)
		return err
	}

	targetType := toInt64(msg["target_type"])
	timestamp := toInt64(msg["timestamp"])

	notice := &model.NoticeComment{
		SnowNid:        snowNid,
		TargetType:     targetType,
		CommenterUid:   toInt64(msg["commenter_uid"]),
		Uid:            toInt64(msg["recipient_uid"]),
		SnowTid:        toInt64(msg["snow_tid"]),
		SnowCid:        toInt64(msg["snow_cid"]),
		RootId:         toInt64(msg["root_id"]),
		ParentId:       toInt64(msg["parent_id"]),
		Content:        toString(msg["content"]),
		RepliedContent: toString(msg["replied_content"]),
		IsRead:         0,
		CreatedAt:      timestamp,
		UpdatedAt:      timestamp,
		Status:         0,
	}

	_, err = c.svcCtx.NoticeCommentModel.Insert(ctx, notice)
	if err != nil {
		logx.Errorf("NoticeConsumer handleComment Insert error: %v", err)
		return err
	}

	logx.Infof("NoticeConsumer handleComment success, snowNid:%d, commenterUid:%d, recipientUid:%d", snowNid, notice.CommenterUid, notice.Uid)
	return nil
}
