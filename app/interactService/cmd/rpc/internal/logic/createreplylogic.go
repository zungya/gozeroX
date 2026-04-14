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

type CreateReplyLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateReplyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateReplyLogic {
	return &CreateReplyLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// CreateReply 创建回复（子评论，write-behind模式）
func (l *CreateReplyLogic) CreateReply(in *pb.CreateReplyReq) (*pb.CreateReplyResp, error) {
	// 1. 生成雪花ID
	snowCid, err := idgen.GenID()
	if err != nil {
		return &pb.CreateReplyResp{Code: 120101, Msg: "生成回复ID失败"}, nil
	}

	// 2. 构建回复对象
	now := time.Now().UnixMilli()
	reply := &model.Reply{
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

	// 3. 写缓存
	if err := l.svcCtx.SetReplyToCache(l.ctx, snowCid, reply); err != nil {
		return &pb.CreateReplyResp{Code: 120103, Msg: "缓存回复失败"}, nil
	}

	// 4. 更新缓存计数
	go func() {
		// 推文评论数 +1
		if err := l.svcCtx.IncrTweetCommentCount(context.Background(), in.SnowTid, 1); err != nil {
			logx.Errorf("CreateReply incr tweet comment count error, snowTid:%d, err:%v", in.SnowTid, err)
		}
		// 父评论回复数 +1
		if err := l.svcCtx.IncrCommentReplyCount(context.Background(), in.ParentId, 1); err != nil {
			logx.Errorf("CreateReply incr comment reply count error, parentId:%d, err:%v", in.ParentId, err)
		}
		// 添加到回复 Set
		if reply.RootId != 0 {
			_ = l.svcCtx.CacheManager.SAdd(context.Background(), "comment", "replies", reply.RootId, snowCid)
			_ = l.svcCtx.CacheManager.Expire(context.Background(), "comment", "replies", reply.RootId, 1800)
		}
	}()

	// 5. 发送 Kafka 消息异步落库
	if err := l.sendCreateReplyMessage(reply); err != nil {
		logx.Errorf("CreateReply send queue message error, snowCid:%d, err:%v", snowCid, err)
		go l.recordFailedMessage(reply)
	}

	// 6. 异步通知被回复者
	go func() {
		defer func() { recover() }()
		l.sendReplyNotification(reply)
	}()

	// 7. 异步发送推荐事件
	go func() {
		defer func() { recover() }()
		l.sendRecommendInteraction("reply_comment", reply.Uid, reply.SnowTid, reply.SnowCid, reply.Content)
	}()

	// 8. 获取用户信息
	var nickname, avatar string
	userInfoResp, err := l.svcCtx.UserCenterRpc.GetUserInfo(l.ctx, &usercenter.GetUserInfoReq{Uid: reply.Uid})
	if err != nil {
		nickname, avatar = "用户", ""
	} else if userInfoResp.Code == 0 && userInfoResp.UserInfo != nil {
		nickname, avatar = userInfoResp.UserInfo.Nickname, userInfoResp.UserInfo.Avatar
	}

	// 9. 返回
	replyInfo := &pb.ReplyInfo{
		SnowCid:    snowCid,
		SnowTid:    reply.SnowTid,
		Uid:        reply.Uid,
		ParentId:   reply.ParentId,
		RootId:     reply.RootId,
		Content:    reply.Content,
		LikeCount:  reply.LikeCount,
		ReplyCount: reply.ReplyCount,
		CreateTime: reply.CreatedAt,
		Nickname:   nickname,
		Avatar:     avatar,
	}

	return &pb.CreateReplyResp{Code: 0, Msg: "success", Reply: replyInfo}, nil
}

func (l *CreateReplyLogic) sendCreateReplyMessage(reply *model.Reply) error {
	message := map[string]interface{}{
		"action":      "create_reply",
		"snow_cid":    reply.SnowCid,
		"snow_tid":    reply.SnowTid,
		"uid":         reply.Uid,
		"parent_id":   reply.ParentId,
		"root_id":     reply.RootId,
		"content":     reply.Content,
		"status":      reply.Status,
		"created_at":  reply.CreatedAt,
		"updated_at":  reply.UpdatedAt,
		"like_count":  reply.LikeCount,
		"reply_count": reply.ReplyCount,
	}
	body, err := json.Marshal(message)
	if err != nil {
		return err
	}
	pusher := l.svcCtx.GetPusher("comment_create")
	return pusher.PushWithKey(l.ctx, fmt.Sprintf("%d", reply.SnowCid), string(body))
}

func (l *CreateReplyLogic) recordFailedMessage(reply *model.Reply) {
	failedMsg := map[string]interface{}{
		"snow_cid":    reply.SnowCid,
		"snow_tid":    reply.SnowTid,
		"uid":         reply.Uid,
		"parent_id":   reply.ParentId,
		"root_id":     reply.RootId,
		"content":     reply.Content,
		"created_at":  reply.CreatedAt,
		"retry_count": 0,
		"last_retry":  time.Now().Unix(),
	}
	failedBody, _ := json.Marshal(failedMsg)
	_, err := l.svcCtx.RedisClient.LpushCtx(l.ctx, "failed:reply:create", string(failedBody))
	if err != nil {
		logx.Errorf("recordFailedMessage lpush error: %v", err)
	}
}

func (l *CreateReplyLogic) sendReplyNotification(reply *model.Reply) {
	// 获取父评论作者UID和内容
	parentComment, err := l.svcCtx.GetCommentOrReplyBySnowCid(context.Background(), reply.ParentId)
	if err != nil {
		logx.Errorf("sendReplyNotification GetCommentOrReply error, parentId:%d, err:%v", reply.ParentId, err)
		return
	}
	// 自己回复自己不发通知
	if parentComment.Uid == reply.Uid {
		return
	}

	message := map[string]interface{}{
		"action":          "reply_comment",
		"target_type":     1,
		"commenter_uid":   reply.Uid,
		"recipient_uid":   parentComment.Uid,
		"snow_tid":        reply.SnowTid,
		"snow_cid":        reply.SnowCid,
		"root_id":         reply.RootId,
		"parent_id":       reply.ParentId,
		"content":         reply.Content,
		"replied_content": parentComment.Content,
		"timestamp":       reply.CreatedAt,
	}
	body, _ := json.Marshal(message)
	pusher := l.svcCtx.GetPusher("notice")
	if err := pusher.PushWithKey(context.Background(), fmt.Sprintf("reply_%d_%d", parentComment.Uid, reply.SnowCid), string(body)); err != nil {
		logx.Errorf("sendReplyNotification push error: %v", err)
	}
}

func (l *CreateReplyLogic) sendRecommendInteraction(action string, uid, snowTid, snowCid int64, content string) {
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
		return
	}
	pusher := l.svcCtx.GetPusher("recommend_interaction")
	if err := pusher.PushWithKey(context.Background(), fmt.Sprintf("%d_%d", uid, snowCid), string(body)); err != nil {
		logx.Errorf("sendRecommendInteraction push error: %v", err)
	}
}
