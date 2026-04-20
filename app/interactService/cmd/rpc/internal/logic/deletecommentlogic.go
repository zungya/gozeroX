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

// DeleteComment 删除评论/回复（软删除）
// 根据 is_reply 字段确定目标表：0=comment表，1=reply表
func (l *DeleteCommentLogic) DeleteComment(in *pb.DeleteCommentReq) (*pb.DeleteCommentResp, error) {
	if in.IsReply == 0 {
		return l.deleteRootComment(in)
	}
	return l.deleteReply(in)
}

// deleteRootComment 删除根评论
func (l *DeleteCommentLogic) deleteRootComment(in *pb.DeleteCommentReq) (*pb.DeleteCommentResp, error) {
	comment, err := l.svcCtx.GetCommentBySnowCid(l.ctx, in.SnowCid)
	if err != nil {
		l.Errorf("deleteRootComment GetCommentBySnowCid error, snowCid:%d, err:%v", in.SnowCid, err)
		return &pb.DeleteCommentResp{Code: 120201, Msg: "评论不存在", Success: false}, nil
	}
	if comment.Status != 0 {
		return &pb.DeleteCommentResp{Code: 120201, Msg: "评论不存在", Success: false}, nil
	}
	if comment.Uid != in.Uid {
		return &pb.DeleteCommentResp{Code: 120202, Msg: "无权删除此评论", Success: false}, nil
	}

	now := time.Now().UnixMilli()
	newData := &model.Comment{
		SnowCid:    comment.SnowCid,
		Cid:        comment.Cid,
		SnowTid:    comment.SnowTid,
		Uid:        comment.Uid,
		Content:    comment.Content,
		LikeCount:  comment.LikeCount,
		ReplyCount: comment.ReplyCount,
		Status:     1,
		CreatedAt:  comment.CreatedAt,
		UpdatedAt:  now,
	}

	if err := l.svcCtx.CommentModel.Update(l.ctx, newData); err != nil {
		l.Errorf("deleteRootComment CommentModel.Update error, snowCid:%d, err:%v", in.SnowCid, err)
		return &pb.DeleteCommentResp{Code: 120203, Msg: "删除评论失败", Success: false}, nil
	}

	if err := l.svcCtx.SetCommentToCache(l.ctx, in.SnowCid, newData); err != nil {
		go l.sendStatusSyncMessage(in.SnowCid, 1)
	}

	go func() {
		if err := l.svcCtx.IncrTweetCommentCount(context.Background(), comment.SnowTid, -1); err != nil {
			l.Errorf("deleteRootComment decr tweet comment count cache error, snowTid:%d, err:%v", comment.SnowTid, err)
		}
		if _, err := l.svcCtx.ContentServiceRpc.UpdateTweetStats(context.Background(), &content.UpdateTweetStatsReq{
			SnowTid: comment.SnowTid, UpdateType: 2, Delta: -1,
		}); err != nil {
			l.Errorf("deleteRootComment decr tweet comment count DB error, snowTid:%d, err:%v", comment.SnowTid, err)
		}
	}()

	return &pb.DeleteCommentResp{Code: 0, Msg: "success", Success: true}, nil
}

// deleteReply 删除回复
func (l *DeleteCommentLogic) deleteReply(in *pb.DeleteCommentReq) (*pb.DeleteCommentResp, error) {
	reply, err := l.svcCtx.GetReplyBySnowCid(l.ctx, in.SnowCid)
	if err != nil {
		l.Errorf("deleteReply GetReplyBySnowCid error, snowCid:%d, err:%v", in.SnowCid, err)
		return &pb.DeleteCommentResp{Code: 120201, Msg: "回复不存在", Success: false}, nil
	}
	if reply.Status != 0 {
		return &pb.DeleteCommentResp{Code: 120201, Msg: "回复不存在", Success: false}, nil
	}
	if reply.Uid != in.Uid {
		return &pb.DeleteCommentResp{Code: 120202, Msg: "无权删除此回复", Success: false}, nil
	}

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
		Status:     1,
		CreatedAt:  reply.CreatedAt,
		UpdatedAt:  now,
	}

	if err := l.svcCtx.ReplyModel.Update(l.ctx, newData); err != nil {
		l.Errorf("deleteReply ReplyModel.Update error, snowCid:%d, err:%v", in.SnowCid, err)
		return &pb.DeleteCommentResp{Code: 120203, Msg: "删除回复失败", Success: false}, nil
	}

	_ = l.svcCtx.SetReplyToCache(l.ctx, in.SnowCid, newData)

	go func() {
		if err := l.svcCtx.IncrTweetCommentCount(context.Background(), reply.SnowTid, -1); err != nil {
			l.Errorf("deleteReply decr tweet comment count cache error, snowTid:%d, err:%v", reply.SnowTid, err)
		}
		if err := l.svcCtx.IncrCommentReplyCount(context.Background(), reply.ParentId, -1); err != nil {
			l.Errorf("deleteReply decr comment reply count cache error, parentId:%d, err:%v", reply.ParentId, err)
		}
		if _, err := l.svcCtx.ContentServiceRpc.UpdateTweetStats(context.Background(), &content.UpdateTweetStatsReq{
			SnowTid: reply.SnowTid, UpdateType: 2, Delta: -1,
		}); err != nil {
			l.Errorf("deleteReply decr tweet comment count DB error, snowTid:%d, err:%v", reply.SnowTid, err)
		}
		if err := l.svcCtx.CommentModel.UpdateCount(context.Background(), reply.ParentId, 2, -1); err != nil {
			l.Errorf("deleteReply decr comment reply count DB error, parentId:%d, err:%v", reply.ParentId, err)
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
		l.Errorf("sendStatusSyncMessage errorx, snowCid:%d, err:%v", snowCid, err)
	}
}
