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

// CommentConsumer 评论创建消息消费者
type CommentConsumer struct {
	svcCtx *svc.ServiceContext
}

func NewCommentConsumer(svcCtx *svc.ServiceContext) *CommentConsumer {
	return &CommentConsumer{svcCtx: svcCtx}
}

func (c *CommentConsumer) Consume(ctx context.Context, key, value string) error {
	logx.Infof("CommentConsumer received message, key:%s", key)

	var msg map[string]interface{}
	if err := json.Unmarshal([]byte(value), &msg); err != nil {
		logx.Errorf("CommentConsumer json unmarshal error: %v, value:%s", err, value)
		return err
	}

	action, _ := msg["action"].(string)
	switch action {
	case "create_comment":
		return c.handleCreateComment(ctx, msg)
	case "create_reply":
		return c.handleCreateReply(ctx, msg)
	default:
		logx.Infof("CommentConsumer unknown action: %s", action)
		return fmt.Errorf("unknown action: %s", action)
	}
}

// handleCreateComment 处理根评论创建（无 parent_id/root_id）
func (c *CommentConsumer) handleCreateComment(ctx context.Context, msg map[string]interface{}) error {
	comment := &model.Comment{
		SnowCid:    toInt64(msg["snow_cid"]),
		Cid:        0,
		SnowTid:    toInt64(msg["snow_tid"]),
		Uid:        toInt64(msg["uid"]),
		Content:    toString(msg["content"]),
		LikeCount:  toInt64(msg["like_count"]),
		ReplyCount: toInt64(msg["reply_count"]),
		Status:     toInt64(msg["status"]),
	}

	_, err := c.svcCtx.CommentModel.Insert(ctx, comment)
	if err != nil {
		logx.Errorf("CommentConsumer insert comment error, snowCid:%d, err:%v", comment.SnowCid, err)
		return err
	}

	logx.Infof("CommentConsumer insert comment success, snowCid:%d, snowTid:%d", comment.SnowCid, comment.SnowTid)

	// 更新推文评论数（DB）
	c.updateTweetCommentCount(ctx, comment.SnowTid, 1)

	return nil
}

// handleCreateReply 处理回复创建
func (c *CommentConsumer) handleCreateReply(ctx context.Context, msg map[string]interface{}) error {
	reply := &model.Reply{
		SnowCid:    toInt64(msg["snow_cid"]),
		Cid:        0,
		SnowTid:    toInt64(msg["snow_tid"]),
		Uid:        toInt64(msg["uid"]),
		ParentId:   toInt64(msg["parent_id"]),
		RootId:     toInt64(msg["root_id"]),
		Content:    toString(msg["content"]),
		LikeCount:  toInt64(msg["like_count"]),
		ReplyCount: toInt64(msg["reply_count"]),
		Status:     toInt64(msg["status"]),
	}

	_, err := c.svcCtx.ReplyModel.Insert(ctx, reply)
	if err != nil {
		logx.Errorf("CommentConsumer insert reply error, snowCid:%d, err:%v", reply.SnowCid, err)
		return err
	}

	logx.Infof("CommentConsumer insert reply success, snowCid:%d, snowTid:%d, rootId:%d", reply.SnowCid, reply.SnowTid, reply.RootId)

	// 更新推文评论数（DB）
	c.updateTweetCommentCount(ctx, reply.SnowTid, 1)
	// 更新父评论的回复数（DB）
	c.updateParentReplyCount(ctx, reply.ParentId, 1)
	// 如果是子评论的回复（RootId != 0 且 RootId != ParentId），根评论回复数也 +1
	if reply.RootId != 0 && reply.RootId != reply.ParentId {
		c.updateParentReplyCount(ctx, reply.RootId, 1)
	}

	return nil
}

// updateTweetCommentCount 通过 RPC 更新推文的评论计数
func (c *CommentConsumer) updateTweetCommentCount(ctx context.Context, snowTid int64, delta int64) {
	if snowTid == 0 {
		return
	}
	resp, err := c.svcCtx.ContentServiceRpc.UpdateTweetStats(ctx, &content.UpdateTweetStatsReq{
		SnowTid:    snowTid,
		UpdateType: 2, // 2=comment_count
		Delta:      delta,
	})
	if err != nil {
		logx.Errorf("CommentConsumer updateTweetCommentCount RPC error, snowTid:%d, delta:%d, err:%v", snowTid, delta, err)
		return
	}
	if resp.Code != 0 {
		logx.Errorf("CommentConsumer updateTweetCommentCount RPC failed, snowTid:%d, delta:%d, code:%d, msg:%s", snowTid, delta, resp.Code, resp.Msg)
	}
}

// updateParentReplyCount 更新父评论的回复计数
func (c *CommentConsumer) updateParentReplyCount(ctx context.Context, parentSnowCid int64, delta int64) {
	if parentSnowCid == 0 {
		return
	}
	if err := c.svcCtx.CommentModel.UpdateCount(ctx, parentSnowCid, 2, delta); err != nil {
		logx.Errorf("CommentConsumer updateParentReplyCount error, parentSnowCid:%d, delta:%d, err:%v", parentSnowCid, delta, err)
	}
}
