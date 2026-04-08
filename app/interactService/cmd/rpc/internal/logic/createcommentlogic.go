package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"gozeroX/app/interactService/model"
	"gozeroX/app/usercenter/cmd/rpc/usercenter"
	"gozeroX/pkg/idgen"
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
		return &pb.CreateCommentResp{
			Code: 120101,
			Msg:  "生成评论ID失败",
		}, nil
	}

	// 2. 构建评论对象（时间用毫秒戳）
	now := time.Now().UnixMilli()
	comment := &model.Comment{
		SnowCid:    snowCid,
		SnowTid:    in.SnowTid,
		Uid:        in.Uid,
		ParentId:   in.ParentId,
		RootId:     in.RootId,
		Content:    in.Content,
		LikeCount:  0,
		ReplyCount: 0,
		Status:     0,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	// 3. 处理rootId逻辑
	if in.ParentId != 0 {
		// 父评论的rootId如果是0，说明父评论是顶级评论,则使用父评论的SnowCid作为rootId，否则就说明父评论不是根评论，直接使用rootId
		if in.RootId == 0 {
			comment.RootId = in.ParentId
		} else {
			comment.RootId = in.RootId
		}
	} else {
		comment.RootId = 0 // 顶级评论，rootId为0
	}

	// 4. 先写缓存
	if err := l.svcCtx.SetCommentToCache(l.ctx, snowCid, comment); err != nil {
		logx.Errorf("CreateComment SetCommentToCache errorx, snowCid:%d, err:%v", snowCid, err)
		return &pb.CreateCommentResp{
			Code: 120103,
			Msg:  "缓存评论失败",
		}, nil
	}

	// 5. 更新相关缓存计数和列表
	if in.ParentId == 0 {
		// 顶级评论：添加到推文的顶级评论列表
		go func() {
			// 增加推文评论数
			if err := l.svcCtx.IncrTweetCommentCount(l.ctx, in.SnowTid, 1); err != nil {
				logx.Errorf("CreateComment incr tweet comment count errorx, snowTid:%d, err:%v", in.SnowTid, err)
			}
			// 添加到顶级评论Set
			_ = l.svcCtx.CacheManager.SAdd(
				context.Background(),
				"tweet",
				"top_comments",
				in.SnowTid,
				snowCid,
			)
		}()
	} else {
		// 回复评论：增加父评论的回复数，添加到回复列表
		go func() {
			if err := l.svcCtx.IncrTweetCommentCount(l.ctx, in.SnowTid, 1); err != nil {
				logx.Errorf("CreateComment incr tweet comment count errorx, snowTid:%d, err:%v", in.SnowTid, err)
			}
			if err := l.svcCtx.IncrCommentReplyCount(l.ctx, in.ParentId, 1); err != nil {
				logx.Errorf("CreateComment incr comment reply count errorx, parentId:%d, err:%v", in.ParentId, err)
			}
			// 添加到回复Set（使用rootId作为key）
			if comment.RootId != 0 {
				_ = l.svcCtx.CacheManager.SAdd(
					context.Background(),
					"comment",
					"replies",
					comment.RootId,
					snowCid,
				)
				_ = l.svcCtx.CacheManager.Expire(context.Background(), "comment", "replies", comment.RootId, 1800)
			}
		}()
	}

	// 6. 发送go-queue消息，异步落库
	if err := l.sendCreateCommentMessage(comment); err != nil {
		logx.Errorf("CreateComment send queue message errorx, snowCid:%d, err:%v", snowCid, err)
		go l.recordFailedMessage(comment)
	}

	// 7. 调用 usercenter RPC 获取用户信息（昵称、头像）
	var nickname, avatar string
	userInfoResp, err := l.svcCtx.UserCenterRpc.GetUserInfo(l.ctx, &usercenter.GetUserInfoReq{
		Uid: comment.Uid,
	})
	if err != nil {
		logx.Errorf("CreateComment GetUserInfo RPC errorx, uid:%d, err:%v", comment.Uid, err)
		// RPC 调用失败不影响主流程，使用默认值
		nickname = "用户"
		avatar = ""
	} else if userInfoResp.Code == 0 && userInfoResp.UserInfo != nil {
		nickname = userInfoResp.UserInfo.Nickname
		avatar = userInfoResp.UserInfo.Avatar
	}

	// 8. 构建返回的CommentInfo对象
	commentInfo := &pb.CommentInfo{
		SnowCid:    snowCid,
		SnowTid:    comment.SnowTid,
		Uid:        comment.Uid,
		ParentId:   comment.ParentId,
		RootId:     comment.RootId,
		Content:    comment.Content,
		LikeCount:  comment.LikeCount,
		ReplyCount: comment.ReplyCount,
		CreateTime: comment.CreatedAt,
		Nickname:   nickname,
		Avatar:     avatar,
	}

	return &pb.CreateCommentResp{
		Code:    0,
		Msg:     "success",
		Comment: commentInfo,
	}, nil
}

// sendCreateCommentMessage 发送创建评论消息到Kafka
func (l *CreateCommentLogic) sendCreateCommentMessage(comment *model.Comment) error {
	message := map[string]interface{}{
		"action":      "create_comment",
		"snow_cid":    comment.SnowCid,
		"snow_tid":    comment.SnowTid,
		"uid":         comment.Uid,
		"parent_id":   comment.ParentId,
		"root_id":     comment.RootId,
		"content":     comment.Content,
		"status":      comment.Status,
		"created_at":  comment.CreatedAt,
		"updated_at":  comment.UpdatedAt,
		"like_count":  comment.LikeCount,
		"reply_count": comment.ReplyCount,
	}

	body, err := json.Marshal(message)
	if err != nil {
		return err
	}

	pusher := l.svcCtx.GetPusher("comment_create")
	return pusher.PushWithKey(l.ctx, fmt.Sprintf("%d", comment.SnowCid), string(body))
}

// recordFailedMessage 记录发送失败的消息
func (l *CreateCommentLogic) recordFailedMessage(comment *model.Comment) {
	failedMsg := map[string]interface{}{
		"snow_cid":    comment.SnowCid,
		"snow_tid":    comment.SnowTid,
		"uid":         comment.Uid,
		"parent_id":   comment.ParentId,
		"root_id":     comment.RootId,
		"content":     comment.Content,
		"created_at":  comment.CreatedAt,
		"retry_count": 0,
		"last_retry":  time.Now().Unix(),
	}

	failedBody, _ := json.Marshal(failedMsg)
	_, err := l.svcCtx.RedisClient.LpushCtx(l.ctx, "failed:comment:create", string(failedBody))
	if err != nil {
		logx.Errorf("recordFailedMessage lpush errorx: %v", err)
	}
}
