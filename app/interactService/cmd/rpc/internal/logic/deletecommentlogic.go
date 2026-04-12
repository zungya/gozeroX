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

// DeleteComment 删除评论/回复（软删除：先更新DB，再更新缓存中的status）
// snow_cid 可能在 comment 表或 reply 表
func (l *DeleteCommentLogic) DeleteComment(in *pb.DeleteCommentReq) (*pb.DeleteCommentResp, error) {
	// 1. 尝试从 comment 表查找
	comment, err := l.svcCtx.GetCommentBySnowCid(l.ctx, in.SnowCid)
	if err == nil && comment != nil && comment.Status == 0 {
		// 权限校验
		if comment.Uid != in.Uid {
			return &pb.DeleteCommentResp{Code: 120202, Msg: "无权删除此评论", Success: false}, nil
		}
		return l.deleteRootComment(in, comment)
	}

	// 2. 尝试从 reply 表查找
	reply, err := l.svcCtx.GetReplyBySnowCid(l.ctx, in.SnowCid)
	if err == nil && reply != nil && reply.Status == 0 {
		// 权限校验
		if reply.Uid != in.Uid {
			return &pb.DeleteCommentResp{Code: 120202, Msg: "无权删除此评论", Success: false}, nil
		}
		return l.deleteReply(in, reply)
	}

	// 3. 都找不到
	return &pb.DeleteCommentResp{Code: 120201, Msg: "评论不存在", Success: false}, nil
}

// deleteRootComment 删除根评论
func (l *DeleteCommentLogic) deleteRootComment(in *pb.DeleteCommentReq, comment *model.Comment) (*pb.DeleteCommentResp, error) {
	now := time.Now().UnixMilli()
	newData := &model.Comment{
		SnowCid:    comment.SnowCid,
		Cid:        comment.Cid,
		SnowTid:    comment.SnowTid,
		Uid:        comment.Uid,
		Content:    comment.Content,
		LikeCount:  comment.LikeCount,
		ReplyCount: comment.ReplyCount,
		Status:     1, // 软删除
		CreatedAt:  comment.CreatedAt,
		UpdatedAt:  now,
	}

	// 先更新数据库
	if err := l.svcCtx.CommentModel.Update(l.ctx, newData); err != nil {
		logx.Errorf("deleteRootComment update db error, snowCid:%d, err:%v", in.SnowCid, err)
		return &pb.DeleteCommentResp{Code: 120203, Msg: "删除评论失败", Success: false}, nil
	}

	// 更新缓存
	if err := l.svcCtx.SetCommentToCache(l.ctx, in.SnowCid, newData); err != nil {
		logx.Errorf("deleteRootComment update cache error, snowCid:%d, err:%v", in.SnowCid, err)
		go l.sendStatusSyncMessage(in.SnowCid, 1)
	}

	// 更新计数（Redis + DB）
	go func() {
		// Redis: 减少推文评论数
		if err := l.svcCtx.IncrTweetCommentCount(l.ctx, comment.SnowTid, -1); err != nil {
			logx.Errorf("deleteRootComment decr tweet comment count cache error, snowTid:%d, err:%v", comment.SnowTid, err)
		}
		// DB: 减少推文评论数
		if _, err := l.svcCtx.ContentServiceRpc.UpdateTweetStats(l.ctx, &content.UpdateTweetStatsReq{
			SnowTid:    comment.SnowTid,
			UpdateType: 2,
			Delta:      -1,
		}); err != nil {
			logx.Errorf("deleteRootComment decr tweet comment count DB error, snowTid:%d, err:%v", comment.SnowTid, err)
		}
	}()

	return &pb.DeleteCommentResp{Code: 0, Msg: "success", Success: true}, nil
}

// deleteReply 删除回复
func (l *DeleteCommentLogic) deleteReply(in *pb.DeleteCommentReq, reply *model.Reply) (*pb.DeleteCommentResp, error) {
	now := time.Now().UnixMilli()
	newData := &model.Reply{
		SnowCid:    reply.SnowCid,
		Cid:        reply.Cid,
		SnowTid:    reply.SnowTid,
		Uid:        reply.Uid,
		ParentId:   reply.ParentId,
		RootId:     reply.RootId,
		Content:    reply.Content,
		LikeCount:  reply.LikeCount,
		ReplyCount: reply.ReplyCount,
		Status:     1, // 软删除
		CreatedAt:  reply.CreatedAt,
		UpdatedAt:  now,
	}

	// 先更新数据库
	if err := l.svcCtx.ReplyModel.Update(l.ctx, newData); err != nil {
		logx.Errorf("deleteReply update db error, snowCid:%d, err:%v", in.SnowCid, err)
		return &pb.DeleteCommentResp{Code: 120203, Msg: "删除评论失败", Success: false}, nil
	}

	// 更新缓存
	if err := l.svcCtx.SetReplyToCache(l.ctx, in.SnowCid, newData); err != nil {
		logx.Errorf("deleteReply update cache error, snowCid:%d, err:%v", in.SnowCid, err)
	}

	// 更新计数（Redis + DB）
	go func() {
		// Redis: 减少推文评论数
		if err := l.svcCtx.IncrTweetCommentCount(l.ctx, reply.SnowTid, -1); err != nil {
			logx.Errorf("deleteReply decr tweet comment count cache error, snowTid:%d, err:%v", reply.SnowTid, err)
		}
		// Redis: 减少父评论回复数
		if err := l.svcCtx.IncrCommentReplyCount(l.ctx, reply.ParentId, -1); err != nil {
			logx.Errorf("deleteReply decr comment reply count cache error, parentId:%d, err:%v", reply.ParentId, err)
		}
		// DB: 减少推文评论数
		if _, err := l.svcCtx.ContentServiceRpc.UpdateTweetStats(l.ctx, &content.UpdateTweetStatsReq{
			SnowTid:    reply.SnowTid,
			UpdateType: 2,
			Delta:      -1,
		}); err != nil {
			logx.Errorf("deleteReply decr tweet comment count DB error, snowTid:%d, err:%v", reply.SnowTid, err)
		}
		// DB: 减少父评论回复数
		if err := l.svcCtx.CommentModel.UpdateCount(l.ctx, reply.ParentId, 2, -1); err != nil {
			logx.Errorf("deleteReply decr comment reply count DB error, parentId:%d, err:%v", reply.ParentId, err)
		}
	}()

	return &pb.DeleteCommentResp{Code: 0, Msg: "success", Success: true}, nil
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
