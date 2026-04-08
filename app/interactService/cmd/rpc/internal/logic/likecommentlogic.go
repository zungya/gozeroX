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
			logx.Errorf("LikeComment generate snowflake id errorx: %v", err)
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
		Status:      in.Status,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := l.sendLikeCommentMessage(likeRecord, in.IsCreated == 0); err != nil {
		logx.Errorf("LikeComment send queue message errorx, err:%v", err)
	}

	// 更新评论点赞数缓存
	delta := 1
	if in.Status == 0 {
		delta = -1
	}
	go l.updateCommentLikeCount(in.SnowCid, delta)

	// 返回点赞信息
	return &pb.LikeCommentResp{
		Code: 0,
		Msg:  "success",
		Like: &pb.LikeCommentInfo{
			SnowLikesId: snowLikesId,
			Uid:         in.Uid,
			SnowCid:     in.SnowCid,
			Status:      in.Status,
			UpdateTime:  now,
		},
	}, nil
}

// sendLikeCommentMessage 发送点赞消息到 Kafka
func (l *LikeCommentLogic) sendLikeCommentMessage(like *model.LikesComment, isNew bool) error {
	action := "update_like_comment"
	if isNew {
		action = "create_like_comment"
	}

	message := map[string]interface{}{
		"action":        action,
		"snow_likes_id": like.SnowLikesId,
		"uid":           like.Uid,
		"snow_cid":      like.SnowCid,
		"status":        like.Status,
		"created_at":    like.CreatedAt,
		"updated_at":    like.UpdatedAt,
	}

	body, err := json.Marshal(message)
	if err != nil {
		return err
	}

	pusher := l.svcCtx.GetPusher("like_comment")
	return pusher.PushWithKey(l.ctx, fmt.Sprintf("%d", like.SnowLikesId), string(body))
}

// updateCommentLikeCount 更新评论点赞数缓存
func (l *LikeCommentLogic) updateCommentLikeCount(snowCid int64, delta int) {
	err := l.svcCtx.IncrCommentLikeCount(l.ctx, snowCid, delta)
	if err != nil {
		logx.Errorf("updateCommentLikeCount errorx, snowCid:%d, delta:%d, err:%v", snowCid, delta, err)
	}
}
