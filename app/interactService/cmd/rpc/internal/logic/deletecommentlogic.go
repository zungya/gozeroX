package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"gozeroX/app/interactService/model"
	"time"

	"gozeroX/app/contentService/cmd/rpc/content"
	"gozeroX/app/interactService/cmd/rpc/internal/svc"
	"gozeroX/app/interactService/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteCommentLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteCommentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteCommentLogic {
	return &DeleteCommentLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// DeleteComment 删除评论（软删除：先更新DB，再更新缓存中的status）
func (l *DeleteCommentLogic) DeleteComment(in *pb.DeleteCommentReq) (*pb.DeleteCommentResp, error) {
	// 1. 查询评论是否存在
	comment, err := l.svcCtx.GetCommentBySnowCid(l.ctx, in.SnowCid)
	if err != nil {
		logx.Errorf("DeleteComment find comment by snowCid %d errorx: %v", in.SnowCid, err)
		return &pb.DeleteCommentResp{
			Code:    120201,
			Msg:     "评论不存在",
			Success: false,
		}, nil
	}

	// 2. 权限校验：只能删除自己的评论
	if comment.Uid != in.Uid {
		return &pb.DeleteCommentResp{
			Code:    120202,
			Msg:     "无权删除此评论",
			Success: false,
		}, nil
	}

	// 3. 准备更新的数据（软删除，status=1）
	now := time.Now().UnixMilli()
	newData := &model.Comment{
		SnowCid:    comment.SnowCid,
		Cid:        comment.Cid,
		SnowTid:    comment.SnowTid,
		Uid:        comment.Uid,
		ParentId:   comment.ParentId,
		RootId:     comment.RootId,
		Content:    comment.Content,
		LikeCount:  comment.LikeCount,
		ReplyCount: comment.ReplyCount,
		Status:     1, // 软删除
		CreatedAt:  comment.CreatedAt,
		UpdatedAt:  now,
	}

	// 4. 先更新数据库
	err = l.svcCtx.CommentModel.Update(l.ctx, newData)
	if err != nil {
		logx.Errorf("DeleteComment update db errorx, snowCid:%d, err:%v", in.SnowCid, err)
		return &pb.DeleteCommentResp{
			Code:    120203,
			Msg:     "删除评论失败",
			Success: false,
		}, nil
	}

	// 5. 更新缓存中的status为1
	if err := l.svcCtx.SetCommentToCache(l.ctx, in.SnowCid, newData); err != nil {
		logx.Errorf("DeleteComment update cache errorx, snowCid:%d, err:%v", in.SnowCid, err)
		// 缓存更新失败，发送补偿消息
		go l.sendStatusSyncMessage(in.SnowCid, 1)
	}

	// 6. 更新相关计数（Redis + DB）
	go func() {
		// --- Redis 缓存计数更新 ---
		// 减少推文评论数
		if err := l.svcCtx.IncrTweetCommentCount(l.ctx, comment.SnowTid, -1); err != nil {
			logx.Errorf("DeleteComment decr tweet comment count cache errorx, snowTid:%d, err:%v", comment.SnowTid, err)
		}
		// 如果是回复，减少父评论的回复数
		if comment.ParentId != 0 {
			if err := l.svcCtx.IncrCommentReplyCount(l.ctx, comment.ParentId, -1); err != nil {
				logx.Errorf("DeleteComment decr comment reply count cache errorx, parentId:%d, err:%v", comment.ParentId, err)
			}
		}

		// --- DB 数据库计数更新 ---
		// 减少推文评论数（通过 contentService RPC）
		if _, err := l.svcCtx.ContentServiceRpc.UpdateTweetStats(l.ctx, &content.UpdateTweetStatsReq{
			SnowTid:    comment.SnowTid,
			UpdateType: 2, // 2=comment_count
			Delta:      -1,
		}); err != nil {
			logx.Errorf("DeleteComment decr tweet comment count DB errorx, snowTid:%d, err:%v", comment.SnowTid, err)
		}
		// 如果是回复，减少父评论的回复数
		if comment.ParentId != 0 {
			if err := l.svcCtx.CommentModel.UpdateCount(l.ctx, comment.ParentId, 2, -1); err != nil {
				logx.Errorf("DeleteComment decr comment reply count DB errorx, parentId:%d, err:%v", comment.ParentId, err)
			}
		}
	}()

	return &pb.DeleteCommentResp{
		Code:    0,
		Msg:     "success",
		Success: true,
	}, nil
}

// sendStatusSyncMessage 发送状态同步消息（用于缓存更新失败时的补偿）
func (l *DeleteCommentLogic) sendStatusSyncMessage(snowCid int64, status int64) {
	message := map[string]interface{}{
		"action":      "sync_comment_status",
		"snow_cid":    snowCid,
		"status":      status,
		"update_time": time.Now().UnixMilli(),
	}

	body, _ := json.Marshal(message)
	pusher := l.svcCtx.GetPusher("comment_status_sync")

	err := pusher.PushWithKey(context.Background(), fmt.Sprintf("%d", snowCid), string(body))
	if err != nil {
		logx.Errorf("sendStatusSyncMessage errorx, snowCid:%d, err:%v", snowCid, err)
	}
}
