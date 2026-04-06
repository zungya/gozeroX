package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"gozeroX/app/interactService/model"
	"strconv"
	"time"

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
	// 1. 将前端传来的string类型的snow_cid转换为int64
	snowCid, err := strconv.ParseInt(in.SnowCid, 10, 64)
	if err != nil {
		logx.Errorf("DeleteComment parse snow_cid errorx: %v", err)
		return nil, fmt.Errorf("评论ID格式错误: %v", err)
	}

	data, err := l.svcCtx.CommentModel.FindOneBySnowCid(l.ctx, snowCid)
	if err != nil {
		return nil, fmt.Errorf("删除的评论再数据库中不存在: %v", err)
	}

	// 准备更新的数据
	newData := &model.Comment{
		Cid:        data.Cid,
		Tid:        data.Tid,
		Uid:        data.Uid,
		ParentId:   data.ParentId,
		RootId:     data.RootId,
		Content:    data.Content,
		LikeCount:  data.LikeCount,
		ReplyCount: data.ReplyCount,
		Status:     1, // 只更新status
		SnowCid:    data.SnowCid,
	}

	// 2. 先更新数据库（软删除）

	err = l.svcCtx.CommentModel.Update(l.ctx, newData) // status=1表示删除
	if err != nil {
		logx.Errorf("DeleteComment update db errorx, snowCid:%d, err:%v", snowCid, err)
		return nil, fmt.Errorf("删除评论失败: %v", err)
	}

	// 3. 获取评论信息（直接从数据库查，因为刚更新完）
	comment, err := l.svcCtx.CommentModel.FindOneBySnowCid(l.ctx, snowCid)
	if err != nil {
		logx.Errorf("DeleteComment get comment from db errorx, snowCid:%d, err:%v", snowCid, err)
	} else {
		// 4. 更新缓存中的status为1（不删缓存，只更新状态）
		if err := l.svcCtx.SetCommentToCache(l.ctx, snowCid, comment); err != nil {
			logx.Errorf("DeleteComment update cache errorx, snowCid:%d, err:%v", snowCid, err)
			// 缓存更新失败，但数据库已更新，可以返回成功
			// 后续有缓存会通过消息队列补偿
			go l.sendStatusSyncMessage(snowCid, 1)
		}

		// 5. 异步更新计数（推文评论数减1）
		if err := l.svcCtx.IncrTweetCommentCount(l.ctx, comment.Tid, 1); err != nil {
			logx.Errorf("CreateComment incr comment reply count errorx, commentid:%d, err:%v", comment.Tid, err)
		}
	}

	// 6. 返回成功
	return &pb.DeleteCommentResp{
		Success: true,
		SnowCid: in.SnowCid,
	}, nil
}

// sendStatusSyncMessage 发送状态同步消息（用于缓存更新失败时的补偿）
func (l *DeleteCommentLogic) sendStatusSyncMessage(snowCid int64, status int64) {
	message := map[string]interface{}{
		"action":      "sync_comment_status",
		"snow_cid":    snowCid,
		"status":      status,
		"update_time": time.Now().Unix(),
	}

	body, _ := json.Marshal(message)
	pusher := l.svcCtx.GetPusher("comment_status_sync")

	err := pusher.PushWithKey(context.Background(), strconv.FormatInt(snowCid, 10), string(body))
	if err != nil {
		logx.Errorf("sendStatusSyncMessage errorx, snowCid:%d, err:%v", snowCid, err)
	}
}
