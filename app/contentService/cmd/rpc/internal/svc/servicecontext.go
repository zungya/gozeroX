package svc

import (
	"context"
	"encoding/json"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/zrpc"
	"gozeroX/app/contentService/cmd/rpc/internal/config"
	"gozeroX/app/contentService/cmd/rpc/pb"
	"gozeroX/app/contentService/model"
	"gozeroX/app/usercenter/cmd/rpc/usercenter"
	"gozeroX/pkg/cache"
	"strconv"
	"time"
)

type ServiceContext struct {
	Config        config.Config
	RedisClient   *redis.Redis
	CacheManager  *cache.Manager
	TweetModel    model.TweetModel
	UserCenterRpc usercenter.UserCenter
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
		Config:        c,
		RedisClient:   redisClient,
		CacheManager:  cacheManager,
		TweetModel:    model.NewTweetModel(sqlConn, c.Cache),
		UserCenterRpc: usercenter.NewUserCenter(zrpc.MustNewClient(c.UserCenterRpcConf)),
	}
}

// SetTweetToCache 将推文存入Hash缓存
func (s *ServiceContext) SetTweetToCache(ctx context.Context, tid int64, tweet *model.Tweet) error {
	// 序列化数组字段
	mediaUrlsJson, _ := json.Marshal(tweet.MediaUrls)
	tagsJson, _ := json.Marshal(tweet.Tags)

	fields := map[string]interface{}{
		"tid":           tweet.Tid,
		"uid":           tweet.Uid,
		"content":       tweet.Content,
		"media_urls":    string(mediaUrlsJson),
		"tags":          string(tagsJson),
		"is_public":     tweet.IsPublic,
		"created_at":    tweet.CreatedAt.Format(time.RFC3339),
		"is_deleted":    tweet.IsDeleted,
		"like_count":    tweet.LikeCount,
		"comment_count": tweet.CommentCount,
	}

	if err := s.CacheManager.HSetAll(ctx, "tweet", "info", tid, fields); err != nil {
		logx.Errorf("SetTweetToCache error, tid:%d, err:%v", tid, err)
		return err
	}

	if err := s.CacheManager.Expire(ctx, "tweet", "info", tid, 3600); err != nil {
		logx.Errorf("SetTweetToCache set expire error, tid:%d, err:%v", tid, err)
		return err
	}

	return nil
}

// GetTweetFromCache 从Hash缓存获取推文
func (s *ServiceContext) GetTweetFromCache(ctx context.Context, tid int64) (*model.Tweet, error) {
	fields, err := s.CacheManager.HGetAll(ctx, "tweet", "info", tid)
	if err != nil {
		logx.Errorf("GetTweetFromCache error, tid:%d, err:%v", tid, err)
		return nil, err
	}

	if len(fields) == 0 {
		return nil, redis.Nil
	}

	tweet := &model.Tweet{}

	if v, ok := fields["tid"]; ok {
		tweet.Tid, _ = strconv.ParseInt(v, 10, 64)
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
		tweet.CreatedAt, _ = time.Parse(time.RFC3339, v)
	}
	if v, ok := fields["is_deleted"]; ok {
		tweet.IsDeleted, _ = strconv.ParseBool(v)
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
func (s *ServiceContext) DelTweetCache(ctx context.Context, tid int64) {
	if err := s.CacheManager.Del(ctx, "tweet", "info", tid); err != nil {
		logx.Errorf("DelTweetCache error, tid:%d, err:%v", tid, err)
	}
}
