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

type LikeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewLikeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LikeLogic {
	return &LikeLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// Like 点赞/取消点赞（先发Kafka，再写缓存）
func (l *LikeLogic) Like(in *pb.LikeReq) (*pb.LikeResp, error) {
	// 1. 处理snow_likes_id
	var snowLikesId int64
	var err error

	if in.IsCreated == 1 {
		// 非第一次操作，使用前端传来的snow_likes_id
		snowLikesId, err = strconv.ParseInt(in.SnowLikesId, 10, 64)
		if err != nil {
			logx.Errorf("Like parse snow_likes_id errorx: %v", err)
			return nil, fmt.Errorf("点赞记录ID格式错误: %v", err)
		}
	} else {
		// 第一次操作，生成新的雪花ID
		snowLikesId, err = idgen.GenID()
		if err != nil {
			logx.Errorf("Like generate snowflake id errorx: %v", err)
			return nil, fmt.Errorf("生成点赞ID失败: %v", err)
		}
	}

	// 2. 更新时间
	updateTime := time.Now()

	// 3. 构建点赞对象
	like := &model.Likes{
		Uid:         in.Uid,
		TargetType:  int64(in.TargetType),
		TargetId:    in.TargetId,
		Status:      int64(in.Status),
		UpdateTime:  updateTime,
		SnowLikesId: snowLikesId,
	}

	// 4. 先发送异步消息到Kafka（保证数据不丢失）
	go func() {
		if err := l.sendLikeMessage(like, in.IsCreated); err != nil {
			logx.Errorf("Like send queue message errorx, snowLikesId:%d, err:%v", snowLikesId, err)
			l.recordFailedMessage(like, in.IsCreated)
		}
	}()

	// 5. 异步写缓存（点赞记录）
	go func() {
		if err := l.writeLikeToCache(like); err != nil {
			logx.Errorf("Like write to cache errorx: %v", err)
		}
	}()

	// 6. 异步更新用户点赞映射（Hash结构）
	go func() {
		if err := l.updateUserLikeMapping(in.Uid, in.TargetType, in.TargetId, in.Status, snowLikesId); err != nil {
			logx.Errorf("Like update user like mapping errorx: %v", err)
		}
	}()

	// 7. 异步原子更新目标对象的点赞数缓存（使用Hash的HIncrBy）
	go l.updateTargetLikeCountAtomic(in.TargetType, in.TargetId, in.Status)

	// 8. 立即返回（用户感知操作成功）
	likeInfo := &pb.LikeInfo{
		SnowLikesId: strconv.FormatInt(snowLikesId, 10),
		Uid:         in.Uid,
		TargetType:  in.TargetType,
		TargetId:    in.TargetId,
		Status:      in.Status,
		UpdateTime:  updateTime.Format(time.RFC3339),
	}

	logx.Infof("Like success, snowLikesId:%d, uid:%d, targetType:%d, targetId:%d, status:%d",
		snowLikesId, in.Uid, in.TargetType, in.TargetId, in.Status)

	return &pb.LikeResp{
		Like: likeInfo,
	}, nil
}

// writeLikeToCache 将点赞记录写入缓存（直接覆盖）
func (l *LikeLogic) writeLikeToCache(like *model.Likes) error {
	return l.svcCtx.CacheManager.Set(
		context.Background(),
		"like",
		"info",
		like.SnowLikesId,
		map[string]interface{}{
			"snow_likes_id": like.SnowLikesId,
			"uid":           like.Uid,
			"target_type":   like.TargetType,
			"target_id":     like.TargetId,
			"status":        like.Status,
			"update_time":   like.UpdateTime,
		},
		3600,
	)
}

// updateUserLikeMapping 更新用户点赞映射（使用Hash结构）
func (l *LikeLogic) updateUserLikeMapping(uid int64, targetType int32, targetId int64, status int32, snowLikesId int64) error {
	field := strconv.FormatInt(targetId, 10)
	keyId := fmt.Sprintf("%d:%d", uid, targetType)

	if status == 1 {
		// 点赞：存储snow_likes_id
		return l.svcCtx.CacheManager.HSet(context.Background(), "user", "likes", keyId, field, snowLikesId)
	}
	// 取消点赞：删除映射
	return l.svcCtx.CacheManager.HDel(context.Background(), "user", "likes", keyId, field)
}

// updateTargetLikeCountAtomic 原子更新目标对象的点赞数缓存（使用Hash的HIncrBy）
func (l *LikeLogic) updateTargetLikeCountAtomic(targetType int32, targetId int64, status int32) {
	var module, dataType string

	// 根据目标类型确定缓存key
	switch targetType {
	case 1:
		module = "tweet"
		dataType = "info"
	case 2:
		module = "comment"
		dataType = "info"
	default:
		logx.Errorf("updateTargetLikeCountAtomic unknown target type: %d", targetType)
		return
	}

	// 先检查key是否存在
	exists, err := l.svcCtx.CacheManager.Exists(context.Background(), module, dataType, targetId)
	if err != nil {
		logx.Errorf("updateTargetLikeCountAtomic check exists errorx: %v", err)
		return
	}

	if !exists {
		// key不存在，不进行操作
		logx.Infof("updateTargetLikeCountAtomic key not exists, skip, targetType:%d, targetId:%d", targetType, targetId)
		return
	}

	// key存在，使用Hash原子操作
	if status == 1 {
		// 点赞：like_count +1
		newCount, err := l.svcCtx.CacheManager.HIncrBy(context.Background(), module, dataType, targetId, "like_count", 1)
		if err != nil {
			logx.Errorf("updateTargetLikeCountAtomic incr errorx: %v", err)
			return
		}
		logx.Infof("updateTargetLikeCountAtomic incr success, targetType:%d, targetId:%d, newCount:%d",
			targetType, targetId, newCount)
	} else {
		// 取消点赞：like_count -1
		newCount, err := l.svcCtx.CacheManager.HIncrBy(context.Background(), module, dataType, targetId, "like_count", -1)
		if err != nil {
			logx.Errorf("updateTargetLikeCountAtomic decr errorx: %v", err)
			return
		}

		logx.Infof("updateTargetLikeCountAtomic decr success, targetType:%d, targetId:%d, newCount:%d",
			targetType, targetId, newCount)
	}
}

// sendLikeMessage 发送点赞消息到Kafka
func (l *LikeLogic) sendLikeMessage(like *model.Likes, isCreated int32) error {
	message := map[string]interface{}{
		"action":        "like",
		"is_created":    isCreated,
		"snow_likes_id": like.SnowLikesId,
		"uid":           like.Uid,
		"target_type":   like.TargetType,
		"target_id":     like.TargetId,
		"status":        like.Status,
		"update_time":   like.UpdateTime,
	}

	body, err := json.Marshal(message)
	if err != nil {
		return err
	}

	pusher := l.svcCtx.GetPusher("like")
	return pusher.PushWithKey(context.Background(), strconv.FormatInt(like.SnowLikesId, 10), string(body))
}

// recordFailedMessage 记录失败消息
func (l *LikeLogic) recordFailedMessage(like *model.Likes, isCreated int32) {
	failedMsg := map[string]interface{}{
		"snow_likes_id": like.SnowLikesId,
		"uid":           like.Uid,
		"target_type":   like.TargetType,
		"target_id":     like.TargetId,
		"status":        like.Status,
		"is_created":    isCreated,
		"update_time":   like.UpdateTime,
		"retry_count":   0,
		"last_retry":    time.Now().Unix(),
	}

	failedBody, _ := json.Marshal(failedMsg)
	_, err := l.svcCtx.RedisClient.LpushCtx(context.Background(), "failed:like", string(failedBody))
	if err != nil {
		logx.Errorf("recordFailedMessage lpush errorx: %v", err)
	}
}
