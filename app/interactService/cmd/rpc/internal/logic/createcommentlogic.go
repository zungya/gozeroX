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

// CreateComment 创建根评论（write-behind模式）
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

	// 2. 构建评论对象（根评论，无 ParentId/RootId）
	now := time.Now().UnixMilli()
	comment := &model.Comment{
		SnowCid:    snowCid,
		SnowTid:    in.SnowTid,
		Uid:        in.Uid,
		Content:    in.Content,
		LikeCount:  0,
		ReplyCount: 0,
		Status:     0,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	// 3. 先写缓存
	if err := l.svcCtx.SetCommentToCache(l.ctx, snowCid, comment); err != nil {
		logx.Errorf("CreateComment SetCommentToCache errorx, snowCid:%d, err:%v", snowCid, err)
		return &pb.CreateCommentResp{
			Code: 120103,
			Msg:  "缓存评论失败",
		}, nil
	}

	// 4. 更新缓存计数和列表
	go func() {
		if err := l.svcCtx.IncrTweetCommentCount(context.Background(), in.SnowTid, 1); err != nil {
			logx.Errorf("CreateComment incr tweet comment count errorx, snowTid:%d, err:%v", in.SnowTid, err)
		}
		_ = l.svcCtx.CacheManager.SAdd(
			context.Background(),
			"tweet",
			"top_comments",
			in.SnowTid,
			snowCid,
		)
	}()

	// 5. 发送Kafka消息，异步落库
	if err := l.sendCreateCommentMessage(comment); err != nil {
		logx.Errorf("CreateComment send queue message errorx, snowCid:%d, err:%v", snowCid, err)
		go l.recordFailedMessage(comment)
	}

	// 6. 异步通知推文作者
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logx.Errorf("sendTweetCommentNotification panic: %v", r)
			}
		}()
		l.sendTweetCommentNotification(comment)
	}()

	// 7. 异步发送互动事件到 Kafka（推荐系统用）
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logx.Errorf("sendRecommendInteraction panic: %v", r)
			}
		}()
		l.sendRecommendInteraction("comment_tweet", comment.Uid, comment.SnowTid, comment.SnowCid, comment.Content)
	}()

	// 8. 调用 usercenter RPC 获取用户信息（昵称、头像）
	var nickname, avatar string
	userInfoResp, err := l.svcCtx.UserCenterRpc.GetUserInfo(l.ctx, &usercenter.GetUserInfoReq{
		Uid: comment.Uid,
	})
	if err != nil {
		logx.Errorf("CreateComment GetUserInfo RPC errorx, uid:%d, err:%v", comment.Uid, err)
		nickname = "用户"
		avatar = ""
	} else if userInfoResp.Code == 0 && userInfoResp.UserInfo != nil {
		nickname = userInfoResp.UserInfo.Nickname
		avatar = userInfoResp.UserInfo.Avatar
	}

	// 9. 构建返回的CommentInfo对象
	commentInfo := &pb.CommentInfo{
		SnowCid:    snowCid,
		SnowTid:    comment.SnowTid,
		Uid:        comment.Uid,
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

// sendTweetCommentNotification 根评论通知推文作者
func (l *CreateCommentLogic) sendTweetCommentNotification(comment *model.Comment) {
	// 获取推文作者UID
	recipientUid, err := l.svcCtx.GetTweetAuthorUid(context.Background(), comment.SnowTid)
	if err != nil {
		logx.Errorf("sendTweetCommentNotification GetTweetAuthorUid error: %v", err)
		return
	}
	// 自己评论自己的推文不发通知
	if recipientUid == comment.Uid {
		return
	}

	message := map[string]interface{}{
		"action":          "comment_tweet",
		"target_type":     0,
		"commenter_uid":   comment.Uid,
		"recipient_uid":   recipientUid,
		"snow_tid":        comment.SnowTid,
		"snow_cid":        comment.SnowCid,
		"root_id":         0,
		"parent_id":       0,
		"content":         comment.Content,
		"replied_content": "",
		"timestamp":       comment.CreatedAt,
	}
	body, err := json.Marshal(message)
	if err != nil {
		logx.Errorf("sendTweetCommentNotification marshal error: %v", err)
		return
	}
	pusher := l.svcCtx.GetPusher("notice")
	if err := pusher.PushWithKey(context.Background(), fmt.Sprintf("comment_%d_%d", recipientUid, comment.SnowCid), string(body)); err != nil {
		logx.Errorf("sendTweetCommentNotification push error: %v", err)
	}
}

// sendRecommendInteraction 发送互动事件到 Kafka recommend_interaction topic（推荐系统用）
func (l *CreateCommentLogic) sendRecommendInteraction(action string, uid, snowTid, snowCid int64, content string) {
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
		logx.Errorf("sendRecommendInteraction marshal error: %v", err)
		return
	}
	pusher := l.svcCtx.GetPusher("recommend_interaction")
	if err := pusher.PushWithKey(context.Background(), fmt.Sprintf("%d_%d", uid, snowCid), string(body)); err != nil {
		logx.Errorf("sendRecommendInteraction push error: %v", err)
	}
}
