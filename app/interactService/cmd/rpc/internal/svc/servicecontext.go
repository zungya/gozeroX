package svc

import (
	"context"
	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/queue"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"gozeroX/app/interactService/cmd/rpc/internal/config"
	"gozeroX/app/interactService/model"
	"gozeroX/pkg/cache"
	"strconv"
	"sync"
	"time"
)

type ServiceContext struct {
	Config        config.Config
	RedisClient   *redis.Redis
	CacheManager  *cache.Manager
	CommentModel  model.CommentModel
	LikesModel    model.LikesModel
	QueueProducer *queue.Producer

	pusherMu   sync.RWMutex
	pusherPool map[string]*kq.Pusher // topic -> pusher
}

func NewServiceContext(c config.Config) *ServiceContext {
	// Postgres 连接
	sqlConn := sqlx.NewSqlConn("postgres", c.DB.DataSource)

	// Redis 客户端
	redisClient := redis.MustNewRedis(redis.RedisConf{
		Host: c.Redis.Host,
		Pass: c.Redis.Pass,
		Type: c.Redis.Type,
	})

	// 缓存管理器
	cacheManager := cache.NewManager(redisClient)

	return &ServiceContext{
		Config:       c,
		RedisClient:  redisClient,
		CacheManager: cacheManager,
		CommentModel: model.NewCommentModel(sqlConn, c.Cache),
		LikesModel:   model.NewLikesModel(sqlConn, c.Cache),
		pusherPool:   make(map[string]*kq.Pusher),
	}
}

// GetPusher 获取或创建指定topic的Pusher（单例）
func (s *ServiceContext) GetPusher(topic string) *kq.Pusher {
	s.pusherMu.RLock()
	pusher, ok := s.pusherPool[topic]
	s.pusherMu.RUnlock()

	if ok {
		return pusher
	}

	s.pusherMu.Lock()
	defer s.pusherMu.Unlock()

	// 双重检查
	if pusher, ok = s.pusherPool[topic]; ok {
		return pusher
	}

	// 创建新的pusher
	pusher = kq.NewPusher(
		s.Config.Kafka.Addrs,
		topic,
		kq.WithChunkSize(1024),
		kq.WithFlushInterval(time.Second),
	)

	s.pusherPool[topic] = pusher
	return pusher
}

func (s *ServiceContext) Close() {
	s.pusherMu.Lock()
	defer s.pusherMu.Unlock()

	for topic, pusher := range s.pusherPool {
		if err := pusher.Close(); err != nil {
			logx.Errorf("关闭Kafka生产者失败, topic:%s, errorx:%v", topic, err)
		}
		delete(s.pusherPool, topic)
	}
}
func (s *ServiceContext) SetCommentToCache(ctx context.Context, snowCid int64, comment *model.Comment) error {
	fields := map[string]interface{}{
		"snow_cid":    comment.SnowCid,
		"tid":         comment.Tid,
		"uid":         comment.Uid,
		"parent_id":   comment.ParentId,
		"root_id":     comment.RootId,
		"content":     comment.Content,
		"status":      comment.Status,
		"create_time": comment.CreateTime,
		"like_count":  comment.LikeCount,
		"reply_count": comment.ReplyCount,
	}

	if err := s.CacheManager.HSetAll(ctx, "comment", "info", snowCid, fields); err != nil {
		logx.Errorf("SetCommentToCache errorx, snow_cid:%d, err:%v", snowCid, err)
		return err
	}

	if err := s.CacheManager.Expire(ctx, "comment", "info", snowCid, 3600); err != nil {
		logx.Errorf("SetCommentToCache set expire errorx, snow_cid:%d, err:%v", snowCid, err)
		return err
	}

	return nil
}

// GetCommentFromCache 从Hash缓存获取评论
func (s *ServiceContext) GetCommentFromCache(ctx context.Context, snowCid int64) (*model.Comment, error) {
	fields, err := s.CacheManager.HGetAll(ctx, "comment", "info", snowCid)
	if err != nil {
		logx.Errorf("GetCommentFromCache errorx, snowCid:%d, err:%v", snowCid, err)
		return nil, err
	}

	if len(fields) == 0 {
		return nil, redis.Nil
	}

	comment := &model.Comment{}

	if v, ok := fields["snow_cid"]; ok {
		comment.SnowCid, _ = strconv.ParseInt(v, 10, 64)
	}
	if v, ok := fields["tid"]; ok {
		comment.Tid, _ = strconv.ParseInt(v, 10, 64)
	}
	if v, ok := fields["uid"]; ok {
		comment.Uid, _ = strconv.ParseInt(v, 10, 64)
	}
	if v, ok := fields["parent_id"]; ok {
		comment.ParentId, _ = strconv.ParseInt(v, 10, 64)
	}
	if v, ok := fields["root_id"]; ok {
		comment.RootId, _ = strconv.ParseInt(v, 10, 64)
	}
	if v, ok := fields["content"]; ok {
		comment.Content = v
	}
	if v, ok := fields["status"]; ok {
		comment.Status, _ = strconv.ParseInt(v, 10, 64)
	}
	if v, ok := fields["create_time"]; ok {
		comment.CreateTime, _ = time.Parse(time.RFC3339, v)
	}
	if v, ok := fields["like_count"]; ok {
		comment.LikeCount, _ = strconv.ParseInt(v, 10, 64)
	}
	if v, ok := fields["reply_count"]; ok {
		comment.ReplyCount, _ = strconv.ParseInt(v, 10, 64)
	}

	return comment, nil
}

