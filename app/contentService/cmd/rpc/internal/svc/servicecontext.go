package svc

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"gozeroX/app/contentService/cmd/rpc/internal/config"
	"gozeroX/app/contentService/cmd/rpc/pb"
	"gozeroX/app/contentService/model"
	"gozeroX/app/usercenter/cmd/rpc/usercenter"
	"gozeroX/pkg/cache"

	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config        config.Config
	RedisClient   *redis.Redis
	CacheManager  *cache.Manager
	TweetModel    model.TweetModel
	UserCenterRpc usercenter.UserCenter

	// Kafka pusher
	pusherMu   sync.RWMutex
	pusherPool map[string]*kq.Pusher // topic -> pusher
}

func NewServiceContext(c config.Config) *ServiceContext {
	// Postgres 连接
	sqlConn := sqlx.NewSqlConn("postgres", c.DB.DataSource)

	// Redis 客户端
	redisClient := redis.MustNewRedis(redis.RedisConf{
		Host: c.RedisConf.Host,
		Pass: c.RedisConf.Pass,
		Type: c.RedisConf.Type,
	})

	// 缓存管理器
	cacheManager := cache.NewManager(redisClient)

	return &ServiceContext{
		Config:        c,
		RedisClient:   redisClient,
		CacheManager:  cacheManager,
		TweetModel:    model.NewTweetModel(sqlConn, c.Cache),
		UserCenterRpc: usercenter.NewUserCenter(zrpc.MustNewClient(c.UserCenterRpcConf)),
		pusherPool:    make(map[string]*kq.Pusher),
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

// Close 关闭资源
func (s *ServiceContext) Close() {
	s.pusherMu.Lock()
	defer s.pusherMu.Unlock()

	for topic, pusher := range s.pusherPool {
		if err := pusher.Close(); err != nil {
			logx.Errorf("关闭Kafka生产者失败, topic:%s, error:%v", topic, err)
		}
		delete(s.pusherPool, topic)
	}
}

// ==================== 推文缓存相关 ====================

// SetTweetToCache 将推文存入Hash缓存
func (s *ServiceContext) SetTweetToCache(ctx context.Context, snowTid int64, tweet *model.Tweet) error {
	// 序列化数组字段
	mediaUrlsJson, _ := json.Marshal(tweet.MediaUrls)
	tagsJson, _ := json.Marshal(tweet.Tags)

	fields := map[string]interface{}{
		"snow_tid":      tweet.SnowTid,
		"uid":           tweet.Uid,
		"content":       tweet.Content,
		"media_urls":    string(mediaUrlsJson),
		"tags":          string(tagsJson),
		"is_public":     tweet.IsPublic,
		"created_at":    tweet.CreatedAt, // int64 时间戳
		"status":        tweet.Status,    // 0正常, 1删除, 2审核
		"like_count":    tweet.LikeCount,
		"comment_count": tweet.CommentCount,
	}

	if err := s.CacheManager.HSetAll(ctx, "tweet", "info", snowTid, fields); err != nil {
		logx.Errorf("SetTweetToCache error, snowTid:%d, err:%v", snowTid, err)
		return err
	}

	if err := s.CacheManager.Expire(ctx, "tweet", "info", snowTid, 3600); err != nil {
		logx.Errorf("SetTweetToCache set expire error, snowTid:%d, err:%v", snowTid, err)
		return err
	}

	return nil
}

// GetTweetFromCache 从Hash缓存获取推文
func (s *ServiceContext) GetTweetFromCache(ctx context.Context, snowTid int64) (*model.Tweet, error) {
	fields, err := s.CacheManager.HGetAll(ctx, "tweet", "info", snowTid)
	if err != nil {
		logx.Errorf("GetTweetFromCache error, snowTid:%d, err:%v", snowTid, err)
		return nil, err
	}

	if len(fields) == 0 {
		return nil, redis.Nil
	}

	tweet := &model.Tweet{}

	if v, ok := fields["snow_tid"]; ok {
		tweet.SnowTid, _ = strconv.ParseInt(v, 10, 64)
	}
	if v, ok := fields["uid"]; ok {
		tweet.Uid, _ = strconv.ParseInt(v, 10, 64)
	}
	if v, ok := fields["content"]; ok {
		tweet.Content = v
	}
	if v, ok := fields["media_urls"]; ok {
		var mediaUrls []string
		_ = json.Unmarshal([]byte(v), &mediaUrls)
		tweet.MediaUrls = mediaUrls
	}
	if v, ok := fields["tags"]; ok {
		var tags []string
		_ = json.Unmarshal([]byte(v), &tags)
		tweet.Tags = tags
	}
	if v, ok := fields["is_public"]; ok {
		tweet.IsPublic, _ = strconv.ParseBool(v)
	}
	if v, ok := fields["created_at"]; ok {
		tweet.CreatedAt, _ = strconv.ParseInt(v, 10, 64) // int64 时间戳
	}
	if v, ok := fields["status"]; ok {
		tweet.Status, _ = strconv.ParseInt(v, 10, 64) // 0正常, 1删除, 2审核
	}
	if v, ok := fields["like_count"]; ok {
		tweet.LikeCount, _ = strconv.ParseInt(v, 10, 64)
	}
	if v, ok := fields["comment_count"]; ok {
		tweet.CommentCount, _ = strconv.ParseInt(v, 10, 64)
	}

	return tweet, nil
}

// DelTweetCache 删除推文缓存
func (s *ServiceContext) DelTweetCache(ctx context.Context, snowTid int64) {
	if err := s.CacheManager.Del(ctx, "tweet", "info", snowTid); err != nil {
		logx.Errorf("DelTweetCache error, snowTid:%d, err:%v", snowTid, err)
	}
}

// ==================== 数据转换 ====================

// BuildTweet 将 model.Tweet 转换为 pb.Tweet
func (s *ServiceContext) BuildTweet(tweet *model.Tweet) *pb.Tweet {
	if tweet == nil {
		return nil
	}
	return &pb.Tweet{
		SnowTid:      tweet.SnowTid,
		Uid:          tweet.Uid,
		Content:      tweet.Content,
		MediaUrls:    tweet.MediaUrls,
		Tags:         tweet.Tags,
		IsPublic:     tweet.IsPublic,
		CreatedAt:    tweet.CreatedAt,
		LikeCount:    tweet.LikeCount,
		CommentCount: tweet.CommentCount,
		Status:       tweet.Status, // 0正常, 1删除, 2审核
	}
}

// ==================== 用户统计相关（调用 UserCenter RPC） ====================

// IncrUserPostCount 增加用户发帖数
func (s *ServiceContext) IncrUserPostCount(ctx context.Context, uid int64, delta int64) error {
	resp, err := s.UserCenterRpc.UpdateUserStats(ctx, &usercenter.UpdateUserStatsReq{
		Uid:        uid,
		UpdateType: 3, // 3=post_count
		Delta:      delta,
	})
	if err != nil {
		return err
	}
	if resp.Code != 0 {
		return fmt.Errorf("UpdateUserStats failed: %s", resp.Msg)
	}
	return nil
}
