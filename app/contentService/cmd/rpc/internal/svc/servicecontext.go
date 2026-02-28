package svc

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"gozeroX/app/contentService/cmd/rpc/internal/config"
	"gozeroX/app/contentService/cmd/rpc/pb"
	"gozeroX/app/contentService/model"
	"gozeroX/pkg/cache"
)

type ServiceContext struct {
	Config             config.Config
	RedisClient        *redis.Redis
	CacheManager       *cache.Manager
	TweetModel         model.TweetModel
	TweetStatsLogModel model.TweetStatsLogModel
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
		Config:             c,
		RedisClient:        redisClient,
		CacheManager:       cacheManager,
		TweetModel:         model.NewTweetModel(sqlConn, c.Cache),
		TweetStatsLogModel: model.NewTweetStatsLogModel(sqlConn, c.Cache),
	}
}

// BuildTweet 构建推文返回
func (svcCtx *ServiceContext) BuildTweet(tweet *model.Tweet) *pb.Tweet {
	if tweet == nil {
		return nil
	}

	return &pb.Tweet{
		Tid:          tweet.Tid,
		Uid:          tweet.Uid,
		Content:      tweet.Content,
		MediaUrls:    tweet.MediaUrls,
		Tags:         tweet.Tags,
		IsPublic:     tweet.IsPublic,
		CreatedAt:    tweet.CreatedAt.Unix(),
		IsDeleted:    tweet.IsDeleted,
		LikeCount:    tweet.LikeCount,
		CommentCount: tweet.CommentCount,
	}
}