// IncrCommentLikeCount 原子增加评论点赞数
func (s *ServiceContext) IncrCommentLikeCount(ctx context.Context, snowCid int64, delta int) error {
	_, err := s.CacheManager.HIncrBy(ctx, "comment", "info", snowCid, "like_count", delta)
	return err
}

// IncrCommentReplyCount 原子增加评论回复数
func (s *ServiceContext) IncrCommentReplyCount(ctx context.Context, snowCid int64, delta int) error {
	_, err := s.CacheManager.HIncrBy(ctx, "comment", "info", snowCid, "reply_count", delta)
	return err
}

// DelCommentCache 删除评论缓存
func (s *ServiceContext) DelCommentCache(ctx context.Context, snowCid int64) {
	if err := s.CacheManager.Del(ctx, "comment", "info", snowCid); err != nil {
		logx.Errorf("DelCommentCache errorx, snowCid:%d, err:%v", snowCid, err)
	}
}

func (s *ServiceContext) IncrTweetCommentCount(ctx context.Context, tid int64, delta int) error {
	_, err := s.CacheManager.HIncrBy(ctx, "tweet", "info", tid, "comment_count", delta)
	return err
}

func (s *ServiceContext) GetCommentBySnowCid(ctx context.Context, snowCid int64) (*model.Comment, error) {
	// 1. 先从缓存获取
	comment, err := s.GetCommentFromCache(ctx, snowCid)
	if err == nil {
		return comment, nil // 缓存命中
	}

	// 2. 缓存未命中，从数据库查询
	comment, err = s.CommentModel.FindOneBySnowCid(ctx, snowCid)
	if err != nil {
		return nil, err
	}

	// 3. 回写缓存（异步，不影响主流程）
	go func() {
		_ = s.SetCommentToCache(context.Background(), snowCid, comment)
	}()

	return comment, nil
}

// GetTopCommentsByTid 获取推文的顶级评论snow_cid列表（先缓存后DB）
func (s *ServiceContext) GetTopCommentsByTid(ctx context.Context, tid int64) ([]int64, error) {
	// 1. 先从缓存获取snow_cid列表
	snowCids, err := s.CacheManager.SMembers(ctx, "tweet", "top_comments", tid)
	if err == nil && len(snowCids) > 0 {
		return snowCids, nil
	}

	// 2. 缓存未命中，从数据库查询顶级评论（parent_id=0）
	dbSnowCids, err := s.CommentModel.FindTopSnowCidsByTid(ctx, tid)
	if err != nil {
		return nil, err
	}

	// 3. 回写缓存（异步）
	if len(dbSnowCids) > 0 {
		go func() {
			// 使用CacheManager的SAdd方法存储到Set
			_ = s.CacheManager.SAdd(context.Background(), "tweet", "top_comments", tid, dbSnowCids...)
			// 设置过期时间（30分钟）
			_ = s.CacheManager.Expire(context.Background(), "tweet", "top_comments", tid, 1800)
		}()
	}

	return dbSnowCids, nil
}

// GetRepliesByParentId 获取父评论的回复snow_cid列表（先缓存后DB）
func (s *ServiceContext) GetRepliesByParentId(ctx context.Context, parentSnowCid int64) ([]int64, error) {
	// 1. 先从缓存获取回复snow_cid列表
	replySnowCids, err := s.CacheManager.SMembers(ctx, "comment", "replies", parentSnowCid)
	if err == nil && len(replySnowCids) > 0 {
		return replySnowCids, nil
	}

	// 2. 缓存未命中，从数据库查询回复（parent_id = parentSnowCid）
	dbReplySnowCids, err := s.CommentModel.FindReplySnowCidsByParentId(ctx, parentSnowCid)
	if err != nil {
		return nil, err
	}

	// 3. 回写缓存（异步）
	if len(dbReplySnowCids) > 0 {
		go func() {
			// 使用CacheManager的SAdd方法存储到Set
			_ = s.CacheManager.SAdd(context.Background(), "comment", "replies", parentSnowCid, dbReplySnowCids...)
			// 设置过期时间（30分钟）
			_ = s.CacheManager.Expire(context.Background(), "comment", "replies", parentSnowCid, 1800)
		}()
	}

	return dbReplySnowCids, nil
}
