package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"gozeroX/app/interactService/model"
	"gozeroX/pkg/idgen"
	"strconv"
	"time"

	"gozeroX/app/interactService/cmd/rpc/internal/svc"
	"gozeroX/app/interactService/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateCommentLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateCommentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateCommentLogic {
	return &CreateCommentLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// CreateComment 创建评论（write-behind模式）
func (l *CreateCommentLogic) CreateComment(in *pb.CreateCommentReq) (*pb.CreateCommentResp, error) {
	// 1. 生成雪花ID作为业务主键
	snowCid, err := idgen.GenID()
	if err != nil {
		logx.Errorf("CreateComment generate snowflake id errorx: %v", err)
		return nil, fmt.Errorf("生成评论ID失败: %v", err)
	}

	// 2. 构建评论对象
	now := time.Now()
	comment := &model.Comment{
		Tid:        in.Tid,
		Uid:        in.Uid,
		ParentId:   in.ParentId,
		Content:    in.Content,
		LikeCount:  0,
		ReplyCount: 0,
		Status:     0,
		CreateTime: now,
		SnowCid:    snowCid,
	}

	// 3. 处理rootId逻辑（使用SnowCid）
	if in.ParentId != 0 {
		// 查询父评论（统一方法：先查缓存，再查DB，然后回写缓存）
		parentComment, err := l.svcCtx.GetCommentBySnowCid(l.ctx, in.ParentId)
		if err != nil {
			logx.Errorf("CreateComment find parent comment by snowCid %d errorx: %v", in.ParentId, err)
			return nil, fmt.Errorf("父评论不存在: %v", err)
		}

		// 父评论的rootId如果是0，说明父评论是顶级评论，则使用父评论的SnowCid作为rootId
		if parentComment.RootId == 0 {
			comment.RootId = parentComment.SnowCid
		} else {
			comment.RootId = parentComment.RootId
		}
	} else {
		comment.RootId = 0 // 顶级评论，rootId为0
	}

	// 4. 先写缓存
	if err := l.svcCtx.SetCommentToCache(l.ctx, snowCid, comment); err != nil {
		logx.Errorf("CreateComment SetCommentToCache errorx, snowCid:%d, err:%v", snowCid, err)
		return nil, fmt.Errorf("缓存评论失败: %v", err)
	}
	// 在写缓存之后，如果是顶级评论，添加到tid的Set
	if in.ParentId == 0 {
		go func() {
			if err := l.svcCtx.IncrTweetCommentCount(l.ctx, in.ParentId, 1); err != nil {
				logx.Errorf("CreateComment incr comment reply count errorx, parentid:%d, err:%v", in.ParentId, err)
			}
			// 使用CacheManager的SAdd方法
			_ = l.svcCtx.CacheManager.SAdd(
				context.Background(),
				"tweet",
				"top_comments",
				in.Tid,
				snowCid, // 直接传int64
			)
		}()
	}

	if in.ParentId != 0 {
		go func() {
			if err := l.svcCtx.IncrCommentReplyCount(l.ctx, in.ParentId, 1); err != nil {
				logx.Errorf("CreateComment incr comment reply count errorx, parentid:%d, err:%v", in.ParentId, err)
			}
			// 使用CacheManager的SAdd方法
			_ = l.svcCtx.CacheManager.SAdd(
				context.Background(),
				"comment",
				"replies",
				in.ParentId, // 父评论的snow_cid
				snowCid,     // 当前回复的snow_cid
			)
			// 设置过期时间（30分钟）
			_ = l.svcCtx.CacheManager.Expire(context.Background(), "comment", "replies", in.ParentId, 1800)
		}()
	}

	// 5. 发送go-queue消息，异步落库
	if err := l.sendCreateCommentMessage(comment); err != nil {
		logx.Errorf("CreateComment send queue message errorx, snowCid:%d, err:%v", snowCid, err)
		go l.recordFailedMessage(comment)
	}

	// 7. 构建返回的CommentInfo对象
	commentInfo := &pb.CommentInfo{
		SnowCid:    strconv.FormatInt(snowCid, 10),
		Tid:        comment.Tid,
		Uid:        comment.Uid,
		ParentId:   comment.ParentId,
		RootId:     comment.RootId,
		Content:    comment.Content,
		LikeCount:  comment.LikeCount,
		ReplyCount: comment.ReplyCount,
		Status:     int32(comment.Status),
		CreateTime: comment.CreateTime.Format("2006-01-02 15:04:05"),
	}

	return &pb.CreateCommentResp{
		Comment: commentInfo,
	}, nil
}

// 简化的send方法
func (l *CreateCommentLogic) sendCreateCommentMessage(comment *model.Comment) error {
	message := map[string]interface{}{
		"action":      "create_comment",
		"snow_cid":    comment.SnowCid,
		"tid":         comment.Tid,
		"uid":         comment.Uid,
		"parent_id":   comment.ParentId,
		"root_id":     comment.RootId,
		"content":     comment.Content,
		"status":      comment.Status,
		"create_time": comment.CreateTime.Format(time.RFC3339),
		"like_count":  comment.LikeCount,
		"reply_count": comment.ReplyCount,
	}

	body, err := json.Marshal(message)
	if err != nil {
		return err
	}

	pusher := l.svcCtx.GetPusher("comment_create")
	return pusher.PushWithKey(l.ctx, strconv.FormatInt(comment.SnowCid, 10), string(body))
}

// 简化的recordFailedMessage
func (l *CreateCommentLogic) recordFailedMessage(comment *model.Comment) {
	failedMsg := map[string]interface{}{
		"snow_cid":    comment.SnowCid,
		"tid":         comment.Tid,
		"uid":         comment.Uid,
		"parent_id":   comment.ParentId,
		"root_id":     comment.RootId,
		"content":     comment.Content,
		"create_time": comment.CreateTime,
		"retry_count": 0,
		"last_retry":  time.Now().Unix(),
	}

	failedBody, _ := json.Marshal(failedMsg)
	_, err := l.svcCtx.RedisClient.LpushCtx(l.ctx, "failed:comment:create", string(failedBody))
	if err != nil {
		logx.Errorf("recordFailedMessage lpush errorx: %v", err)
	}
}
