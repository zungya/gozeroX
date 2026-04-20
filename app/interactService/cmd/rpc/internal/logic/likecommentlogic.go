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

type LikeCommentLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewLikeCommentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LikeCommentLogic {
	return &LikeCommentLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// LikeComment 评论点赞/取消点赞（write-behind模式）
// 设计原则：
// 1. 第一次操作：生成ID → 发Kafka → 更新评论点赞数缓存
// 2. 第二次操作：发Kafka → 更新评论点赞数缓存
// 用户点赞关系由前端本地存储，登录时通过 GetUserAllLikes 获取
func (l *LikeCommentLogic) LikeComment(in *pb.LikeCommentReq) (*pb.LikeCommentResp, error) {
	now := time.Now().UnixMilli()

	var snowLikesId int64

	if in.IsCreated == 0 {
		// 第一次操作：生成新的点赞记录
		newId, err := idgen.GenID()
		if err != nil {
			l.Errorf("LikeComment generate snowflake id errorx: %v", err)
			return &pb.LikeCommentResp{
				Code: 120501,
				Msg:  "生成点赞ID失败",
			}, nil
		}
		snowLikesId = newId
	} else {
		// 第二次操作：直接使用前端传来的 snowLikesId
		snowLikesId = in.SnowLikesId
	}

	// 发送消息到 Kafka（异步落库）
	likeRecord := &model.LikesComment{
		SnowLikesId: snowLikesId,
		Uid:         in.Uid,
		SnowCid:     in.SnowCid,
		SnowTid:     in.SnowTid,
		Status:      in.Status,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := l.sendLikeCommentMessage(likeRecord, in.IsCreated == 0, in.IsReply); err != nil {
		l.Errorf("LikeComment send queue message errorx, err:%v", err)
	}

	// 更新评论/回复点赞数缓存
	delta := 1
	if in.Status == 0 {
		delta = -1
	}
	go l.updateCommentLikeCount(in.SnowCid, delta, in.IsReply)

	// 异步发送评论点赞通知（不影响主流程）
	go func() {
		defer func() {
			if r := recover(); r != nil {
				l.Errorf("sendLikeCommentNotification panic: %v", r)
			}
		}()
		l.sendLikeCommentNotification(in.Uid, in.SnowCid, in.SnowTid, in.Status, in.IsReply)
	}()

	// 异步发送互动事件到 Kafka（推荐系统用）
	go func() {
		defer func() {
			if r := recover(); r != nil {
				l.Errorf("sendRecommendInteraction panic: %v", r)
			}
		}()
		action := "like_comment"
		if in.Status == 0 {
			action = "cancel_like_comment"
		}
		l.sendRecommendInteraction(action, in.Uid, in.SnowTid, in.SnowCid, "")
	}()

	// 返回点赞信息
	return &pb.LikeCommentResp{
		Code: 0,
		Msg:  "success",
		Like: &pb.LikeCommentInfo{
			SnowLikesId: snowLikesId,
			Uid:         in.Uid,
			SnowCid:     in.SnowCid,
			SnowTid:     in.SnowTid,
			Status:      in.Status,
			UpdateTime:  now,
		},
	}, nil
}

// sendLikeCommentMessage 发送点赞消息到 Kafka
func (l *LikeCommentLogic) sendLikeCommentMessage(like *model.LikesComment, isNew bool, isReply int64) error {
	action := "update_like_comment"
	if isNew {
		action = "create_like_comment"
	}

	message := map[string]interface{}{
		"action":        action,
		"snow_likes_id": like.SnowLikesId,
		"uid":           like.Uid,
		"snow_cid":      like.SnowCid,
		"snow_tid":      like.SnowTid,
		"status":        like.Status,
		"created_at":    like.CreatedAt,
		"updated_at":    like.UpdatedAt,
		"is_reply":      isReply,
	}

	body, err := json.Marshal(message)
	if err != nil {
		return err
	}

	pusher := l.svcCtx.GetPusher("like_comment")
	return pusher.PushWithKey(l.ctx, fmt.Sprintf("%d", like.SnowLikesId), string(body))
}

// updateCommentLikeCount 更新评论/回复点赞数缓存
func (l *LikeCommentLogic) updateCommentLikeCount(snowCid int64, delta int, isReply int64) {
	if isReply == 0 {
		err := l.svcCtx.IncrCommentLikeCount(context.Background(), snowCid, delta)
		if err != nil {
			l.Errorf("updateCommentLikeCount errorx, snowCid:%d, delta:%d, err:%v", snowCid, delta, err)
		}
	} else {
		err := l.svcCtx.IncrReplyLikeCount(context.Background(), snowCid, delta)
		if err != nil {
			l.Errorf("updateReplyLikeCount errorx, snowCid:%d, delta:%d, err:%v", snowCid, delta, err)
		}
	}
}

// sendLikeCommentNotification 异步发送评论点赞通知到 Kafka notice topic
func (l *LikeCommentLogic) sendLikeCommentNotification(likerUid, snowCid, snowTid int64, status int64, isReply int64) {
	var recipientUid int64
	var rootId int64

	if isReply == 0 {
		// 根评论：查 comment 表
		comment, err := l.svcCtx.GetCommentBySnowCid(context.Background(), snowCid)
		if err != nil {
			l.Errorf("sendLikeCommentNotification GetCommentBySnowCid error: %v", err)
			return
		}
		recipientUid = comment.Uid
	} else {
		// 子评论：查 reply 表
		reply, err := l.svcCtx.GetReplyBySnowCid(context.Background(), snowCid)
		if err != nil {
			l.Errorf("sendLikeCommentNotification GetReplyBySnowCid error: %v", err)
			return
		}
		recipientUid = reply.Uid
		rootId = reply.RootId
	}

	// 自己赞自己不发通知
	if recipientUid == likerUid {
		return
	}

	action := "like_comment"
	if status == 0 {
		action = "cancel_like_comment"
	}
	now := time.Now().UnixMilli()
	message := map[string]interface{}{
		"action":        action,
		"target_type":   1,
		"target_id":     snowCid,
		"snow_tid":      snowTid,
		"root_id":       rootId,
		"liker_uid":     likerUid,
		"recipient_uid": recipientUid,
		"timestamp":     now,
	}
	body, err := json.Marshal(message)
	if err != nil {
		l.Errorf("sendLikeCommentNotification marshal error: %v", err)
		return
	}

	pusher := l.svcCtx.GetPusher("notice")
	if err := pusher.PushWithKey(context.Background(), fmt.Sprintf("like_comment_%d_%d", recipientUid, snowCid), string(body)); err != nil {
		l.Errorf("sendLikeCommentNotification push error: %v", err)
	}
}

// sendRecommendInteraction 发送互动事件到 Kafka recommend_interaction topic（推荐系统用）
func (l *LikeCommentLogic) sendRecommendInteraction(action string, uid, snowTid, snowCid int64, content string) {
	message := map[string]interface{}{
		"action":    action,
		"uid":       uid,
		"snow_tid":  snowTid,
		"snow_cid":  snowCid,
		"content":   content,
		"timestamp": time.Now().UnixMilli(),
	}
	body, err := json.Marshal(message)
	if err != nil {
		l.Errorf("sendRecommendInteraction marshal error: %v", err)
		return
	}
	pusher := l.svcCtx.GetPusher("recommend_interaction")
	if err := pusher.PushWithKey(context.Background(), fmt.Sprintf("%d_%d", uid, snowCid), string(body)); err != nil {
		l.Errorf("sendRecommendInteraction push error: %v", err)
	}
}
